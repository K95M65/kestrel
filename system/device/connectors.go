package device

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"go.autonomous.ai/os/system/domain"
)

const (
	connectorsSchemaVer = 1
	connectorsFileMode  = 0o600
	connectorsDirMode   = 0o700
)

var validConnectorCode = regexp.MustCompile(`^[a-z0-9_-]{1,64}$`)

// ServiceStatus is a secret-free row for Guided Setup / Device → Channels.
type ServiceStatus struct {
	ID         string `json:"id"`
	Kind       string `json:"kind"` // connector | channel
	Connected  bool   `json:"connected"`
	AuthType   string `json:"auth_type,omitempty"`
	UserEmail  string `json:"user_email,omitempty"`
	Label      string `json:"label,omitempty"`
	ConnectHow string `json:"connect_how,omitempty"` // pat | oauth | ical | token
}

// SetConnectorPAT is the home-user Gmail path: Google app password + email.
type SetConnectorPAT struct {
	Code  string
	Email string
	Key   string
}

func (s *Service) connectorsDir() string {
	dir := strings.TrimSpace(s.config.OpenclawConfigDir)
	if dir == "" {
		dir = "/root/.openclaw"
	}
	return filepath.Join(dir, "workspace", "configs")
}

// ListServices reports Gmail, Calendar, and Telegram without secrets.
func (s *Service) ListServices() []ServiceStatus {
	out := []ServiceStatus{
		s.connectorStatus("gmail", "pat"),
		s.connectorStatus("google_calendar", "ical"),
		{
			ID:         "telegram",
			Kind:       "channel",
			Connected:  s.config.TelegramBotToken != "" && s.config.TelegramUserID != "",
			Label:      s.config.TelegramUserID,
			ConnectHow: "token",
		},
	}
	return out
}

func (s *Service) connectorStatus(code, how string) ServiceStatus {
	st := ServiceStatus{ID: code, Kind: "connector", ConnectHow: how}
	entry, ok, err := s.readConnector(code)
	if err != nil || !ok {
		return st
	}
	st.AuthType = entry.AuthType
	st.UserEmail = entry.UserEmail
	if st.UserEmail == "" && entry.Credentials != nil {
		st.UserEmail = entry.Credentials["email"]
	}
	ical := entry.AuthType == "ical" && entry.Credentials != nil && strings.TrimSpace(entry.Credentials["url"]) != ""
	st.Connected = entry.AccessToken != "" || entry.APIKey != "" || ical
	if ical {
		st.ConnectHow = "ical"
		if st.UserEmail == "" {
			st.Label = "iCal"
		}
	}
	if entry.AuthType == "oauth" {
		st.ConnectHow = "oauth"
	}
	if entry.ExpiresAt > 0 && entry.ExpiresAt < time.Now().Unix() && entry.APIKey == "" && !ical {
		st.Connected = false
		st.Label = "expired"
	}
	return st
}

func (s *Service) readConnector(code string) (domain.ConnectorEntry, bool, error) {
	path, err := s.connectorPath(code)
	if err != nil {
		return domain.ConnectorEntry{}, false, err
	}
	file, err := loadConnectorsFile(path)
	if err != nil {
		return domain.ConnectorEntry{}, false, err
	}
	entry, ok := file.Connectors[code]
	return entry, ok, nil
}

func (s *Service) connectorPath(code string) (string, error) {
	if !validConnectorCode.MatchString(code) {
		return "", fmt.Errorf("invalid connector code")
	}
	return filepath.Join(s.connectorsDir(), code+"_access_tokens.json"), nil
}

// SetGmailPAT stores a Google app password so overnight mail works without OAuth.
// Calendar is not accepted here — Google Calendar's REST API rejects app passwords.
func (s *Service) SetGmailPAT(in SetConnectorPAT) error {
	code := strings.TrimSpace(in.Code)
	if code != "gmail" {
		return fmt.Errorf("only gmail accepts an app password here; calendar needs Google sign-in")
	}
	email := strings.TrimSpace(in.Email)
	key := strings.TrimSpace(in.Key)
	if !strings.Contains(email, "@") {
		return fmt.Errorf("email is required")
	}
	if len(key) < 8 {
		return fmt.Errorf("app password is too short")
	}
	path, err := s.connectorPath(code)
	if err != nil {
		return err
	}
	file, err := loadConnectorsFile(path)
	if err != nil {
		return err
	}
	file.Connectors[code] = domain.ConnectorEntry{
		AuthType:    "pat",
		APIKey:      key,
		Credentials: map[string]string{"email": email},
		ObtainedAt:  time.Now().Unix(),
	}
	return writeConnectorsFile(path, file)
}

// SetCalendarICS stores a Google Calendar secret iCal URL (desk path; no OAuth client).
func (s *Service) SetCalendarICS(rawURL string) error {
	u, err := normalizeGoogleICS(rawURL)
	if err != nil {
		return err
	}
	path, err := s.connectorPath("google_calendar")
	if err != nil {
		return err
	}
	file, err := loadConnectorsFile(path)
	if err != nil {
		return err
	}
	file.Connectors["google_calendar"] = domain.ConnectorEntry{
		AuthType:    "ical",
		Credentials: map[string]string{"url": u},
		ObtainedAt:  time.Now().Unix(),
	}
	return writeConnectorsFile(path, file)
}

func normalizeGoogleICS(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" {
		return "", fmt.Errorf("paste the secret https iCal address from Google Calendar")
	}
	host := strings.ToLower(u.Hostname())
	if host != "calendar.google.com" {
		return "", fmt.Errorf("iCal address must be from calendar.google.com")
	}
	p := strings.ToLower(u.EscapedPath())
	if !strings.Contains(p, "/calendar/ical/") || !strings.HasSuffix(p, ".ics") {
		return "", fmt.Errorf("that does not look like a Google Calendar iCal address")
	}
	return u.String(), nil
}

// RemoveConnector deletes a token file. Does not touch MCP (gmail is credential-only).
func (s *Service) RemoveConnector(code string) error {
	path, err := s.connectorPath(code)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}

func loadConnectorsFile(path string) (domain.ConnectorsFile, error) {
	out := domain.ConnectorsFile{Version: connectorsSchemaVer, Connectors: map[string]domain.ConnectorEntry{}}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return out, nil
		}
		return out, fmt.Errorf("read %s: %w", path, err)
	}
	if len(data) == 0 {
		return out, nil
	}
	if err := json.Unmarshal(data, &out); err != nil {
		return out, fmt.Errorf("parse %s: %w", path, err)
	}
	if out.Connectors == nil {
		out.Connectors = map[string]domain.ConnectorEntry{}
	}
	if out.Version == 0 {
		out.Version = connectorsSchemaVer
	}
	return out, nil
}

func writeConnectorsFile(path string, file domain.ConnectorsFile) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, connectorsDirMode); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}
	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	tmp, err := os.CreateTemp(dir, ".connectors.*.tmp")
	if err != nil {
		return fmt.Errorf("create tmp: %w", err)
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return err
	}
	if err := tmp.Chmod(connectorsFileMode); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return err
	}
	return nil
}
