package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

type pairingRecord struct {
	BuddyID    string `json:"buddy_id"`
	DeviceHost string `json:"device_host"`
	Token      string `json:"token"`
	PairedAt   string `json:"paired_at"`
}

type fileStore struct {
	mu   sync.Mutex
	path string
}

func defaultStorePath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	// Historical folder name — keep so existing pairing.json files still load.
	p := filepath.Join(dir, "AutonomousBuddy")
	if err := os.MkdirAll(p, 0o700); err != nil {
		return "", err
	}
	return filepath.Join(p, "pairing.json"), nil
}

func (s *fileStore) load() (*pairingRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var rec pairingRecord
	if err := json.Unmarshal(data, &rec); err != nil {
		return nil, err
	}
	if rec.Token == "" {
		return nil, nil
	}
	return &rec, nil
}

func (s *fileStore) save(rec pairingRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0o600)
}

func (s *fileStore) clear() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return os.Remove(s.path)
}

func fingerprint() string {
	host, _ := os.Hostname()
	user := os.Getenv("USER")
	if user == "" {
		user = os.Getenv("USERNAME")
	}
	raw := host + "|" + user + "|" + runtime.GOOS + "|" + runtime.GOARCH
	if b, err := os.ReadFile("/etc/machine-id"); err == nil {
		raw += "|" + string(b)
	}
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:8])
}

func deviceName() string {
	host, err := os.Hostname()
	if err != nil || host == "" {
		return "Kestrel Buddy"
	}
	return host
}
