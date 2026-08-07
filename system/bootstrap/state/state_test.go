package state

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSaveWritesReadableStateWithoutTemporaryFiles(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "state.json")
	want := &State{Components: map[string]string{"os-server": "1.2.3"}}

	if err := Save(path, want); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got.Components["os-server"] != "1.2.3" {
		t.Fatalf("saved version = %q, want %q", got.Components["os-server"], "1.2.3")
	}
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != "state.json" {
		t.Fatalf("state directory entries = %v, want only state.json", entries)
	}
}

func TestLoadMovesInvalidStateAsideAndReturnsEmptyState(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	if err := os.WriteFile(path, []byte(`{"components":`), 0600); err != nil {
		t.Fatalf("write corrupt state: %v", err)
	}

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(got.Components) != 0 {
		t.Fatalf("recovered components = %v, want empty", got.Components)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}
	if len(entries) != 1 || !strings.HasPrefix(entries[0].Name(), "state.json.corrupt-") {
		t.Fatalf("state directory entries = %v, want corrupt-state backup", entries)
	}
	data, err := os.ReadFile(filepath.Join(dir, entries[0].Name()))
	if err != nil {
		t.Fatalf("read corrupt-state backup: %v", err)
	}
	if string(data) != `{"components":` {
		t.Fatalf("corrupt-state backup contents = %q, want original bytes", data)
	}
}
