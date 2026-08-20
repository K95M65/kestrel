package device

import (
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"go.autonomous.ai/os/system/lib/hal"
	"go.autonomous.ai/os/system/lib/i18n"
	"go.autonomous.ai/os/system/server/config"
)

// ErrSessionResetInFlight is returned when Talk/web_chat would overlap a
// rename's NewSession. Sensing waits or queues instead of sending a turn that
// OpenClaw would drop as busy.
var ErrSessionResetInFlight = errors.New("session reset in flight")

var (
	identityNameRe = regexp.MustCompile(`^[\p{L}\p{N}][\p{L}\p{N} '\-]{0,31}$`)
	identityWakeRe = regexp.MustCompile(`^[\p{L}\p{N}][\p{L}\p{N} '\-]{1,47}$`)
)

// NormalizeIdentityName trims and collapses whitespace. 1–32 letters, numbers,
// spaces, hyphens, or apostrophes.
func NormalizeIdentityName(name string) (string, error) {
	n := strings.Join(strings.Fields(strings.TrimSpace(name)), " ")
	if n == "" {
		return "", fmt.Errorf("name is required")
	}
	if !identityNameRe.MatchString(n) {
		return "", fmt.Errorf("name must be 1–32 letters, numbers, spaces, or hyphens")
	}
	return n, nil
}

// NormalizeWakePhrase lowercases and collapses whitespace. Empty is valid
// (generated aliases). Otherwise 2–48 letters, numbers, spaces, or hyphens.
func NormalizeWakePhrase(phrase string) (string, error) {
	p := strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(phrase))), " ")
	if p == "" {
		return "", nil
	}
	if !identityWakeRe.MatchString(p) {
		return "", fmt.Errorf("wake phrase must be 2–48 letters, numbers, or spaces")
	}
	return p, nil
}

// exclusiveFromNameAndWake returns the exclusive HAL list. A blank phrase, or
// the default "hey {name}", means generated aliases — not exclusive.
func exclusiveFromNameAndWake(name, wake string) []string {
	if wake == "" {
		return nil
	}
	if wake == "hey "+strings.ToLower(name) {
		return nil
	}
	return []string{wake}
}

func sameStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// SetIdentity writes the agent name into IDENTITY.md, optionally an exclusive
// wake phrase into config.json wake_words, and pushes the live HAL list.
// Exclusive vs generated is decided at HAL VoiceService construction, so a
// change to the exclusive list restarts HAL. A name-only change live-pushes.
func (s *Service) SetIdentity(name, wakePhrase string) error {
	n, err := NormalizeIdentityName(name)
	if err != nil {
		return err
	}
	wake, err := NormalizeWakePhrase(wakePhrase)
	if err != nil {
		return err
	}
	exclusive := exclusiveFromNameAndWake(n, wake)

	if s.agentGateway != nil {
		if err := s.agentGateway.UpdateIdentityName(n); err != nil {
			return fmt.Errorf("save name: %w", err)
		}
	}

	prevExclusive := append([]string(nil), s.config.WakeWords...)
	exclusiveChanged := !sameStrings(prevExclusive, exclusive)
	enable := true
	if err := s.config.WithLockSave(func(c *config.Config) {
		c.WakeWords = exclusive
		if c.WakeWord == nil || !*c.WakeWord {
			c.WakeWord = &enable
		}
	}); err != nil {
		return fmt.Errorf("save identity: %w", err)
	}

	i18n.SetDeviceName(n)
	i18n.SetExclusiveWakeWords(exclusive)

	words := i18n.VoiceWakeWordsForName(n)
	slog.Info("identity updated", "component", "device", "name", n, "exclusive", exclusive, "words", words)

	if exclusiveChanged {
		s.restartHAL("identity wake words")
	} else if len(words) > 0 {
		hal.SetVoiceConfig(words)
	}

	// Drop the in-session chat so the next Talk turn re-reads IDENTITY.md.
	// Without this, Guided Setup's "what's your name?" still answers as the
	// previous name until the conversation happens to rotate.
	s.resetAgentSession("identity name change", n)
	return nil
}

// How long resetAgentSession waits for OpenClaw to finish /new (WS write
// returns immediately; IsBusy stays true until lifecycle_end, ~3s).
var identityResetBusyWait = 10 * time.Second
var identityResetBusyPoll = 40 * time.Millisecond

func (s *Service) setSessionResetInFlight(v bool) {
	s.sessionResetMu.Lock()
	s.sessionResetInFlight = v
	s.sessionResetMu.Unlock()
}

// SessionResetInFlight is true while a rename's NewSession (and the follow-up
// name ping) is still running. Talk must not send until this is false.
func (s *Service) SessionResetInFlight() bool {
	s.sessionResetMu.Lock()
	defer s.sessionResetMu.Unlock()
	return s.sessionResetInFlight
}

// TryBeginOwnerChat is the gate for Talk / guided-setup chat. It refuses while
// a rename reset is in flight so the next turn is not dropped as busy.
func (s *Service) TryBeginOwnerChat() error {
	if s.SessionResetInFlight() {
		return ErrSessionResetInFlight
	}
	return nil
}

// WaitSessionReady blocks until a rename reset finishes or the timeout hits.
func (s *Service) WaitSessionReady(d time.Duration) error {
	deadline := time.Now().Add(d)
	for s.SessionResetInFlight() {
		if time.Now().After(deadline) {
			return ErrSessionResetInFlight
		}
		time.Sleep(identityResetBusyPoll)
	}
	return nil
}

func (s *Service) waitGatewayIdle(d time.Duration) {
	if s.agentGateway == nil {
		return
	}
	deadline := time.Now().Add(d)
	for s.agentGateway.IsBusy() {
		if time.Now().After(deadline) {
			slog.Warn("gateway still busy after identity session reset", "component", "device")
			return
		}
		time.Sleep(identityResetBusyPoll)
	}
}

// resetAgentSession asks the gateway to start a fresh conversation, then tells
// the agent its new name. OpenClaw NewSession is a fire-and-forget /new write:
// sendChat returns while IsBusy stays true until lifecycle_end. The in-flight
// flag stays set through that idle wait so Talk/web_chat is not queued-as-busy.
func (s *Service) resetAgentSession(reason, name string) {
	if s.agentGateway == nil {
		return
	}
	s.setSessionResetInFlight(true)
	defer s.setSessionResetInFlight(false)

	key := s.agentGateway.GetSessionKey()
	if err := s.agentGateway.NewSession(key); err != nil {
		slog.Warn("NewSession after identity change failed", "component", "device", "reason", reason, "error", err)
		return
	}
	slog.Info("agent session reset", "component", "device", "reason", reason, "session", key)
	s.waitGatewayIdle(identityResetBusyWait)

	prompt := fmt.Sprintf("[system] Your name is now \"%s\" (IDENTITY.md). Introduce yourself with that name only.", name)
	if _, err := s.agentGateway.SendSystemChatMessage(prompt); err != nil {
		slog.Warn("rename greeting failed", "component", "device", "error", err)
		return
	}
	slog.Info("rename greeting sent", "component", "device", "name", name)
	s.waitGatewayIdle(identityResetBusyWait)
}
