package googleauth

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) Do(r *http.Request) (*http.Response, error) { return f(r) }

func TestRequestDeviceCode(t *testing.T) {
	c := New("client.apps.googleusercontent.com", "secret")
	c.HTTP = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.String() != DeviceAuthorizationURL {
			t.Fatalf("url = %s", r.URL)
		}
		body := `{"device_code":"dc","user_code":"ABCD-EFGH","verification_url":"https://www.google.com/device","expires_in":1800,"interval":5}`
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})
	dc, err := c.RequestDeviceCode()
	if err != nil {
		t.Fatal(err)
	}
	if dc.UserCode != "ABCD-EFGH" || dc.VerificationURI != "https://www.google.com/device" {
		t.Fatalf("%+v", dc)
	}
}

func TestExchangePending(t *testing.T) {
	c := New("client.apps.googleusercontent.com", "secret")
	c.HTTP = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		body := `{"error":"authorization_pending"}`
		return &http.Response{StatusCode: 400, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})
	_, pending, err := c.Exchange(DeviceCode{DeviceCode: "dc"})
	if err != nil {
		t.Fatal(err)
	}
	if pending != "authorization_pending" {
		t.Fatalf("pending = %q", pending)
	}
}

func TestExchangeDenied(t *testing.T) {
	c := New("id", "s")
	c.HTTP = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		body := `{"error":"access_denied"}`
		return &http.Response{StatusCode: 400, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})
	_, _, err := c.Exchange(DeviceCode{DeviceCode: "dc"})
	if !TerminalDeviceError(err) {
		t.Fatalf("err = %v", err)
	}
}

func TestNewRequiresClient(t *testing.T) {
	c := New("", "")
	if _, err := c.RequestDeviceCode(); err == nil {
		t.Fatal("empty client id must fail")
	}
}
