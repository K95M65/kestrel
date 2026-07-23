package migrateconfig

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type opencodeAdapter struct{}

func (opencodeAdapter) runtime() Runtime { return RuntimeOpenCode }

// OpenCode splits the LLM config across two presync-owned files: opencode.json
// (XDG: ~/.config/opencode) carries provider.campaign.options.baseURL, and
// /root/.opencode/.env carries the actual key (LLM_API_KEY=…; opencode.json
// only references it as "{env:LLM_API_KEY}"). Mirrors the codex adapter.
func (opencodeAdapter) read(opts Options) (LLMConfig, error) {
	return LLMConfig{
		APIKey:  readEnvVar(filepath.Join(opts.OpenCodeHome, ".env"), "LLM_API_KEY"),
		BaseURL: readOpenCodeBaseURL(opts.OpenCodeConfig),
	}, nil
}

func (opencodeAdapter) write(cfg LLMConfig, opts Options) error {
	if cfg.APIKey != "" {
		if err := writeEnvVar(filepath.Join(opts.OpenCodeHome, ".env"), "LLM_API_KEY", cfg.APIKey); err != nil {
			return err
		}
	}
	if cfg.BaseURL != "" {
		if err := writeOpenCodeBaseURL(opts.OpenCodeConfig, cfg.BaseURL); err != nil {
			return err
		}
	}
	return nil
}

// readOpenCodeBaseURL extracts provider.campaign.options.baseURL from
// opencode.json. Missing file/keys → "" (nothing to carry).
func readOpenCodeBaseURL(configJSON string) string {
	raw, err := os.ReadFile(configJSON)
	if err != nil {
		return ""
	}
	root := map[string]any{}
	if err := json.Unmarshal(raw, &root); err != nil {
		return ""
	}
	provider, _ := root["provider"].(map[string]any)
	campaign, _ := provider["campaign"].(map[string]any)
	options, _ := campaign["options"].(map[string]any)
	baseURL, _ := options["baseURL"].(string)
	return baseURL
}

// writeOpenCodeBaseURL patches provider.campaign.options.baseURL in opencode.json.
// A missing opencode.json is not an error — presync regenerates the whole head
// from config.json (already synced by the caller) on the next switch/boot.
func writeOpenCodeBaseURL(configJSON, baseURL string) error {
	raw, err := os.ReadFile(configJSON)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	root := map[string]any{}
	if err := json.Unmarshal(raw, &root); err != nil {
		return err
	}

	provider := ensureMap(root, "provider")
	campaign := ensureMap(provider, "campaign")
	options := ensureMap(campaign, "options")
	if options["baseURL"] == baseURL {
		return nil
	}
	options["baseURL"] = baseURL

	out, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return err
	}
	out = append(out, '\n')
	return atomicWrite(configJSON, out, 0o644)
}
