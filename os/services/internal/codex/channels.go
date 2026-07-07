package codex

import (
	"context"
	"fmt"

	"go.autonomous.ai/os/domain"
)

// Telegram is DEVICE-OWNED under Codex. The Codex CLI has no channel layer of
// its own (unlike picoclaw, whose runtime binary polls the Telegram Bot API
// itself — see internal/picoclaw/presync.sh), so os-server runs the inbound
// getUpdates receive loop: telegram_poll.go, one goroutine started from
// StartWS. Because it lives inside this service's lifecycle it runs ONLY while
// codex is the active runtime — it can never fight another runtime's poller
// for getUpdates (Telegram 409s concurrent pollers; the hermes lesson).
// Slack / discord / whatsapp have no receive loop and stay unsupported.

// SupportedChannels — telegram only, via the device-owned receive loop
// (telegram_poll.go).
func (s *CodexService) SupportedChannels() []string {
	return []string{domain.ChannelTelegram}
}

// AddChannel — telegram is an honest no-op success: the receive loop reads
// config.TelegramBotToken / TelegramUserID fresh on every iteration, so the
// creds the caller just persisted to config.json are all it needs — there is
// nothing agent-side to write. Every other channel has no receive loop and
// returns domain.ErrChannelNotSupported.
func (s *CodexService) AddChannel(_ context.Context, data domain.AddChannelRequest) error {
	if data.EffectiveChannel() == domain.ChannelTelegram {
		return nil
	}
	return fmt.Errorf("codex: channel %q: %w", data.EffectiveChannel(), domain.ErrChannelNotSupported)
}

// RefreshChannelConfig — same rule: telegram creds are consumed live from
// Device config by the poll loop, so there is nothing to refresh and no
// derived value to return. Other channels are not supported.
func (s *CodexService) RefreshChannelConfig(_ context.Context, req domain.RefreshChannelRequest) (string, error) {
	if req.Channel == domain.ChannelTelegram {
		return "", nil
	}
	return "", fmt.Errorf("codex: channel %q: %w", req.Channel, domain.ErrChannelNotSupported)
}
