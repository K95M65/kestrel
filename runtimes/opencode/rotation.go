package opencode

import (
	"log/slog"

	"go.autonomous.ai/os/system/domain"
)

// CompactSession — OpenCode does not expose a compact API; rotate via
// NewSession instead.
func (s *OpenCodeService) CompactSession(sessionKey string) error {
	slog.Info("CompactSession: not supported (opencode backend)", "component", "opencode", "session", sessionKey)
	return domain.ErrNotSupportedByRuntime
}

// opencodeFallbackTokenThreshold is a safety net only: OpenCode auto-compacts its
// own context, so the reported per-turn input stays bounded and this rarely
// fires. It exists so a runaway session (compaction bug, oversized tool outputs)
// still gets rotated.
const opencodeFallbackTokenThreshold = 150_000

// ShouldRotateSession rotates on real reported token count (see
// domain.AgentGateway). turn.completed usage maps input+cached+output into
// TotalTokens (translator.go), which approximates the live context size.
func (s *OpenCodeService) ShouldRotateSession(totalTokens, _ int) bool {
	return totalTokens > opencodeFallbackTokenThreshold
}

// NewSession tells the bridge to drop the persisted session id (session.new
// frame) so the next `opencode run` starts a fresh session, and clears the local
// session key. Best-effort when the socket is down: the local clear still
// happens and the bridge's stale session id will fail the --session resume → the
// bridge retries fresh on its own.
func (s *OpenCodeService) NewSession(sessionKey string) error {
	slog.Info("NewSession: requesting fresh opencode session", "component", "opencode", "key", sessionKey)
	s.sessionUUID.Store("")
	if err := s.sendFrame(map[string]any{"type": "session.new"}); err != nil {
		slog.Warn("session.new frame send failed (bridge will retry fresh on resume failure)",
			"component", "opencode", "error", err)
	}
	return nil
}
