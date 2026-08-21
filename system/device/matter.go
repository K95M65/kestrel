package device

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// MatterStatus is the HA-backed commissioner. Kestrel is not a Matter
// accessory (no CHIP stack / DAC). It commissions devices into Home Assistant
// the same way Google Home / Apple Home scan a Matter QR.
type MatterStatus struct {
	Ready bool   `json:"ready"`
	HAURL string `json:"ha_url,omitempty"`
	Hint  string `json:"hint"`
}

func (s *Service) MatterStatus() MatterStatus {
	b := s.config.BehaviorsOrDefault()
	st := MatterStatus{
		Hint: "Paste a Matter pairing code. This robot asks Home Assistant to add the device. It does not join Apple Home or Google Home as an accessory.",
	}
	if b.HomeAssistant.Enabled && strings.TrimSpace(b.HomeAssistant.URL) != "" && s.haToken() != "" {
		st.Ready = true
		st.HAURL = strings.TrimRight(strings.TrimSpace(b.HomeAssistant.URL), "/")
	} else {
		st.Hint = "Turn on House → Behaviors → Home Assistant (URL + token). Then paste a Matter QR / pairing code here."
	}
	return st
}

func (s *Service) haToken() string {
	if s.config == nil || s.config.Behaviors == nil {
		return ""
	}
	return strings.TrimSpace(s.config.Behaviors.HomeAssistant.Token)
}

// CommissionMatter posts the pairing code to Home Assistant's matter.commission.
func (s *Service) CommissionMatter(code string) error {
	code = strings.TrimSpace(code)
	if code == "" {
		return fmt.Errorf("pairing code is required")
	}
	st := s.MatterStatus()
	if !st.Ready {
		return fmt.Errorf("%s", st.Hint)
	}
	token := s.haToken()
	body, _ := json.Marshal(map[string]any{"code": code})
	req, err := http.NewRequest(http.MethodPost, st.HAURL+"/api/services/matter/commission", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	cli := &http.Client{Timeout: 60 * time.Second}
	resp, err := cli.Do(req)
	if err != nil {
		return fmt.Errorf("home assistant: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("home assistant HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	return nil
}
