package claudecode

import (
	"log/slog"

	"go.autonomous.ai/os/domain"
)

// CompactSession — Claude Code auto-compacts its own context when it
// approaches the window limit; there is no external compact RPC to call.
// Returns ErrNotSupportedByRuntime so callers see nothing was done here.
func (s *ClaudeCodeService) CompactSession(sessionKey string) error {
	slog.Info("CompactSession: not supported (claudecode auto-compacts)", "component", "claudecode", "session", sessionKey)
	return domain.ErrNotSupportedByRuntime
}

// rotateMaxTurns / rotateTokenThreshold gate session rotation (see
// ShouldRotateSession). Auto-compaction bounds the context SIZE but not
// persona fidelity: device-observed 2026-07-08, after ~18h / hundreds of
// sensing turns and several compaction cycles the one-line IDENTITY.md name
// had drifted out of the compacted context and the agent invented a name —
// CLAUDE.md @imports are only re-read at session start, so rotation is the
// re-anchor. Turn count is the primary trigger (compaction keeps reported
// tokens bounded, so a token threshold alone may never fire); the token
// gate is the codex-style safety net for a runaway thread.
const (
	rotateMaxTurns       = 80
	rotateTokenThreshold = 150_000
)

// ShouldRotateSession rotates on turn count (primary — persona re-anchor,
// see the constants above) or a token spike (safety net). The handler's
// NewSession path is instant (no compact freeze) and MEMORY.md/KNOWLEDGE.md
// long-term memory survives via the CLAUDE.md imports; only verbatim
// in-session conversation is lost.
func (s *ClaudeCodeService) ShouldRotateSession(totalTokens, turnsSinceRotation int) bool {
	return turnsSinceRotation >= rotateMaxTurns || totalTokens > rotateTokenThreshold
}

// NewSession asks the bridge to restart Claude Code WITHOUT --resume, starting a
// fresh session; the local session id is dropped so the init event of the new
// session is adopted.
func (s *ClaudeCodeService) NewSession(sessionKey string) error {
	slog.Info("NewSession: requesting fresh claude session", "component", "claudecode", "key", sessionKey)
	s.sessionUUID.Store("")
	if err := s.sendFrame(map[string]any{"type": "session.new"}); err != nil {
		// Not connected — the bridge never saw the request, so its session.json
		// still resumes the old session on the next spawn. Best-effort by design:
		// rotation is user-triggered and can simply be retried once the bridge
		// is back.
		slog.Warn("NewSession: bridge not reachable", "component", "claudecode", "error", err)
	}
	return nil
}
