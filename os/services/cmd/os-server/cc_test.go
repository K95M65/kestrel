package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"go.autonomous.ai/os/internal/claudecode"
)

func TestCcAgentRuntime(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	if got := ccAgentRuntime(path); got != "" {
		t.Fatalf("missing file: got %q, want empty", got)
	}
	if err := os.WriteFile(path, []byte(`{"agent_runtime":"codex"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := ccAgentRuntime(path); got != "codex" {
		t.Fatalf("got %q, want codex", got)
	}
}

func TestCcFilterFolder(t *testing.T) {
	sessions := []claudecode.CodingSessionInfo{
		{Folder: "/root/a", SessionID: "1"},
		{Folder: "/root/b", SessionID: "2"},
		{Folder: "/root/a", SessionID: "3"},
	}
	got := ccFilterFolder(sessions, "/root/a")
	if len(got) != 2 || got[0].SessionID != "1" || got[1].SessionID != "3" {
		t.Fatalf("unexpected filter result: %+v", got)
	}
}

func TestCcDedupeEnv(t *testing.T) {
	in := []string{"HOME=/home/user", "FOO=1", "HOME=/root", "BAR=x"}
	want := []string{"HOME=/root", "FOO=1", "BAR=x"}
	if got := ccDedupeEnv(in); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestCcEnvFilePairs(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	content := "# comment\nANTHROPIC_API_KEY=\"secret\"\n\nBAD LINE\nMODEL=Auto-AI\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	want := []string{"ANTHROPIC_API_KEY=secret", "MODEL=Auto-AI"}
	if got := ccEnvFilePairs(path); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}
