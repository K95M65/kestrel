package codex

import (
	"context"
	"fmt"

	"go.autonomous.ai/os/domain"
)

// Channels are DEVICE-OWNED under Codex. The Codex CLI has no channel layer
// of its own (unlike picoclaw, whose runtime binary polls the Telegram Bot API
// itself — see internal/picoclaw/presync.sh), so os-server runs both inbound
// paths:
//   - telegram: the getUpdates receive loop (telegram_poll.go, one goroutine
//     started from StartWS — it lives inside this service's lifecycle, so it
//     runs ONLY while codex is the active runtime and can never fight another
//     runtime's poller; Telegram 409s concurrent pollers, the hermes lesson).
//   - slack (HTTP mode): the bff-campaign-service proxy fans Slack events out
//     over MQTT to the device's slack_event handler, which type-asserts the
//     active gateway to domain.SlackBridge — implemented in slack.go, so
//     events route here only while codex is active. Replies go back via
//     chat.postMessage (config.SlackBotToken).
//
// Discord / whatsapp have no receive path and stay unsupported.

// SupportedChannels — telegram (device-owned receive loop, telegram_poll.go)
// and slack (HTTP-mode proxy path, slack.go).
func (s *CodexService) SupportedChannels() []string {
	return []string{domain.ChannelTelegram, domain.ChannelSlack}
}

// AddChannel — telegram and slack are honest no-op successes: every consumer
// reads the creds fresh from Device config on each use (the receive loop reads
// config.TelegramBotToken / TelegramUserID per iteration; the Slack bridge
// reads config.SlackUserID per event and config.SlackBotToken per Web API
// call), so the creds the caller just persisted to config.json are all that
// is needed — there is nothing agent-side to write. Slack HTTP mode's signing
// secret is consumed by the public proxy, not on the device (same as hermes'
// bridge: the authenticated MQTT path is trusted). Every other channel has no
// receive path and returns domain.ErrChannelNotSupported.
func (s *CodexService) AddChannel(_ context.Context, data domain.AddChannelRequest) error {
	channel := data.EffectiveChannel()
	if !domain.ChannelSupported(s, channel) {
		return fmt.Errorf("codex: channel %q: %w", channel, domain.ErrChannelNotSupported)
	}
	return nil
}

// RefreshChannelConfig — same rule: telegram/slack creds are consumed live
// from Device config, so there is nothing to refresh and no derived value to
// return. Other channels are not supported.
func (s *CodexService) RefreshChannelConfig(_ context.Context, req domain.RefreshChannelRequest) (string, error) {
	if !domain.ChannelSupported(s, req.Channel) {
		return "", fmt.Errorf("codex: channel %q: %w", req.Channel, domain.ErrChannelNotSupported)
	}
	return "", nil
}
