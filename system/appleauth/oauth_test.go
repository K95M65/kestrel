package appleauth

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) Do(r *http.Request) (*http.Response, error) { return f(r) }

func testP8(t *testing.T) string {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}))
}

func TestReadyHTTPS(t *testing.T) {
	c := New("com.kestrel.siwa", "TEAM", "KEY", testP8(t), "http://10.10.2.160/api/auth/apple/callback")
	if err := c.Ready(); err == nil {
		t.Fatal("http return url should fail")
	}
	c.ReturnURL = "https://kestrel.example/api/auth/apple/callback"
	if err := c.Ready(); err != nil {
		t.Fatal(err)
	}
}

func TestAuthorizeURL(t *testing.T) {
	c := New("com.kestrel.siwa", "TEAM", "KEY", testP8(t), "https://kestrel.example/cb")
	u, err := c.AuthorizeURL("st")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(u, "client_id=com.kestrel.siwa") || !strings.Contains(u, "state=st") {
		t.Fatalf("%s", u)
	}
}

func TestClientSecretJWT(t *testing.T) {
	c := New("com.kestrel.siwa", "TEAMID", "KEYID", testP8(t), "https://kestrel.example/cb")
	c.Now = func() time.Time { return time.Unix(1_700_000_000, 0) }
	tok, err := c.clientSecret()
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.Split(tok, ".")
	if len(parts) != 3 {
		t.Fatalf("parts=%d", len(parts))
	}
}

func TestExchange(t *testing.T) {
	c := New("com.kestrel.siwa", "TEAM", "KEY", testP8(t), "https://kestrel.example/cb")
	c.HTTP = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		body := `{"id_token":"aaa.eyJzdWIiOiIxIiwiZW1haWwiOiJhQGIifQ.sig"}`
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})
	id, err := c.Exchange("code")
	if err != nil {
		t.Fatal(err)
	}
	if id.Email != "a@b" || id.Sub != "1" {
		t.Fatalf("%+v", id)
	}
}
