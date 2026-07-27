package domain

import (
	"testing"

	"github.com/go-playground/validator/v10"
)

// baseSetupRequest returns a request with every still-required field populated,
// so each test below varies exactly one thing.
func baseSetupRequest() SetupRequest {
	return SetupRequest{
		SSID:             "home-wifi",
		Password:         "hunter2",
		LLMBaseURL:       "https://api.example.com",
		LLMAPIKey:        "sk-test",
		DeviceID:         "lamp-7f72",
		Channel:          "telegram",
		TelegramBotToken: "123:abc",
		TelegramUserID:   "456",
	}
}

// TestSetupRequestNetworkFieldsOptional locks in the contract that makes a
// wired setup possible: an absent SSID is a valid request meaning "the device
// already has an uplink, don't join any WiFi". device.Setup is what verifies
// that claim — validation must not reject it up front.
func TestSetupRequestNetworkFieldsOptional(t *testing.T) {
	v := validator.New()

	tests := []struct {
		name     string
		mutate   func(*SetupRequest)
		wantPass bool
	}{
		{
			name:     "full wifi request",
			mutate:   func(*SetupRequest) {},
			wantPass: true,
		},
		{
			// The ethernet case.
			name:     "no ssid and no password",
			mutate:   func(r *SetupRequest) { r.SSID = ""; r.Password = "" },
			wantPass: true,
		},
		{
			// Open WiFi — also rejected before this change.
			name:     "ssid without password",
			mutate:   func(r *SetupRequest) { r.Password = "" },
			wantPass: true,
		},
		{
			// Guard against over-relaxing: the rest of the contract still holds.
			name:     "missing llm api key still rejected",
			mutate:   func(r *SetupRequest) { r.LLMAPIKey = "" },
			wantPass: false,
		},
		{
			name:     "missing device id still rejected",
			mutate:   func(r *SetupRequest) { r.DeviceID = "" },
			wantPass: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := baseSetupRequest()
			tt.mutate(&req)
			err := v.Struct(req)
			if tt.wantPass && err != nil {
				t.Fatalf("validation rejected a request that should pass: %v", err)
			}
			if !tt.wantPass && err == nil {
				t.Fatal("validation accepted a request that should be rejected")
			}
		})
	}
}
