package urlnorm_test

import (
	"testing"

	"go.autonomous.ai/os/lib/urlnorm"
)

func TestNormalizeBaseURL(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "autonomous.ai ending in /ai gets /v1 appended",
			input: "https://campaign-api.autonomous.ai/api/v1/ai",
			want:  "https://campaign-api.autonomous.ai/api/v1/ai/v1",
		},
		{
			name:  "already normalized url is left untouched",
			input: "https://campaign-api.autonomous.ai/api/v1/ai/v1",
			want:  "https://campaign-api.autonomous.ai/api/v1/ai/v1",
		},
		{
			// claudecode/presync.sh strips /v1 before writing ANTHROPIC_BASE_URL;
			// reading it back during a runtime switch must re-normalize.
			name:  "stripped url from claudecode presync is re-normalized",
			input: "https://campaign-api.autonomous.ai/api/v1/ai",
			want:  "https://campaign-api.autonomous.ai/api/v1/ai/v1",
		},
		{
			name:  "trailing slash is trimmed before check",
			input: "https://campaign-api.autonomous.ai/api/v1/ai/",
			want:  "https://campaign-api.autonomous.ai/api/v1/ai/v1",
		},
		{
			name:  "non-autonomous url is left untouched",
			input: "https://openai.com/v1",
			want:  "https://openai.com/v1",
		},
		{
			name:  "empty string is left untouched",
			input: "",
			want:  "",
		},
		{
			name:  "autonomous url not ending in /ai is left untouched",
			input: "https://campaign-api.autonomous.ai/api/v1/ai/v1/extra",
			want:  "https://campaign-api.autonomous.ai/api/v1/ai/v1/extra",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := urlnorm.NormalizeBaseURL(tc.input)
			if got != tc.want {
				t.Errorf("NormalizeBaseURL(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}
