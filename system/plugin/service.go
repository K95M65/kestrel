package plugin

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"go.autonomous.ai/os/system/domain"
)

const (
	pluginsDir = "/var/lib/os-plugins"
	unitPrefix = "os-plugin-"
	systemdDir = "/etc/systemd/system"
)

var pluginNameRe = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]{0,62}$`)

// manifest is the parsed plugin.json.
type manifest struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
	Entry       string `json:"entry"`
	Oneshot     bool   `json:"oneshot"`
}

type installJob struct {
	URL    string
	Subdir string
	Name   string
	Status string
	Err    string
}

type Service struct {
	jobs sync.Map // installKey → installJob
}

func ProvideService() *Service {
	return &Service{}
}

func installKey(url, subdir string) string {
	return url + "\n" + subdir
}

// Install clones a git repo, sets up a venv, and creates a systemd unit.
func (s *Service) Install(url string) (*domain.Plugin, error) {
	return s.InstallFrom(domain.PluginInstallRequest{URL: url})
}

// InstallFrom clones url (optionally one subdir) and installs the plugin.
func (s *Service) InstallFrom(req domain.PluginInstallRequest) (*domain.Plugin, error) {
	rawURL := strings.TrimSpace(req.URL)
	subdir := strings.Trim(strings.TrimSpace(req.Subdir), "/")
	if err := validGitURL(rawURL); err != nil {
		return nil, err
	}
	if err := validSubdir(subdir); err != nil {
		return nil, err
	}

	key := installKey(rawURL, subdir)
	job := installJob{URL: rawURL, Subdir: subdir, Status: "installing"}
	if _, loaded := s.jobs.LoadOrStore(key, job); loaded {
		return nil, fmt.Errorf("install already in progress")
	}
	fail := func(err error) (*domain.Plugin, error) {
		job.Status = "failed"
		job.Err = err.Error()
		s.jobs.Store(key, job)
		return nil, err
	}

	// Clone to a temp dir first, then read plugin.json to get the name.
	tmpDir, err := os.MkdirTemp("", "os-plugin-clone-*")
	if err != nil {
		return fail(fmt.Errorf("create temp dir: %w", err))
	}
	defer os.RemoveAll(tmpDir)

	slog.Info("[plugins] cloning", "component", "plugin", "url", rawURL, "subdir", subdir)
	if err := cloneRepo(rawURL, subdir, tmpDir); err != nil {
		return fail(err)
	}

	src := tmpDir
	if subdir != "" {
		src = filepath.Join(tmpDir, subdir)
	}

	// Parse plugin.json.
	m, err := readManifest(src)
	if err != nil {
		return fail(fmt.Errorf("read plugin.json: %w", err))
	}
	if !validPluginName(m.Name) {
		return fail(fmt.Errorf("plugin.json: invalid name"))
	}
	if m.Entry == "" {
		m.Entry = "main.py"
	}
	if !validEntry(m.Entry) {
		return fail(fmt.Errorf("plugin.json: invalid entry"))
	}
	job.Name = m.Name
	s.jobs.Store(key, job)

	// Ensure plugins dir exists.
	if err := os.MkdirAll(pluginsDir, 0o755); err != nil {
		return fail(fmt.Errorf("create plugins dir: %w", err))
	}

	dest, err := pluginDir(m.Name)
	if err != nil {
		return fail(err)
	}
	if _, err := os.Stat(dest); err == nil {
		return fail(fmt.Errorf("plugin %q already installed", m.Name))
	}

	// Move the plugin tree (whole clone, or just the subdir) into place.
	if out, err := exec.Command("mv", src, dest).CombinedOutput(); err != nil {
		return fail(fmt.Errorf("move plugin: %s: %w", strings.TrimSpace(string(out)), err))
	}

	// Create Python venv.
	slog.Info("[plugins] creating venv", "component", "plugin", "name", m.Name)
	if out, err := exec.Command("python3", "-m", "venv", filepath.Join(dest, ".venv")).CombinedOutput(); err != nil {
		os.RemoveAll(dest)
		return fail(fmt.Errorf("create venv: %s: %w", strings.TrimSpace(string(out)), err))
	}

	// Install requirements if present.
	reqFile := filepath.Join(dest, "requirements.txt")
	if _, err := os.Stat(reqFile); err == nil {
		slog.Info("[plugins] installing requirements", "component", "plugin", "name", m.Name)
		pip := filepath.Join(dest, ".venv", "bin", "pip")
		if out, err := exec.Command(pip, "install", "-r", reqFile).CombinedOutput(); err != nil {
			os.RemoveAll(dest)
			return fail(fmt.Errorf("pip install: %s: %w", strings.TrimSpace(string(out)), err))
		}
	}

	// Generate systemd unit.
	if err := writeSystemdUnit(m.Name, dest, m.Entry, m.Oneshot); err != nil {
		os.RemoveAll(dest)
		return fail(fmt.Errorf("write systemd unit: %w", err))
	}

	// Reload systemd.
	if out, err := exec.Command("systemctl", "daemon-reload").CombinedOutput(); err != nil {
		slog.Warn("[plugins] daemon-reload failed", "component", "plugin", "err", strings.TrimSpace(string(out)))
	}

	// Write source URL for later reference.
	_ = os.WriteFile(filepath.Join(dest, ".source_url"), []byte(rawURL), 0o644)

	s.jobs.Delete(key)
	slog.Info("[plugins] installed", "component", "plugin", "name", m.Name, "version", m.Version)

	return &domain.Plugin{
		Name:        m.Name,
		Version:     m.Version,
		Description: m.Description,
		Entry:       m.Entry,
		Status:      "stopped",
		URL:         rawURL,
	}, nil
}

// List returns all installed plugins with their current status.
func (s *Service) List() []domain.Plugin {
	entries, err := os.ReadDir(pluginsDir)
	if err != nil {
		entries = nil
	}

	seen := map[string]bool{}
	var plugins []domain.Plugin
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if !validPluginName(e.Name()) {
			continue
		}
		dir := filepath.Join(pluginsDir, e.Name())
		m, err := readManifest(dir)
		if err != nil {
			continue
		}

		status := unitStatus(e.Name())

		url := ""
		if data, err := os.ReadFile(filepath.Join(dir, ".source_url")); err == nil {
			url = strings.TrimSpace(string(data))
		}

		name := m.Name
		if !validPluginName(name) {
			name = e.Name()
		}
		seen[name] = true
		plugins = append(plugins, domain.Plugin{
			Name:        name,
			Version:     m.Version,
			Description: m.Description,
			Entry:       m.Entry,
			Status:      status,
			URL:         url,
		})
	}

	s.jobs.Range(func(_, v any) bool {
		job, ok := v.(installJob)
		if !ok {
			return true
		}
		if job.Name != "" && seen[job.Name] {
			return true
		}
		name := job.Name
		if name == "" {
			name = "(installing)"
		}
		desc := job.Err
		if desc == "" {
			desc = job.URL
		}
		plugins = append(plugins, domain.Plugin{
			Name:        name,
			Description: desc,
			Status:      job.Status,
			URL:         job.URL,
		})
		return true
	})

	if plugins == nil {
		return []domain.Plugin{}
	}
	return plugins
}

// Start starts a plugin's systemd unit.
func (s *Service) Start(name string) error {
	if err := validatePluginExists(name); err != nil {
		return err
	}
	unit := unitPrefix + name + ".service"
	// --no-block so a leftover Type=oneshot unit cannot stall the HTTP handler.
	if out, err := exec.Command("systemctl", "start", "--no-block", unit).CombinedOutput(); err != nil {
		return fmt.Errorf("start %s: %s: %w", unit, strings.TrimSpace(string(out)), err)
	}
	slog.Info("[plugins] started", "component", "plugin", "name", name)
	return nil
}

// Stop stops a plugin's systemd unit.
func (s *Service) Stop(name string) error {
	if err := validatePluginExists(name); err != nil {
		return err
	}
	unit := unitPrefix + name + ".service"
	if out, err := exec.Command("systemctl", "stop", unit).CombinedOutput(); err != nil {
		return fmt.Errorf("stop %s: %s: %w", unit, strings.TrimSpace(string(out)), err)
	}
	slog.Info("[plugins] stopped", "component", "plugin", "name", name)
	return nil
}

// Uninstall stops, removes the systemd unit, and deletes the plugin directory.
func (s *Service) Uninstall(name string) error {
	if err := validatePluginExists(name); err != nil {
		return err
	}

	unit := unitPrefix + name + ".service"
	unitPath := filepath.Join(systemdDir, unit)

	// Stop if running.
	_ = exec.Command("systemctl", "stop", unit).Run()

	// Remove systemd unit.
	if err := os.Remove(unitPath); err != nil && !os.IsNotExist(err) {
		slog.Warn("[plugins] remove unit file failed", "component", "plugin", "name", name, "err", err)
	}

	// Reload systemd.
	_ = exec.Command("systemctl", "daemon-reload").Run()

	dir, err := pluginDir(name)
	if err != nil {
		return err
	}
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("remove plugin dir: %w", err)
	}

	slog.Info("[plugins] uninstalled", "component", "plugin", "name", name)
	return nil
}

// readManifest reads and parses plugin.json from a directory.
func readManifest(dir string) (*manifest, error) {
	data, err := os.ReadFile(filepath.Join(dir, "plugin.json"))
	if err != nil {
		return nil, err
	}
	var m manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

// unitStatus checks systemctl active state for a plugin.
func unitStatus(name string) string {
	unit := unitPrefix + name + ".service"
	out, err := exec.Command("systemctl", "is-active", unit).Output()
	if err != nil {
		return "stopped"
	}
	state := strings.TrimSpace(string(out))
	switch state {
	case "active":
		return "running"
	case "failed":
		return "failed"
	default:
		return "stopped"
	}
}

// validatePluginExists checks that a plugin directory exists.
func validatePluginExists(name string) error {
	dir, err := pluginDir(name)
	if err != nil {
		return err
	}
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return fmt.Errorf("plugin %q not found", name)
	}
	return nil
}

func validPluginName(name string) bool {
	return pluginNameRe.MatchString(strings.TrimSpace(name))
}

func validEntry(entry string) bool {
	entry = strings.TrimSpace(entry)
	if entry == "" || strings.HasPrefix(entry, "-") {
		return false
	}
	if filepath.IsAbs(entry) || strings.ContainsAny(entry, "\x00\n\r") {
		return false
	}
	clean := filepath.ToSlash(filepath.Clean(entry))
	if clean == ".." || strings.HasPrefix(clean, "../") {
		return false
	}
	return true
}

func validSubdir(subdir string) error {
	if subdir == "" {
		return nil
	}
	if strings.Contains(subdir, "..") || filepath.IsAbs(subdir) || strings.ContainsAny(subdir, "\\\x00\n\r") {
		return fmt.Errorf("invalid subdir")
	}
	for _, p := range strings.Split(subdir, "/") {
		if p == "" || p == "." || p == ".." {
			return fmt.Errorf("invalid subdir")
		}
	}
	return nil
}

func validGitURL(raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fmt.Errorf("plugin url is required")
	}
	if strings.HasPrefix(raw, "-") || strings.ContainsAny(raw, "\x00\n\r") {
		return fmt.Errorf("invalid plugin url")
	}
	if strings.HasPrefix(raw, "git@") {
		host, path, ok := strings.Cut(raw[4:], ":")
		if !ok || host == "" || path == "" || strings.Contains(host, "/") {
			return fmt.Errorf("invalid plugin url")
		}
		return nil
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return fmt.Errorf("invalid plugin url")
	}
	switch strings.ToLower(u.Scheme) {
	case "https", "http", "ssh", "git":
		return nil
	default:
		return fmt.Errorf("plugin url must be http(s), ssh, or git")
	}
}

func pluginDir(name string) (string, error) {
	name = strings.TrimSpace(name)
	if !validPluginName(name) {
		return "", fmt.Errorf("invalid plugin name")
	}
	dir := filepath.Join(pluginsDir, name)
	rel, err := filepath.Rel(pluginsDir, filepath.Clean(dir))
	if err != nil || rel == "." || strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
		return "", fmt.Errorf("invalid plugin name")
	}
	if rel != name {
		return "", fmt.Errorf("invalid plugin name")
	}
	return dir, nil
}

func gitCloneArgs(url, subdir, dest string) []string {
	args := []string{
		"-c", "filter.lfs.required=false",
		"-c", "filter.lfs.smudge=",
		"-c", "filter.lfs.process=",
		"clone", "--depth=1", "--filter=blob:none",
	}
	if subdir != "" {
		args = append(args, "--sparse")
	}
	args = append(args, "--", url, dest)
	return args
}

func cloneRepo(url, subdir, dest string) error {
	if out, err := exec.Command("git", gitCloneArgs(url, subdir, dest)...).CombinedOutput(); err != nil {
		return fmt.Errorf("git clone: %s: %w", strings.TrimSpace(string(out)), err)
	}
	if subdir == "" {
		return nil
	}
	if out, err := exec.Command("git", "-C", dest, "sparse-checkout", "set", "--cone", "--", subdir).CombinedOutput(); err != nil {
		return fmt.Errorf("git sparse-checkout: %s: %w", strings.TrimSpace(string(out)), err)
	}
	return nil
}

func systemdUnitBody(name, dir, entry string, oneshot bool) string {
	restart := "Restart=on-failure\nRestartSec=5"
	// Finite "oneshot" apps (dance, phrase teacher) must be Type=simple so
	// systemctl start returns immediately and is-active is "running" while
	// they work. Type=oneshot blocks Start for the whole routine and reports
	// "activating", which the UI maps to stopped.
	if oneshot {
		restart = "Restart=no"
	}
	return fmt.Sprintf(`[Unit]
Description=Autonomous Plugin: %s
After=network.target

[Service]
Type=simple
WorkingDirectory=%s
ExecStart=%s %s
Environment=HAL_URL=http://localhost:5001
%s
MemoryMax=256M

[Install]
WantedBy=multi-user.target
`, name, dir, filepath.Join(dir, ".venv", "bin", "python"), entry, restart)
}

func writeSystemdUnit(name, dir, entry string, oneshot bool) error {
	unitPath := filepath.Join(systemdDir, unitPrefix+name+".service")
	return os.WriteFile(unitPath, []byte(systemdUnitBody(name, dir, entry, oneshot)), 0o644)
}
