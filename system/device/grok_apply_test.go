package device

import "testing"

func TestIsXAIBase(t *testing.T) {
	if !isXAIBase("https://api.x.ai/v1") {
		t.Fatal("expected xAI base")
	}
	if isXAIBase("https://api.openai.com/v1") || isXAIBase("") {
		t.Fatal("non-xAI base should not match")
	}
}
