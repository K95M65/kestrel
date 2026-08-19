package sensingmsg

import (
	"strings"
	"testing"
)

func TestBuildVoiceFollowupIsAnAuthorizedUserTurn(t *testing.T) {
	got := Build("voice_followup", "play music", "", "")
	if !strings.HasPrefix(got, "[user] [spoken] play music") {
		t.Fatalf("voice_followup = %q, want spoken user-priority message", got)
	}
	if strings.Contains(got, "[ambient]") {
		t.Fatalf("voice_followup must not be marked ambient: %q", got)
	}
}

func TestBuildVoiceCommandIsSpoken(t *testing.T) {
	got := Build("voice_command", "hey reachy, what time is it", "", "")
	if !strings.HasPrefix(got, "[user] [spoken] hey reachy, what time is it") {
		t.Fatalf("voice_command = %q, want [user] [spoken] prefix", got)
	}
}
