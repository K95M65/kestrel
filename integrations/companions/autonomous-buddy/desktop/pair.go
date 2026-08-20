package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type pairConfirmResponse struct {
	Token   string `json:"token"`
	BuddyID string `json:"buddy_id"`
}

func normalizeHost(host string) string {
	h := strings.TrimSpace(host)
	h = strings.TrimPrefix(h, "http://")
	h = strings.TrimPrefix(h, "https://")
	h = strings.TrimSuffix(h, "/")
	return h
}

func pairWithDevice(host, code, name, fp, osVer string) (pairingRecord, error) {
	h := normalizeHost(host)
	body, _ := json.Marshal(map[string]string{
		"code":        code,
		"name":        name,
		"fingerprint": fp,
		"os_version":  osVer,
	})
	req, err := http.NewRequest(http.MethodPost, "http://"+h+"/api/buddy/pair/confirm", bytes.NewReader(body))
	if err != nil {
		return pairingRecord{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 12 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return pairingRecord{}, err
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return pairingRecord{}, fmt.Errorf("pairing failed (%d): %s", res.StatusCode, strings.TrimSpace(string(raw)))
	}
	var env struct {
		Status int                  `json:"status"`
		Data   pairConfirmResponse  `json:"data"`
		Token  string               `json:"token"`
		Buddy  string               `json:"buddy_id"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return pairingRecord{}, fmt.Errorf("malformed pairing response")
	}
	token, id := env.Data.Token, env.Data.BuddyID
	if token == "" {
		token = env.Token
	}
	if id == "" {
		id = env.Buddy
	}
	if token == "" || id == "" {
		return pairingRecord{}, fmt.Errorf("malformed pairing response")
	}
	return pairingRecord{
		BuddyID:    id,
		DeviceHost: h,
		Token:      token,
		PairedAt:   time.Now().UTC().Format(time.RFC3339),
	}, nil
}

func notifyRevoke(host, token string) {
	h := normalizeHost(host)
	req, err := http.NewRequest(http.MethodDelete, "http://"+h+"/api/buddy/self", nil)
	if err != nil {
		return
	}
	req.Header.Set("Authorization", "Bearer "+token)
	client := &http.Client{Timeout: 5 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return
	}
	res.Body.Close()
}
