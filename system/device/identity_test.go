package device

import (
	"errors"
	"sync"
	"testing"
	"time"

	"go.autonomous.ai/os/system/domain"
	"go.autonomous.ai/os/system/server/config"
)

func TestNormalizeIdentityName(t *testing.T) {
	got, err := NormalizeIdentityName("  Buddy  ")
	if err != nil || got != "Buddy" {
		t.Fatalf("got %q %v, want Buddy", got, err)
	}
	if _, err := NormalizeIdentityName(""); err == nil {
		t.Fatal("empty name must fail")
	}
	if _, err := NormalizeIdentityName("ok!"); err == nil {
		t.Fatal("punctuation must fail")
	}
}

func TestNormalizeWakePhrase(t *testing.T) {
	got, err := NormalizeWakePhrase("  Hey Buddy  ")
	if err != nil || got != "hey buddy" {
		t.Fatalf("got %q %v", got, err)
	}
	got, err = NormalizeWakePhrase("   ")
	if err != nil || got != "" {
		t.Fatalf("blank = %q %v, want empty", got, err)
	}
	if _, err := NormalizeWakePhrase("x"); err == nil {
		t.Fatal("single char must fail")
	}
}

func TestExclusiveFromNameAndWake(t *testing.T) {
	if got := exclusiveFromNameAndWake("Buddy", ""); got != nil {
		t.Fatalf("empty wake exclusive = %v, want nil", got)
	}
	if got := exclusiveFromNameAndWake("Buddy", "hey buddy"); got != nil {
		t.Fatalf("default hey name exclusive = %v, want nil", got)
	}
	got := exclusiveFromNameAndWake("Buddy", "computer")
	if len(got) != 1 || got[0] != "computer" {
		t.Fatalf("custom exclusive = %v, want [computer]", got)
	}
}

// identityGW mirrors OpenClaw: NewSession's /new write returns immediately
// while IsBusy stays true until lifecycle_end.
type identityGW struct {
	domain.AgentGateway
	mu        sync.Mutex
	busy      bool
	newCalls  int
	sendCalls int
	name      string
}

func (g *identityGW) UpdateIdentityName(n string) error {
	g.mu.Lock()
	g.name = n
	g.mu.Unlock()
	return nil
}
func (g *identityGW) GetSessionKey() string { return "agent:main:main" }
func (g *identityGW) NewSession(string) error {
	g.mu.Lock()
	g.newCalls++
	g.busy = true
	g.mu.Unlock()
	return nil
}
func (g *identityGW) SendSystemChatMessage(string) (string, error) {
	g.mu.Lock()
	g.sendCalls++
	g.mu.Unlock()
	return "ok", nil
}
func (g *identityGW) IsBusy() bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.busy
}
func (g *identityGW) setBusy(v bool) {
	g.mu.Lock()
	g.busy = v
	g.mu.Unlock()
}
func (g *identityGW) counts() (newCalls, sendCalls int) {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.newCalls, g.sendCalls
}

func TestSetIdentityBlocksOwnerChatUntilSessionResetFinishes(t *testing.T) {
	t.Chdir(t.TempDir())
	prevWait, prevPoll := identityResetBusyWait, identityResetBusyPoll
	identityResetBusyWait = 2 * time.Second
	identityResetBusyPoll = 10 * time.Millisecond
	t.Cleanup(func() {
		identityResetBusyWait = prevWait
		identityResetBusyPoll = prevPoll
	})

	gw := &identityGW{}
	s := &Service{config: &config.Config{}, agentGateway: gw}

	if err := s.TryBeginOwnerChat(); err != nil {
		t.Fatalf("idle TryBeginOwnerChat: %v", err)
	}

	done := make(chan error, 1)
	go func() { done <- s.SetIdentity("Buddy", "") }()

	deadline := time.Now().Add(2 * time.Second)
	for {
		n, ping := gw.counts()
		if n >= 1 && s.SessionResetInFlight() {
			if ping != 0 {
				t.Fatal("name ping issued while /new still busy")
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("NewSession did not return with reset still in flight")
		}
		time.Sleep(5 * time.Millisecond)
	}

	if !gw.IsBusy() {
		t.Fatal("expected IsBusy after NewSession returned (OpenClaw write-then-busy)")
	}
	if err := s.WaitSessionReady(80 * time.Millisecond); !errors.Is(err, ErrSessionResetInFlight) {
		t.Fatalf("WaitSessionReady while /new busy = %v, want ErrSessionResetInFlight", err)
	}
	if err := s.TryBeginOwnerChat(); !errors.Is(err, ErrSessionResetInFlight) {
		t.Fatalf("TryBeginOwnerChat while /new busy = %v, want ErrSessionResetInFlight", err)
	}

	gw.setBusy(false)
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("SetIdentity: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("SetIdentity did not return after gateway went idle")
	}

	if s.SessionResetInFlight() {
		t.Fatal("reset still marked in flight after SetIdentity returned")
	}
	if err := s.WaitSessionReady(80 * time.Millisecond); err != nil {
		t.Fatalf("WaitSessionReady after idle: %v", err)
	}
	if err := s.TryBeginOwnerChat(); err != nil {
		t.Fatalf("TryBeginOwnerChat after idle: %v", err)
	}
	n, ping := gw.counts()
	if n != 1 {
		t.Fatalf("NewSession calls = %d, want 1", n)
	}
	if ping != 1 {
		t.Fatalf("rename ping SendSystemChatMessage calls = %d, want 1", ping)
	}
	gw.mu.Lock()
	gotName := gw.name
	gw.mu.Unlock()
	if gotName != "Buddy" {
		t.Fatalf("UpdateIdentityName = %q, want Buddy", gotName)
	}
}
