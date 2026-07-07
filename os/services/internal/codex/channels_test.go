package codex

import (
	"context"
	"errors"
	"testing"

	"go.autonomous.ai/os/domain"
)

// Telegram is device-owned under Codex (os-server runs the getUpdates receive
// loop — see telegram_poll.go), so the channel API accepts telegram and
// refuses everything else.

func TestCodexSupportedChannels(t *testing.T) {
	got := (&CodexService{}).SupportedChannels()
	if len(got) != 1 || got[0] != domain.ChannelTelegram {
		t.Fatalf("SupportedChannels() = %v, want [telegram] (device-owned receive loop)", got)
	}
}

func TestCodexAddChannel(t *testing.T) {
	s := &CodexService{}
	if err := s.AddChannel(context.Background(), domain.AddChannelRequest{Channel: domain.ChannelTelegram}); err != nil {
		t.Errorf("AddChannel(telegram) err = %v, want nil (creds in config.json drive the device-owned loop)", err)
	}
	err := s.AddChannel(context.Background(), domain.AddChannelRequest{Channel: domain.ChannelSlack, SlackBotToken: "x"})
	if !errors.Is(err, domain.ErrChannelNotSupported) {
		t.Errorf("AddChannel(slack) err = %v, want ErrChannelNotSupported", err)
	}
}

func TestCodexRefreshChannelConfig(t *testing.T) {
	s := &CodexService{}
	out, err := s.RefreshChannelConfig(context.Background(), domain.RefreshChannelRequest{Channel: domain.ChannelTelegram})
	if err != nil || out != "" {
		t.Errorf(`RefreshChannelConfig(telegram) = (%q, %v), want ("", nil)`, out, err)
	}
	_, err = s.RefreshChannelConfig(context.Background(), domain.RefreshChannelRequest{Channel: domain.ChannelDiscord})
	if !errors.Is(err, domain.ErrChannelNotSupported) {
		t.Errorf("RefreshChannelConfig(discord) err = %v, want ErrChannelNotSupported", err)
	}
}
