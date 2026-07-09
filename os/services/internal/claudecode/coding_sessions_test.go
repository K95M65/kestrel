package claudecode

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// writeTranscript writes a minimal claude JSONL transcript (a summary line + a
// user line carrying cwd) under projectsDir/<projName>/<uuid>.jsonl and stamps
// its mtime so recency ordering is deterministic.
func writeTranscript(t *testing.T, projectsDir, projName, uuid, cwd, summary, userText string, mtime time.Time) {
	t.Helper()
	dir := filepath.Join(projectsDir, projName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	path := filepath.Join(dir, uuid+".jsonl")
	body := ""
	if summary != "" {
		body += `{"type":"summary","summary":` + jsonStr(summary) + `}` + "\n"
	}
	body += `{"type":"user","cwd":` + jsonStr(cwd) + `,"message":{"role":"user","content":` + jsonStr(userText) + `}}` + "\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write transcript: %v", err)
	}
	if err := os.Chtimes(path, mtime, mtime); err != nil {
		t.Fatalf("chtimes: %v", err)
	}
}

func jsonStr(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

func TestAllCodingSessionsAndFolders(t *testing.T) {
	dir := t.TempDir()
	now := time.Now()
	// Two folders; -root has two sessions (older + newest).
	writeTranscript(t, dir, "-root-test", "aaaa1111-0000-0000-0000-000000000001", "/root/test", "Caro game", "make caro game", now.Add(-2*time.Hour))
	writeTranscript(t, dir, "-root", "bbbb2222-0000-0000-0000-000000000002", "/root", "old", "old work", now.Add(-3*time.Hour))
	writeTranscript(t, dir, "-root", "cccc3333-0000-0000-0000-000000000003", "/root", "new", "new work", now.Add(-30*time.Minute))

	s := &ClaudeCodeService{claudeProjectsDirPath: dir}

	all := s.allCodingSessions()
	if len(all) != 3 {
		t.Fatalf("allCodingSessions = %d, want 3", len(all))
	}
	// newest-first ordering
	if all[0].SessionID != "cccc3333-0000-0000-0000-000000000003" {
		t.Errorf("newest session = %s, want cccc3333…", all[0].SessionID)
	}
	if all[0].Folder != "/root" || all[0].Summary != "new" {
		t.Errorf("meta wrong: folder=%q summary=%q", all[0].Folder, all[0].Summary)
	}

	// codingFolders collapses to newest-per-folder: /root (cccc) + /root/test.
	folders := s.codingFolders()
	if len(folders) != 2 {
		t.Fatalf("codingFolders = %d, want 2", len(folders))
	}
	if folders[0].Folder != "/root" || folders[0].SessionID != "cccc3333-0000-0000-0000-000000000003" {
		t.Errorf("folders[0] = %+v, want /root newest cccc", folders[0])
	}

	// folderSessions(/root) returns both, newest first.
	rootSessions := s.folderSessions("/root")
	if len(rootSessions) != 2 || rootSessions[0].SessionID != "cccc3333-0000-0000-0000-000000000003" {
		t.Fatalf("folderSessions(/root) = %+v", rootSessions)
	}

	latest, ok := s.latestSessionForFolder("/root/test")
	if !ok || latest.SessionID != "aaaa1111-0000-0000-0000-000000000001" {
		t.Errorf("latestSessionForFolder(/root/test) = %+v ok=%v", latest, ok)
	}
	if _, ok := s.latestSessionForFolder("/root/nope"); ok {
		t.Errorf("latestSessionForFolder(/root/nope) should be absent")
	}
}

func TestReadTranscriptMetaFallbackToFirstUser(t *testing.T) {
	dir := t.TempDir()
	// No summary line → summary falls back to first user text.
	writeTranscript(t, dir, "-root-app", "dddd4444-0000-0000-0000-000000000004", "/root/app", "", "fix login bug\nmany lines", time.Now())
	s := &ClaudeCodeService{claudeProjectsDirPath: dir}
	sessions := s.folderSessions("/root/app")
	if len(sessions) != 1 {
		t.Fatalf("want 1 session, got %d", len(sessions))
	}
	if sessions[0].Summary != "fix login bug many lines" { // oneLine collapses the newline
		t.Errorf("summary = %q, want collapsed first-user text", sessions[0].Summary)
	}
}

func TestNormalizeFolder(t *testing.T) {
	cases := map[string]string{
		"/root/test":     "/root/test",
		"/root/test/":    "/root/test",
		"test":           "/root/test",
		"~/proj":         "/root/proj",
		"~":              "/root",
		`"/root/a b"`:    "/root/a b",
		"/root/./x/../y": "/root/y",
	}
	for in, want := range cases {
		if got := normalizeFolder(in); got != want {
			t.Errorf("normalizeFolder(%q) = %q, want %q", in, got, want)
		}
	}
	if got := normalizeFolder("   "); got != "" {
		t.Errorf("normalizeFolder(blank) = %q, want empty", got)
	}
}
