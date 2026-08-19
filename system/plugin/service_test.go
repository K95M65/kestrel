package plugin

import (
	"strings"
	"testing"
)

func TestValidPluginName(t *testing.T) {
	ok := []string{"dance", "asl-teacher", "cameraman", "a", "Foo_1.2"}
	for _, n := range ok {
		if !validPluginName(n) {
			t.Errorf("validPluginName(%q) = false", n)
		}
	}
	bad := []string{"", "..", "../etc", "foo/bar", "foo\\bar", "a b", "-sneaky", "has\nnewline"}
	for _, n := range bad {
		if validPluginName(n) {
			t.Errorf("validPluginName(%q) = true", n)
		}
	}
}

func TestPluginDirRejectsTraversal(t *testing.T) {
	for _, n := range []string{"..", "../..", "foo/bar", "/etc"} {
		if _, err := pluginDir(n); err == nil {
			t.Errorf("pluginDir(%q) succeeded", n)
		}
	}
	dir, err := pluginDir("dance")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(dir, "/dance") || strings.Contains(dir, "..") {
		t.Fatalf("pluginDir(dance) = %q", dir)
	}
}

func TestValidGitURL(t *testing.T) {
	ok := []string{
		"https://github.com/K95M65/kestrel.git",
		"http://example.com/p.git",
		"ssh://git@github.com/org/repo.git",
		"git@github.com:K95M65/kestrel.git",
	}
	for _, u := range ok {
		if err := validGitURL(u); err != nil {
			t.Errorf("validGitURL(%q) = %v", u, err)
		}
	}
	bad := []string{
		"",
		"--upload-pack=evil",
		"file:///etc/passwd",
		"javascript:alert(1)",
		"-uhttps://example.com/x.git",
		"git@github.com/no-colon",
	}
	for _, u := range bad {
		if err := validGitURL(u); err == nil {
			t.Errorf("validGitURL(%q) = nil", u)
		}
	}
}

func TestValidSubdirAndEntry(t *testing.T) {
	if err := validSubdir("integrations/apps/dance"); err != nil {
		t.Fatal(err)
	}
	for _, s := range []string{"../etc", "/abs", "foo/../bar", `foo\bar`} {
		if err := validSubdir(s); err == nil {
			t.Errorf("validSubdir(%q) = nil", s)
		}
	}
	if !validEntry("main.py") || !validEntry("src/app.py") {
		t.Fatal("expected valid entries")
	}
	for _, e := range []string{"-c", "../main.py", "/usr/bin/python", ""} {
		if validEntry(e) {
			t.Errorf("validEntry(%q) = true", e)
		}
	}
}

func TestGitCloneArgsUseDashDash(t *testing.T) {
	args := gitCloneArgs("--upload-pack=evil", "integrations/apps/dance", "/tmp/x")
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "filter.lfs.required=false") {
		t.Fatalf("missing lfs disable: %v", args)
	}
	var dash int
	for i, a := range args {
		if a == "--" {
			dash = i
			break
		}
	}
	if dash == 0 || args[dash+1] != "--upload-pack=evil" || args[dash+2] != "/tmp/x" {
		t.Fatalf("url not after -- : %v", args)
	}
}

func TestSystemdUnitOneshotIsSimple(t *testing.T) {
	body := systemdUnitBody("dance", "/var/lib/os-plugins/dance", "main.py", true)
	if strings.Contains(body, "Type=oneshot") {
		t.Fatal("oneshot plugins must not use Type=oneshot (blocks systemctl start)")
	}
	if !strings.Contains(body, "Type=simple") {
		t.Fatal("expected Type=simple")
	}
	if !strings.Contains(body, "Restart=no") {
		t.Fatal("expected Restart=no for finite apps")
	}
	if strings.Contains(body, "Restart=on-failure") {
		t.Fatal("finite apps must not restart")
	}
}

func TestSystemdUnitDaemonRestarts(t *testing.T) {
	body := systemdUnitBody("cameraman", "/var/lib/os-plugins/cameraman", "main.py", false)
	if !strings.Contains(body, "Restart=on-failure") {
		t.Fatal("daemons should restart on failure")
	}
}
