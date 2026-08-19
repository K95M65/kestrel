package domain

import "testing"

func TestListProvidersUniqueKeys(t *testing.T) {
	seen := map[string]bool{}
	for _, p := range ListProviders {
		if p.Key == "" || p.Name == "" {
			t.Fatalf("provider missing key/name: %+v", p)
		}
		if p.Auth == "" {
			t.Fatalf("%s: empty auth", p.Key)
		}
		if seen[p.Key] {
			t.Fatalf("duplicate provider key %q", p.Key)
		}
		seen[p.Key] = true
	}
	for _, need := range []string{"xai", "kimi", "cloudflare-workers-ai", "openai", "ollama"} {
		if !seen[need] {
			t.Fatalf("catalog missing %q", need)
		}
	}
}

func TestLookupAndExpand(t *testing.T) {
	p, ok := LookupLLMProvider("xai")
	if !ok || p.Auth != LLMAuthDeviceCode {
		t.Fatalf("xai lookup = %+v ok=%v", p, ok)
	}
	cf, ok := LookupLLMProvider("cloudflare-workers-ai")
	if !ok {
		t.Fatal("missing cloudflare-workers-ai")
	}
	got := ExpandProviderBaseURL(cf.BaseURL, map[string]string{"account_id": "abc123"})
	want := "https://api.cloudflare.com/client/v4/accounts/abc123/ai/v1"
	if got != want {
		t.Fatalf("expand = %q want %q", got, want)
	}
}
