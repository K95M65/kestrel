package plugin

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"go.autonomous.ai/os/system/domain"
)

const (
	pluginsDir = "/var/lib/os-plugins"
	unitPrefix = "os-plugin-"
	systemdDir = "/etc/systemd/system"
)

// manifest is the parsed plugin.json.
type manifest struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
	Entry       string `json:"entry"`
	Oneshot     bool   `json:"oneshot"`
}

type Service struct{}

func ProvideService() *Service {
	return &Service{}
}

// Install clones a git repo, sets up a venv, and creates a systemd unit.
func (s *Service) Install(url string) (*domain.Plugin, error) {
	return s.InstallFrom(domain.PluginInstallRequest{URL: url})
}

// InstallFrom clones url (optionally one subdir) and installs the plugin.
func (s *Service) InstallFrom(req domain.PluginInstallRequest) (*domain.Plugin, error) {
	url := strings.TrimSpace(req.URL)
	subdir := strings.Trim(strings.TrimSpace(req.Subdir), "/")
	if url == "" {
		return nil, fmt.Errorf("plugin url is required")
	}
	if strings.Contains(subdir, "..") {
		return nil, fmt.Errorf("invalid subdir")
	}

	// Clone to a temp dir first, then read plugin.json to get the name.
	tmpDir, err := os.MkdirTemp("", "os-plugin-clone-*")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	slog.Info("[plugins] cloning", "component", "plugin", "url", url, "subdir", subdir)
	if err := cloneRepo(url, subdir, tmpDir); err != nil {
		return nil, err
	}

	src := tmpDir
	if subdir != "" {
		src = filepath.Join(tmpDir, subdir)
	}

	// Parse plugin.json.
	m, err := readManifest(src)
	if err != nil {
		return nil, fmt.Errorf("read plugin.json: %w", err)
	}
	if m.Name == "" {
		return nil, fmt.Errorf("plugin.json: name is required")
	}
	if m.Entry == "" {
		m.Entry = "main.py"
	}

	// Ensure plugins dir exists.
	if err := os.MkdirAll(pluginsDir, 0o755); err != nil {
		return nil, fmt.Errorf("create plugins dir: %w", err)
	}

	dest := filepath.Join(pluginsDir, m.Name)
	if _, err := os.Stat(dest); err == nil {
		return nil, fmt.Errorf("plugin %q already installed", m.Name)
	}

	// Move the plugin tree (whole clone, or just the subdir) into place.
	if out, err := exec.Command("mv", src, dest).CombinedOutput(); err != nil {
		return nil, fmt.Errorf("move plugin: %s: %w", strings.TrimSpace(string(out)), err)
	}

	// Create Python venv.
	slog.Info("[plugins] creating venv", "component", "plugin", "name", m.Name)
	if out, err := exec.Command("python3", "-m", "venv", filepath.Join(dest, ".venv")).CombinedOutput(); err != nil {
		os.RemoveAll(dest)
		return nil, fmt.Errorf("create venv: %s: %w", strings.TrimSpace(string(out)), err)
	}

	// Install requirements if present.
	reqFile := filepath.Join(dest, "requirements.txt")
	if _, err := os.Stat(reqFile); err == nil {
		slog.Info("[plugins] installing requirements", "component", "plugin", "name", m.Name)
		pip := filepath.Join(dest, ".venv", "bin", "pip")
		if out, err := exec.Command(pip, "install", "-r", reqFile).CombinedOutput(); err != nil {
			os.RemoveAll(dest)
			return nil, fmt.Errorf("pip install: %s: %w", strings.TrimSpace(string(out)), err)
		}
	}

	// Generate systemd unit.
	if err := writeSystemdUnit(m.Name, dest, m.Entry, m.Oneshot); err != nil {
		os.RemoveAll(dest)
		return nil, fmt.Errorf("write systemd unit: %w", err)
	}

	// Reload systemd.
	if out, err := exec.Command("systemctl", "daemon-reload").CombinedOutput(); err != nil {
		slog.Warn("[plugins] daemon-reload failed", "component", "plugin", "err", strings.TrimSpace(string(out)))
	}

	// Write source URL for later reference.
	os.WriteFile(filepath.Join(dest, ".source_url"), []byte(url), 0o644)

	slog.Info("[plugins] installed", "component", "plugin", "name", m.Name, "version", m.Version)

	return &domain.Plugin{
		Name:        m.Name,
		Version:     m.Version,
		Description: m.Description,
		Entry:       m.Entry,
		Status:      "stopped",
		URL:         url,
	}, nil
}

// List returns all installed plugins with their current status.
func (s *Service) List() []domain.Plugin {
	entries, err := os.ReadDir(pluginsDir)
	if err != nil {
		return []domain.Plugin{}
	}

	var plugins []domain.Plugin
	for _, e := range entries {
		if !e.IsDir() {
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

		plugins = append(plugins, domain.Plugin{
			Name:        m.Name,
			Version:     m.Version,
			Description: m.Description,
			Entry:       m.Entry,
			Status:      status,
			URL:         url,
		})
	}
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
	if out, err := exec.Command("systemctl", "start", unit).CombinedOutput(); err != nil {
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
	exec.Command("systemctl", "stop", unit).Run()

	// Remove systemd unit.
	if err := os.Remove(unitPath); err != nil && !os.IsNotExist(err) {
		slog.Warn("[plugins] remove unit file failed", "component", "plugin", "name", name, "err", err)
	}

	// Reload systemd.
	exec.Command("systemctl", "daemon-reload").Run()

	// Remove plugin directory.
	dir := filepath.Join(pluginsDir, name)
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
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("plugin name is required")
	}
	dir := filepath.Join(pluginsDir, name)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return fmt.Errorf("plugin %q not found", name)
	}
	return nil
}

// writeSystemdUnit generates and writes a systemd service unit file.
func cloneRepo(url, subdir, dest string) error {
	args := []string{"clone", "--depth=1", "--filter=blob:none"}
	if subdir != "" {
		args = append(args, "--sparse")
	}
	args = append(args, url, dest)
	if out, err := exec.Command("git", args...).CombinedOutput(); err != nil {
		return fmt.Errorf("git clone: %s: %w", strings.TrimSpace(string(out)), err)
	}
	if subdir == "" {
		return nil
	}
	if out, err := exec.Command("git", "-C", dest, "sparse-checkout", "set", "--cone", subdir).CombinedOutput(); err != nil {
		return fmt.Errorf("git sparse-checkout: %s: %w", strings.TrimSpace(string(out)), err)
	}
	return nil
}

func writeSystemdUnit(name, dir, entry string, oneshot bool) error {
	restart := "Restart=on-failure\nRestartSec=5"
	svcType := "simple"
	if oneshot {
		restart = "Restart=no"
		svcType = "oneshot"
	}
	unit := fmt.Sprintf(`[Unit]
Description=Autonomous Plugin: %s
After=network.target

[Service]
Type=%s
WorkingDirectory=%s
ExecStart=%s %s
Environment=HAL_URL=http://localhost:5001
%s
MemoryMax=256M

[Install]
WantedBy=multi-user.target
`, name, svcType, dir, filepath.Join(dir, ".venv", "bin", "python"), entry, restart)

	unitPath := filepath.Join(systemdDir, unitPrefix+name+".service")
	return os.WriteFile(unitPath, []byte(unit), 0o644)
}
