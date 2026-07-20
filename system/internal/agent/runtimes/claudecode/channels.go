package claudecode

import (
	"context"
	"fmt"

	"go.autonomous.ai/os/domain"
)

// SupportedChannels — all three channels are DEVICE-OWNED (mirroring
// internal/agent/runtimes/codex):
//
// telegram: os-server long-polls getUpdates and injects accepted DMs as chat
// turns (telegram_poll.go, started from StartWS).
//
// discord: os-server owns a discordgo gateway session (discord.go, started
// from StartWS) — Discord has no long-poll receive API, so a bot WS session
// replaces the getUpdates loop.
//
// Claude Code's native telegram/discord channel plugins are deliberately NOT
// used: they proved undebuggable in the field (bun children, no journal logs,
// silent allowlist drops, silent death on bridge-restart races), and they
// would compete with the device-owned loops for the same bot (Telegram 409s
// concurrent pollers; Discord would double-reply). presync removes their
// state and no longer writes CLAUDECODE_CHANNELS.
//
// slack is DEVICE-OWNED (HTTP mode): Claude Code has no slack channel plugin —
// "Claude in Slack" is a separate cloud feature that spawns web sessions from
// @Claude mentions, not a device channel. Instead the bff-campaign-service
// proxy fans Slack events out over MQTT to the device's slack_event handler,
// which type-asserts the active gateway to domain.SlackBridge — implemented in
// slack.go (mirror of internal/agent/runtimes/codex/slack.go), so events route here only
// while claudecode is active. Replies go back via chat.postMessage
// (config.SlackBotToken). whatsapp is OpenClaw-only (Baileys plugin).
func (s *ClaudeCodeService) SupportedChannels() []string {
	return []string{domain.ChannelTelegram, domain.ChannelSlack, domain.ChannelDiscord}
}

// AddChannel — all supported channels are honest no-op successes: the
// device-owned loops read creds fresh from Device config on each use (the
// telegram poll loop per iteration, the discord bot per (re)connect attempt,
// the Slack bridge per event/Web API call), so the creds the caller just
// persisted (persist-then-apply) are all that is needed — nothing agent-side
// to write, no bridge restart. Slack HTTP mode's signing secret is consumed
// by the public proxy, not on the device (the authenticated MQTT path is
// trusted, same as codex/hermes). Unsupported channels return
// domain.ErrChannelNotSupported.
//
// Discord nuance (same as codex): the bot loop rechecks the token only while
// DISCONNECTED (every discordNoTokenWait / discordErrorWait). Rotating the
// token while a session is open takes effect on the next session cycle
// (gateway restart / runtime switch); saving the token the first time is
// picked up within 30 s.
func (s *ClaudeCodeService) AddChannel(_ context.Context, data domain.AddChannelRequest) error {
	channel := data.EffectiveChannel()
	if !domain.ChannelSupported(s, channel) {
		return fmt.Errorf("claudecode: channel %q: %w", channel, domain.ErrChannelNotSupported)
	}
	return nil // creds are read live from Device config (telegram_poll.go / discord.go / slack.go)
}

// RefreshChannelConfig — same capability rule and the same no-op contract as
// AddChannel. Returns "" for the runtime version string (the active version
// surfaces via Version()).
func (s *ClaudeCodeService) RefreshChannelConfig(_ context.Context, req domain.RefreshChannelRequest) (string, error) {
	if !domain.ChannelSupported(s, req.Channel) {
		return "", fmt.Errorf("claudecode: channel %q: %w", req.Channel, domain.ErrChannelNotSupported)
	}
	return "", nil // nothing to refresh — creds are consumed live (see AddChannel)
}
