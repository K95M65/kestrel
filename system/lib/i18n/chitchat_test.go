package i18n

import "testing"

func TestBuildChitchatWakeWordsIncludesHelloAgentName(t *testing.T) {
	words := BuildChitchatWakeWords("Luna")
	if len(words) == 0 || words[0] != "hello luna" {
		t.Fatalf("BuildChitchatWakeWords() = %v, want hello luna first", words)
	}
}
