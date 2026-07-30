package http

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// seedFile writes a file under dir and returns its path.
func seedFile(t *testing.T, dir, name string, size int) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, make([]byte, size), 0644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestResolveAgentFileServesAllowedFile(t *testing.T) {
	root := t.TempDir()
	img := seedFile(t, root, "media/snap.jpg", 8)

	got, ct, err := resolveAgentFile(img, []string{filepath.Join(root, "media")})
	if err != nil {
		t.Fatalf("want served, got %v", err)
	}
	// The returned path is the RESOLVED one — on macOS t.TempDir() lives under
	// /var, itself a symlink to /private/var.
	resolved, _ := filepath.EvalSymlinks(img)
	if got != resolved {
		t.Errorf("path = %q, want %q", got, resolved)
	}
	if ct != "image/jpeg" {
		t.Errorf("content type = %q", ct)
	}
}

// The extension whitelist is checked before the filesystem, so a type we don't
// serve is refused whether or not it exists.
func TestResolveAgentFileRejectsUnservedTypes(t *testing.T) {
	root := t.TempDir()
	roots := []string{root}

	for _, name := range []string{"openclaw.json", "agent.log", "id_rsa", "run.sh", "archive.zip"} {
		p := seedFile(t, root, name, 4)
		if _, _, err := resolveAgentFile(p, roots); !errors.Is(err, errFileType) {
			t.Errorf("%s: err = %v, want errFileType", name, err)
		}
	}
	// …and for a path that doesn't exist either, so the error can't be used to
	// probe for the file's presence.
	if _, _, err := resolveAgentFile(filepath.Join(root, "absent.json"), roots); !errors.Is(err, errFileType) {
		t.Errorf("absent .json: err = %v, want errFileType", err)
	}
}

// `..` must not walk out of an allow-listed root.
func TestResolveAgentFileRejectsTraversal(t *testing.T) {
	base := t.TempDir()
	allowed := filepath.Join(base, "media")
	secret := seedFile(t, base, "secret.txt", 4)
	seedFile(t, allowed, "keep.txt", 4)

	escape := filepath.Join(allowed, "..", "secret.txt")
	if _, _, err := resolveAgentFile(escape, []string{allowed}); !errors.Is(err, errOutsideRoots) {
		t.Fatalf("traversal err = %v, want errOutsideRoots", err)
	}
	// Sanity: the same file IS readable when its own dir is a root, so the test
	// above failed for the right reason.
	if _, _, err := resolveAgentFile(secret, []string{base}); err != nil {
		t.Fatalf("control case failed: %v", err)
	}
}

// A symlink INSIDE a root pointing out of it is the classic /tmp attack.
func TestResolveAgentFileRejectsSymlinkEscape(t *testing.T) {
	base := t.TempDir()
	allowed := filepath.Join(base, "media")
	outside := seedFile(t, base, "outside.txt", 4)
	if err := os.MkdirAll(allowed, 0755); err != nil {
		t.Fatal(err)
	}

	link := filepath.Join(allowed, "innocent.txt")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, _, err := resolveAgentFile(link, []string{allowed}); !errors.Is(err, errOutsideRoots) {
		t.Errorf("symlink escape err = %v, want errOutsideRoots", err)
	}
}

// A sibling whose name merely starts with the root's must not pass as being
// under it.
func TestResolveAgentFileRejectsRootPrefixLookalike(t *testing.T) {
	base := t.TempDir()
	allowed := filepath.Join(base, "media")
	if err := os.MkdirAll(allowed, 0755); err != nil {
		t.Fatal(err)
	}
	evil := seedFile(t, base, "media-evil/snap.jpg", 4)

	if _, _, err := resolveAgentFile(evil, []string{allowed}); !errors.Is(err, errOutsideRoots) {
		t.Errorf("lookalike root err = %v, want errOutsideRoots", err)
	}
}

func TestResolveAgentFileRejectsDirsRelativeAndOversized(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "shots.jpg"), 0755); err != nil {
		t.Fatal(err)
	}
	roots := []string{root}

	// A directory that happens to carry a served extension is not a file.
	if _, _, err := resolveAgentFile(filepath.Join(root, "shots.jpg"), roots); !errors.Is(err, errFileNotFound) {
		t.Errorf("directory err = %v, want errFileNotFound", err)
	}
	// Relative paths never reach the filesystem.
	if _, _, err := resolveAgentFile("media/snap.jpg", roots); !errors.Is(err, errFileNotFound) {
		t.Errorf("relative err = %v, want errFileNotFound", err)
	}
	if _, _, err := resolveAgentFile("", roots); !errors.Is(err, errFileNotFound) {
		t.Errorf("empty err = %v, want errFileNotFound", err)
	}
	// Over the size cap is refused rather than streamed.
	big := seedFile(t, root, "big.jpg", agentFileMaxBytes+1)
	if _, _, err := resolveAgentFile(big, roots); !errors.Is(err, errFileNotFound) {
		t.Errorf("oversized err = %v, want errFileNotFound", err)
	}
}

// A root that doesn't exist on this device must not make everything pass.
func TestResolveAgentFileMissingRootAllowsNothing(t *testing.T) {
	root := t.TempDir()
	img := seedFile(t, root, "snap.jpg", 4)

	if _, _, err := resolveAgentFile(img, []string{filepath.Join(root, "nope")}); !errors.Is(err, errOutsideRoots) {
		t.Errorf("err = %v, want errOutsideRoots", err)
	}
}

// The shipped roots must cover the snapshot dir HAL writes and /tmp, and must
// NOT cover a runtime's config dir (openclaw.json holds gateway tokens).
func TestDefaultAgentFileRootsScope(t *testing.T) {
	roots := defaultAgentFileRoots()
	has := func(want string) bool {
		for _, r := range roots {
			if r == want {
				return true
			}
		}
		return false
	}

	for _, want := range []string{"/root/.openclaw/media", "/root/.openclaw/workspace", "/tmp"} {
		if !has(want) {
			t.Errorf("missing root %q", want)
		}
	}
	for _, unwanted := range []string{"/root/.openclaw", "/root", "/"} {
		if has(unwanted) {
			t.Errorf("root %q must not be served", unwanted)
		}
	}
}
