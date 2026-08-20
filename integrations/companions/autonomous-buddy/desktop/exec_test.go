package main

import (
	"testing"
)

func TestParseHTTPURL(t *testing.T) {
	if _, err := parseHTTPURL("https://gmail.com"); err != nil {
		t.Fatal(err)
	}
	if _, err := parseHTTPURL("http://10.10.2.160/setting"); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{"", "javascript:alert(1)", "file:///etc/passwd", "ftp://x", "gmail.com"} {
		if _, err := parseHTTPURL(bad); err == nil {
			t.Fatalf("expected reject %q", bad)
		}
	}
}

func TestPing(t *testing.T) {
	out, err := dispatch("ping", nil)
	if err != nil {
		t.Fatal(err)
	}
	if out["pong"] != true {
		t.Fatalf("ping = %#v", out)
	}
	if out["os"] == "" {
		t.Fatal("os empty")
	}
}

func TestMissingParams(t *testing.T) {
	if _, err := dispatch("open_url", nil); err == nil {
		t.Fatal("open_url empty")
	}
	if _, err := dispatch("open_app", nil); err == nil {
		t.Fatal("open_app empty")
	}
	if _, err := dispatch("notification", nil); err == nil {
		t.Fatal("notification empty")
	}
}

func TestUnknownAction(t *testing.T) {
	_, err := dispatch("explode", nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestFingerprintStable(t *testing.T) {
	a, b := fingerprint(), fingerprint()
	if a != b || len(a) != 16 {
		t.Fatalf("fingerprint %q %q", a, b)
	}
}
