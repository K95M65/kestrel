package config

import (
	"os"
	"path/filepath"
	"testing"
)

// TestWithLockSave_NormalizesBaseURLs verifies that WithLockSave always
// normalizes LLMBaseURL/STTBaseURL/TTSBaseURL before writing to disk, so a
// caller that forgets to call urlnorm.NormalizeBaseURL (e.g. a new agent
// presync that strips /v1 from ANTHROPIC_BASE_URL then writes it back) cannot
// persist a de-normalized URL to config.json.
func TestWithLockSave_NormalizesBaseURLs(t *testing.T) {
	dir := t.TempDir()
	origPath := configPath
	configPath = filepath.Join(dir, "config.json")
	defer func() { configPath = origPath }()

	c := &Config{}

	// Simulate what claudecode/presync.sh does: strips /v1 from llm_base_url
	// before writing ANTHROPIC_BASE_URL. A migration that reads it back and
	// saves it without normalizing would land the stripped URL on disk.
	stripped := "https://campaign-api.autonomous.ai/api/v1/ai"
	normalized := "https://campaign-api.autonomous.ai/api/v1/ai/v1"

	if err := c.WithLockSave(func(c *Config) {
		c.LLMBaseURL = stripped
		c.STTBaseURL = stripped
		c.TTSBaseURL = stripped
	}); err != nil {
		t.Fatalf("WithLockSave: %v", err)
	}

	if c.LLMBaseURL != normalized {
		t.Errorf("LLMBaseURL = %q, want %q", c.LLMBaseURL, normalized)
	}
	if c.STTBaseURL != normalized {
		t.Errorf("STTBaseURL = %q, want %q", c.STTBaseURL, normalized)
	}
	if c.TTSBaseURL != normalized {
		t.Errorf("TTSBaseURL = %q, want %q", c.TTSBaseURL, normalized)
	}

	// Also verify the on-disk JSON reflects the normalized URL.
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config.json: %v", err)
	}
	content := string(data)
	if !contains(content, normalized) {
		t.Errorf("config.json does not contain normalized URL %q:\n%s", normalized, content)
	}
	if contains(content, `"llm_base_url": "`+stripped+`"`) {
		t.Errorf("config.json still contains stripped URL %q", stripped)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsStr(s, sub))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
