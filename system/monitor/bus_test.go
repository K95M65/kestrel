package monitor

import (
	"testing"
	"time"

	"go.autonomous.ai/os/system/domain"
)

func TestUnsubscribeReturnsAndRemovesSubscriber(t *testing.T) {
	bus := ProvideBus()
	ch, unsub := bus.Subscribe()

	done := make(chan struct{})
	go func() {
		unsub()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("unsubscribe did not return")
	}

	bus.Push(domain.MonitorEvent{Type: "test"})
	select {
	case evt := <-ch:
		t.Fatalf("unsubscribed channel received event: %+v", evt)
	case <-time.After(50 * time.Millisecond):
	}
}
