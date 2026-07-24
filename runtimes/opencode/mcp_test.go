package opencode

import (
	"encoding/json"
	"testing"
)

// TestToOpenCodeMCPEntryHTTP: the canonical OpenClaw-shaped hosted-MCP entry
// ({type:"http", url, headers}) maps to opencode's remote shape
// ({type:"remote", url, headers, enabled:true}).
func TestToOpenCodeMCPEntryHTTP(t *testing.T) {
	entry := map[string]any{
		"type":    "http",
		"url":     "https://mcp.notion.com/mcp",
		"headers": map[string]any{"Authorization": "Bearer tok"},
	}
	got := toOpenCodeMCPEntry(entry)
	if got["type"] != "remote" {
		t.Fatalf("type must be remote, got %v", got)
	}
	if got["url"] != "https://mcp.notion.com/mcp" {
		t.Fatalf("url not preserved: %v", got)
	}
	if got["enabled"] != true {
		t.Fatalf("enabled must default true, got %v", got)
	}
	hh, ok := got["headers"].(map[string]any)
	if !ok || hh["Authorization"] != "Bearer tok" {
		t.Fatalf("headers not mapped: %v", got)
	}
}

// TestToOpenCodeMCPEntryStdio: stdio entries become local, command+args merged
// into one array and env renamed to environment.
func TestToOpenCodeMCPEntryStdio(t *testing.T) {
	entry := map[string]any{
		"command": "npx",
		"args":    []any{"-y", "some-mcp"},
		"env":     map[string]any{"KEY": "v"},
	}
	got := toOpenCodeMCPEntry(entry)
	if got["type"] != "local" {
		t.Fatalf("type must be local, got %v", got)
	}
	cmd, ok := got["command"].([]any)
	if !ok || len(cmd) != 3 || cmd[0] != "npx" || cmd[1] != "-y" || cmd[2] != "some-mcp" {
		t.Fatalf("command must merge command+args, got %v", got["command"])
	}
	if _, ok := got["environment"]; !ok {
		t.Fatalf("env must be renamed to environment, got %v", got)
	}
	if got["enabled"] != true {
		t.Fatalf("enabled must default true, got %v", got)
	}
}

// TestOpenCodeMCPEntryJSONRoundTrip: the mapped entry survives an opencode.json
// marshal/unmarshal cycle under "mcp".<name>, and the provider/model head is
// preserved — the same path WriteMCPEntry drives.
func TestOpenCodeMCPEntryJSONRoundTrip(t *testing.T) {
	cfg := map[string]any{
		"model": "campaign/Auto-AI",
		"provider": map[string]any{
			"campaign": map[string]any{"name": "Autonomous campaign-api"},
		},
	}
	servers := ensureOpenCodeMap(cfg, "mcp")
	servers["notion"] = toOpenCodeMCPEntry(map[string]any{
		"type":    "http",
		"url":     "https://mcp.notion.com/mcp",
		"headers": map[string]any{"Authorization": "Bearer tok"},
	})

	out, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	back := map[string]any{}
	if err := json.Unmarshal(out, &back); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	srv, _ := back["mcp"].(map[string]any)
	notion, _ := srv["notion"].(map[string]any)
	if notion == nil || notion["url"] != "https://mcp.notion.com/mcp" {
		t.Fatalf("round trip lost entry: %s", out)
	}
	if back["model"] != "campaign/Auto-AI" {
		t.Fatalf("round trip lost head keys: %s", out)
	}
	if _, ok := back["provider"].(map[string]any); !ok {
		t.Fatalf("round trip lost provider head: %s", out)
	}
}
