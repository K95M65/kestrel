package i18n

import "testing"

func TestBuildChitchatWakeWordsIncludesHelloAgentName(t *testing.T) {
	words := BuildChitchatWakeWords("Luna")
	want := []string{"wake up luna", "hello luna", "okay luna", "hey luna", "hi luna", "ok luna"}
	if len(words) != len(want) {
		t.Fatalf("BuildChitchatWakeWords() = %v, want %v", words, want)
	}
	for i, word := range want {
		if words[i] != word {
			t.Fatalf("BuildChitchatWakeWords() = %v, want %v", words, want)
		}
	}
}
