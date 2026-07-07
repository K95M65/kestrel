package codex

import (
	"context"
	"errors"
	"testing"

	"go.autonomous.ai/os/domain"
)

// Channels are device-owned under Codex: telegram via the getUpdates receive
// loop (telegram_poll.go) and slack via the HTTP-mode proxy path (slack.go).
// The channel API accepts those two and refuses everything else.

func TestCodexSupportedChannels(t *testing.T) {
	got := (&CodexService{}).SupportedChannels()
	if len(got) != 2 || got[0] != domain.ChannelTelegram || got[1] != domain.ChannelSlack {
		t.Fatalf("SupportedChannels() = %v, want [telegram slack] (device-owned paths)", got)
	}
}

func TestCodexAddChannel(t *testing.T) {
	s := &CodexService{}
	if err := s.AddChannel(context.Background(), domain.AddChannelRequest{Channel: domain.ChannelTelegram}); err != nil {
		t.Errorf("AddChannel(telegram) err = %v, want nil (creds in config.json drive the device-owned loop)", err)
	}
	if err := s.AddChannel(context.Background(), domain.AddChannelRequest{Channel: domain.ChannelSlack, SlackBotToken: "x"}); err != nil {
		t.Errorf("AddChannel(slack) err = %v, want nil (creds in config.json drive the bridge)", err)
	}
	err := s.AddChannel(context.Background(), domain.AddChannelRequest{Channel: domain.ChannelDiscord, DiscordBotToken: "x"})
	if !errors.Is(err, domain.ErrChannelNotSupported) {
		t.Errorf("AddChannel(discord) err = %v, want ErrChannelNotSupported", err)
	}
}

func TestCodexRefreshChannelConfig(t *testing.T) {
	s := &CodexService{}
	out, err := s.RefreshChannelConfig(context.Background(), domain.RefreshChannelRequest{Channel: domain.ChannelTelegram})
	if err != nil || out != "" {
		t.Errorf(`RefreshChannelConfig(telegram) = (%q, %v), want ("", nil)`, out, err)
	}
	out, err = s.RefreshChannelConfig(context.Background(), domain.RefreshChannelRequest{Channel: domain.ChannelSlack})
	if err != nil || out != "" {
		t.Errorf(`RefreshChannelConfig(slack) = (%q, %v), want ("", nil)`, out, err)
	}
	_, err = s.RefreshChannelConfig(context.Background(), domain.RefreshChannelRequest{Channel: domain.ChannelDiscord})
	if !errors.Is(err, domain.ErrChannelNotSupported) {
		t.Errorf("RefreshChannelConfig(discord) err = %v, want ErrChannelNotSupported", err)
	}
}
