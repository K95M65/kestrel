package urlnorm

import "strings"

// NormalizeBaseURL ensures autonomous.ai base URLs include the /v1 OpenAI-compat
// prefix so all backends (TTS, STT, LLM, DL) receive a ready-to-use URL without
// each caller having to patch it individually. Non-autonomous URLs are left untouched.
func NormalizeBaseURL(base string) string {
	base = strings.TrimSuffix(strings.TrimSpace(base), "/")
	if strings.Contains(base, "campaign-api.autonomous.ai") && strings.HasSuffix(base, "/ai") {
		base += "/v1"
	}
	return base
}
