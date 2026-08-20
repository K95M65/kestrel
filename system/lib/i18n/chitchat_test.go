package i18n

import "testing"

func TestBuildChitchatWakeWordsPreservesLocalAttentionAliases(t *testing.T) {
	words := BuildChitchatWakeWords("Luna")
	want := []string{"hello luna", "hey luna", "này luna", "ê luna", "luna ơi", "luna"}
	if len(words) != len(want) {
		t.Fatalf("BuildChitchatWakeWords() = %v, want %v", words, want)
	}
	for i, word := range want {
		if words[i] != word {
			t.Fatalf("BuildChitchatWakeWords() = %v, want %v", words, want)
		}
	}
}

func TestBuildVoiceWakeWordsUsesHALPrefixes(t *testing.T) {
	words := BuildVoiceWakeWords("Luna")
	want := []string{"wake up luna", "hello luna", "okay luna", "hey luna", "hi luna", "alo luna", "ok luna"}
	if len(words) != len(want) {
		t.Fatalf("BuildVoiceWakeWords() = %v, want %v", words, want)
	}
	for i, word := range want {
		if words[i] != word {
			t.Fatalf("BuildVoiceWakeWords() = %v, want %v", words, want)
		}
	}
}

func TestVoiceWakeWordsForNamePrefersExclusiveList(t *testing.T) {
	SetExclusiveWakeWords([]string{"computer"})
	t.Cleanup(func() { SetExclusiveWakeWords(nil) })
	got := VoiceWakeWordsForName("Luna")
	if len(got) != 1 || got[0] != "computer" {
		t.Fatalf("VoiceWakeWordsForName exclusive = %v, want [computer]", got)
	}
}

func TestSetDeviceNamePreservesDisplayCasing(t *testing.T) {
	SetDeviceName("McBot")
	t.Cleanup(func() { SetDeviceName("autonomous") })
	if DeviceName() != "mcbot" {
		t.Fatalf("DeviceName = %q, want mcbot", DeviceName())
	}
	if DeviceDisplayName() != "McBot" {
		t.Fatalf("DeviceDisplayName = %q, want McBot", DeviceDisplayName())
	}
}

func TestBuildSupportedVoiceWakeWordsIncludesCurrentAndPermanentAliases(t *testing.T) {
	words := BuildSupportedVoiceWakeWords("Luna", "lamp")
	want := []string{
		"wake up autonomous", "hello autonomous", "okay autonomous", "hey autonomous", "hi autonomous", "alo autonomous", "ok autonomous",
		"wake up lamp", "hello lamp", "okay lamp", "hey lamp", "hi lamp", "alo lamp", "ok lamp",
		"wake up luna", "hello luna", "okay luna", "hey luna", "hi luna", "alo luna", "ok luna",
	}
	if len(words) != len(want) {
		t.Fatalf("BuildSupportedVoiceWakeWords() = %v, want %v", words, want)
	}
	for i, word := range want {
		if words[i] != word {
			t.Fatalf("BuildSupportedVoiceWakeWords() = %v, want %v", words, want)
		}
	}
}
