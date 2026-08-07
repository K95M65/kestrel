package logger

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGELFHandlerDropsWhenBoundedQueueIsFull(t *testing.T) {
	started := make(chan struct{}, 1)
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started <- struct{}{}
		select {
		case <-release:
		case <-r.Context().Done():
		}
	}))
	defer server.Close()

	oldURL, oldUsername, oldPassword := gelfURL, gelfUsername, gelfPassword
	gelfURL, gelfUsername, gelfPassword = server.URL, "", ""
	defer func() { gelfURL, gelfUsername, gelfPassword = oldURL, oldUsername, oldPassword }()

	h := newGELFHandler(slog.LevelInfo, "test")
	defer h.sender.close()

	if err := h.Handle(context.Background(), slog.NewRecord(time.Now(), slog.LevelInfo, "first", 0)); err != nil {
		t.Fatalf("first Handle() error = %v", err)
	}
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("GELF worker did not start request")
	}

	for i := 0; i <= gelfQueueSize; i++ {
		if err := h.Handle(context.Background(), slog.NewRecord(time.Now(), slog.LevelInfo, "queued", 0)); err != nil {
			t.Fatalf("Handle() error = %v", err)
		}
	}
	if h.sender.dropped.Load() == 0 {
		t.Fatal("expected at least one GELF record to be dropped when queue is full")
	}

	start := time.Now()
	if err := h.Handle(context.Background(), slog.NewRecord(time.Now(), slog.LevelInfo, "overflow", 0)); err != nil {
		t.Fatalf("overflow Handle() error = %v", err)
	}
	if elapsed := time.Since(start); elapsed > 100*time.Millisecond {
		t.Fatalf("overflow Handle() blocked for %s", elapsed)
	}
	close(release)
}
