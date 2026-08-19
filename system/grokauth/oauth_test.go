package grokauth

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestRefreshRotatesRefreshToken(t *testing.T) {
	var gotGrant string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		gotGrant = r.Form.Get("grant_type")
		if r.Form.Get("client_id") != ClientID {
			t.Errorf("client_id = %q", r.Form.Get("client_id"))
		}
		if r.Form.Get("refresh_token") != "old-refresh" {
			t.Errorf("refresh_token = %q", r.Form.Get("refresh_token"))
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token":  "new-access",
			"refresh_token": "new-refresh",
			"expires_in":    21600,
			"token_type":    "Bearer",
		})
	}))
	defer srv.Close()

	c := &Client{HTTP: srv.Client(), TokenURL: srv.URL}
	tok, err := c.Refresh("old-refresh")
	if err != nil {
		t.Fatal(err)
	}
	if gotGrant != "refresh_token" {
		t.Fatalf("grant_type = %q", gotGrant)
	}
	if tok.AccessToken != "new-access" || tok.RefreshToken != "new-refresh" {
		t.Fatalf("tokens = %+v", tok)
	}
	if tok.ExpiresIn != 6*time.Hour {
		t.Fatalf("ExpiresIn = %s", tok.ExpiresIn)
	}
}

func TestExchangePending(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "authorization_pending"})
	}))
	defer srv.Close()
	c := &Client{HTTP: srv.Client(), TokenURL: srv.URL}
	_, pending, err := c.Exchange(DeviceCode{DeviceCode: "dev-1"})
	if err != nil {
		t.Fatal(err)
	}
	if pending != "authorization_pending" {
		t.Fatalf("pending = %q", pending)
	}
}

func TestRefreshEmpty(t *testing.T) {
	if _, err := New().Refresh("  "); err == nil {
		t.Fatal("expected error")
	}
}

func TestRequestDeviceCode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		if r.Form.Get("scope") != Scope {
			t.Errorf("scope = %q", r.Form.Get("scope"))
		}
		if r.Form.Get("referrer") != Referrer {
			t.Errorf("referrer = %q", r.Form.Get("referrer"))
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"device_code":               "dev-1",
			"user_code":                 "ABCD-1234",
			"verification_uri":          "https://auth.x.ai/device",
			"verification_uri_complete": "https://auth.x.ai/device?user_code=ABCD-1234",
			"expires_in":                600,
			"interval":                  5,
		})
	}))
	defer srv.Close()

	c := &Client{HTTP: srv.Client(), DeviceURL: srv.URL}
	dc, err := c.RequestDeviceCode()
	if err != nil {
		t.Fatal(err)
	}
	if dc.UserCode != "ABCD-1234" || dc.DeviceCode != "dev-1" {
		t.Fatalf("device = %+v", dc)
	}
	if dc.ExpiresIn != 10*time.Minute || dc.Interval != 5*time.Second {
		t.Fatalf("timing = %+v", dc)
	}
}

func TestPollPendingThenSuccess(t *testing.T) {
	var n atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		i := n.Add(1)
		if i == 1 {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "authorization_pending"})
			return
		}
		if i == 2 {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "slow_down"})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token":  "access",
			"refresh_token": "refresh",
			"expires_in":    3600,
		})
	}))
	defer srv.Close()

	var slept []time.Duration
	c := &Client{
		HTTP:     srv.Client(),
		TokenURL: srv.URL,
		Now:      func() time.Time { return time.Unix(1_700_000_000, 0) },
		Sleep:    func(d time.Duration) { slept = append(slept, d) },
	}
	tok, err := c.Poll(DeviceCode{
		DeviceCode: "dev-1",
		ExpiresIn:  time.Minute,
		Interval:   time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	if tok.AccessToken != "access" || tok.RefreshToken != "refresh" {
		t.Fatalf("tokens = %+v", tok)
	}
	if n.Load() != 3 {
		t.Fatalf("calls = %d", n.Load())
	}
	if len(slept) != 2 {
		t.Fatalf("sleeps = %v", slept)
	}
	if slept[1] <= slept[0] {
		t.Fatalf("slow_down should increase interval: %v", slept)
	}
}

func TestPollDenied(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"error":"access_denied"}`)
	}))
	defer srv.Close()
	c := &Client{
		HTTP:     srv.Client(),
		TokenURL: srv.URL,
		Now:      func() time.Time { return time.Unix(1_700_000_000, 0) },
		Sleep:    func(time.Duration) {},
	}
	_, err := c.Poll(DeviceCode{DeviceCode: "dev-1", ExpiresIn: time.Minute, Interval: time.Second})
	if err == nil || !strings.Contains(err.Error(), "denied") {
		t.Fatalf("err = %v", err)
	}
}

func TestAccessTokenIsExpiring(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	makeJWT := func(exp int64) string {
		payload, _ := json.Marshal(map[string]int64{"exp": exp})
		return "eyJhbGciOiJub25lIn0." + base64.RawURLEncoding.EncodeToString(payload) + ".x"
	}
	if !AccessTokenIsExpiring(makeJWT(now.Add(30*time.Second).Unix()), 2*time.Minute, now) {
		t.Fatal("token inside skew should be expiring")
	}
	if AccessTokenIsExpiring(makeJWT(now.Add(10*time.Minute).Unix()), 2*time.Minute, now) {
		t.Fatal("fresh token should not be expiring")
	}
	if AccessTokenIsExpiring("not-a-jwt", 2*time.Minute, now) {
		t.Fatal("opaque token should not look expiring")
	}
}

func TestPositiveSecondsGarbageFallsBack(t *testing.T) {
	if got := positiveSeconds("NaN", 5*time.Second); got != 5*time.Second {
		t.Fatalf("got %s", got)
	}
	if got := positiveSeconds(-5.0, 5*time.Second); got != 5*time.Second {
		t.Fatalf("got %s", got)
	}
	if got := positiveSeconds(2.5, time.Second); got != 2500*time.Millisecond {
		t.Fatalf("got %s", got)
	}
}
