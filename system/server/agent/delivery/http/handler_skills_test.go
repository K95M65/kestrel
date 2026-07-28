package http

import (
	"archive/zip"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// writeZip builds a zip at path from name→content pairs.
func writeZip(t *testing.T, path string, entries map[string]string) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create zip: %v", err)
	}
	defer f.Close()

	w := zip.NewWriter(f)
	for name, content := range entries {
		e, err := w.Create(name)
		if err != nil {
			t.Fatalf("create entry %s: %v", name, err)
		}
		if _, err := e.Write([]byte(content)); err != nil {
			t.Fatalf("write entry %s: %v", name, err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}
}

func TestExtractSkillBundle(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "skill.zip")
	writeZip(t, zipPath, map[string]string{
		"my-skill/SKILL.md":            "# My Skill\n",
		"my-skill/reference/notes.md":  "notes\n",
		"my-skill/assets/icon.png":     "\x89PNG\x00\x01\x02binary",
		"my-skill/.DS_Store":           "junk",
		"__MACOSX/my-skill/._SKILL.md": "junk",
	})

	bundle, err := extractSkillBundle(zipPath, filepath.Join(dir, "unpacked"))
	if err != nil {
		t.Fatalf("extract: %v", err)
	}

	byPath := map[string]bool{}
	for _, f := range bundle.Files {
		byPath[f.Path] = true
	}
	if len(bundle.Files) != 3 {
		t.Fatalf("want 3 files (cruft filtered), got %d: %v", len(bundle.Files), byPath)
	}
	if byPath[".DS_Store"] || byPath["my-skill/.DS_Store"] {
		t.Error(".DS_Store must be filtered out")
	}

	for _, f := range bundle.Files {
		switch f.Path {
		case "my-skill/SKILL.md":
			if f.Text != "# My Skill\n" || f.Binary {
				t.Errorf("SKILL.md: text=%q binary=%v", f.Text, f.Binary)
			}
		case "my-skill/assets/icon.png":
			if !f.Binary || f.Text != "" {
				t.Errorf("icon.png must be reported as binary with no text, got binary=%v text=%q", f.Binary, f.Text)
			}
		}
	}

	// The entries must actually land on disk — the endpoint's contract is
	// "download to temp, unzip there, then read".
	if _, err := os.Stat(filepath.Join(dir, "unpacked", "my-skill", "SKILL.md")); err != nil {
		t.Errorf("expected SKILL.md unpacked on disk: %v", err)
	}
}

func TestExtractSkillBundleRejectsTraversal(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "evil.zip")
	writeZip(t, zipPath, map[string]string{"../escaped.md": "pwned"})

	if _, err := extractSkillBundle(zipPath, filepath.Join(dir, "unpacked")); err == nil {
		t.Fatal("expected traversal entry to be rejected")
	}
	if _, err := os.Stat(filepath.Join(dir, "escaped.md")); !os.IsNotExist(err) {
		t.Error("traversal entry escaped the unpack dir")
	}
}

func TestExtractSkillBundleEmptyArchive(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "empty.zip")
	writeZip(t, zipPath, map[string]string{})

	if _, err := extractSkillBundle(zipPath, filepath.Join(dir, "unpacked")); err == nil {
		t.Fatal("expected an empty archive to be an error")
	}
}

func TestBrowseSkills(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var gotPath, gotQuery, gotLocation string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotQuery, gotLocation = r.URL.Path, r.URL.RawQuery, r.Header.Get("location")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":1,"data":{"data":[{"id":"abc","name":"design-critique","version":"1.0.43"}],"total":1}}`))
	}))
	defer srv.Close()
	t.Setenv("SKILL_STORE_BASE_URL", srv.URL)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/agent/skills/browse?keyword=design&status=7", nil)

	(&AgentHandler{}).BrowseSkills(c)

	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if gotPath != "/api/v1/agent-skills" {
		t.Errorf("upstream path = %q", gotPath)
	}
	if gotLocation != "en-US" {
		t.Errorf("location header = %q, want en-US", gotLocation)
	}
	if !strings.Contains(gotQuery, "keyword=design") {
		t.Errorf("keyword not forwarded: %q", gotQuery)
	}
	// `status` can't distinguish unset from 0 upstream, so it is never forwarded.
	if strings.Contains(gotQuery, "status=") {
		t.Errorf("status must not be forwarded: %q", gotQuery)
	}

	var resp struct {
		Status int `json:"status"`
		Data   struct {
			Data []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"data"`
			Total int64 `json:"total"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Status != 1 || len(resp.Data.Data) != 1 || resp.Data.Data[0].Name != "design-critique" {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}

// A catalog business failure arrives as HTTP 200 with a non-1 status — the
// proxy must surface it as an error, not as an empty success.
func TestBrowseSkillsUpstreamBusinessError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"status":4001,"message":"catalog unavailable"}`))
	}))
	defer srv.Close()
	t.Setenv("SKILL_STORE_BASE_URL", srv.URL)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/agent/skills/browse", nil)

	(&AgentHandler{}).BrowseSkills(c)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("want 502, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "catalog unavailable") {
		t.Errorf("upstream message not surfaced: %s", rec.Body.String())
	}
}

func TestSkillBundle(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Build the archive the fake catalog will serve from /download.
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "skill.zip")
	writeZip(t, zipPath, map[string]string{"design-critique/SKILL.md": "# Design Critique\n"})
	archive, err := os.ReadFile(zipPath)
	if err != nil {
		t.Fatalf("read zip: %v", err)
	}

	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/zip")
		_, _ = w.Write(archive)
	}))
	defer srv.Close()
	t.Setenv("SKILL_STORE_BASE_URL", srv.URL)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/agent/skills/bundle?id=abc123", nil)

	(&AgentHandler{}).SkillBundle(c)

	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if gotPath != "/api/v1/agent-skills/abc123/download" {
		t.Errorf("upstream path = %q", gotPath)
	}

	var resp struct {
		Data struct {
			ID    string `json:"id"`
			Files []struct {
				Path string `json:"path"`
				Text string `json:"text"`
			} `json:"files"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Data.ID != "abc123" {
		t.Errorf("id = %q", resp.Data.ID)
	}
	if len(resp.Data.Files) != 1 || resp.Data.Files[0].Path != "design-critique/SKILL.md" {
		t.Fatalf("unexpected files: %s", rec.Body.String())
	}
	if resp.Data.Files[0].Text != "# Design Critique\n" {
		t.Errorf("text = %q", resp.Data.Files[0].Text)
	}
}

func TestSkillBundleRejectsBadID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	for _, id := range []string{"", "../secret", "a/b"} {
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		c.Request = httptest.NewRequest(http.MethodGet, "/api/agent/skills/bundle?id="+id, nil)

		(&AgentHandler{}).SkillBundle(c)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("id %q: want 400, got %d", id, rec.Code)
		}
	}
}
