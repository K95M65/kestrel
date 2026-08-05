package device

import (
	"sync/atomic"
	"testing"
	"time"
)

type testReadiness struct {
	ready atomic.Bool
}

func (r *testReadiness) IsReady() bool { return r.ready.Load() }

func TestWaitForAgentReadyRequiresStableWindow(t *testing.T) {
	r := &testReadiness{}
	r.ready.Store(true)

	// A short outage must reset the stability clock rather than letting an
	// earlier ready observation satisfy the gate.
	go func() {
		time.Sleep(12 * time.Millisecond)
		r.ready.Store(false)
		time.Sleep(12 * time.Millisecond)
		r.ready.Store(true)
	}()

	started := time.Now()
	if !waitForAgentReady(r, 250*time.Millisecond, 30*time.Millisecond, 2*time.Millisecond) {
		t.Fatal("waitForAgentReady returned false")
	}
	if elapsed := time.Since(started); elapsed < 45*time.Millisecond {
		t.Fatalf("returned after %v; want to wait through the readiness reset", elapsed)
	}
}

func TestWaitForAgentReadyTimesOutWhenNeverReady(t *testing.T) {
	r := &testReadiness{}
	if waitForAgentReady(r, 20*time.Millisecond, 0, 2*time.Millisecond) {
		t.Fatal("waitForAgentReady returned true for an unready gateway")
	}
}
