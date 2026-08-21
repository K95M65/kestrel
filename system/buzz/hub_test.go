package buzz

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestBroadcast(t *testing.T) {
	h := NewHub()
	got := make(chan Note, 1)
	h.SetHandler(func(n Note) { got <- n })
	// handler only fires from readPump; Broadcast does not call it.
	h.Broadcast(Note{Type: "note", Text: "hi", From: "a"})
	if h.Count() != 0 {
		t.Fatalf("count=%d", h.Count())
	}
	select {
	case <-got:
		t.Fatal("broadcast should not invoke handler")
	case <-time.After(20 * time.Millisecond):
	}
}

func TestHubRoundTrip(t *testing.T) {
	h := NewHub()
	got := make(chan Note, 1)
	h.SetHandler(func(n Note) { got <- n })
	srv := httptest.NewServer(http.HandlerFunc(h.ServeWS))
	defer srv.Close()

	cli := &Client{URL: srv.URL, Device: "lima", Name: "Lima"}
	stop := make(chan struct{})
	defer close(stop)
	go cli.Start(stop)

	deadline := time.Now().Add(2 * time.Second)
	for !cli.Connected() {
		if time.Now().After(deadline) {
			t.Fatal("client never connected")
		}
		time.Sleep(20 * time.Millisecond)
	}
	if err := cli.Say("hello hive"); err != nil {
		t.Fatal(err)
	}
	select {
	case n := <-got:
		if n.Text != "hello hive" || n.From != "lima" {
			t.Fatalf("%+v", n)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("hub did not see note")
	}
	if h.Count() < 1 {
		t.Fatalf("peers=%d", h.Count())
	}
}
