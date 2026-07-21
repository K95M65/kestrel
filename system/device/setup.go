package device

import (
	"fmt"
	"log/slog"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"

	"go.autonomous.ai/os/system/domain"
	"go.autonomous.ai/os/system/lib/urlnorm"
	"go.autonomous.ai/os/system/statusled"
)

// Setup phase strings exposed via /api/setup/status so the web client can
// follow the device through the AP→STA transition. Phases progress only
// forward; failures park at "failed".
const (
	SetupPhaseIdle       = "idle"
	SetupPhaseConnecting = "connecting"
	SetupPhaseConnected  = "connected"
	SetupPhaseFailed     = "failed"
)

// apSetupIP is wlan0's static address while the device runs the provisioning
// AP (see scripts/provision/setup-ap.sh). The early LAN-IP poll skips it so it
// only ever publishes the STA-side address from the home router's DHCP.
const apSetupIP = "192.168.100.1"

type setupState struct {
	mu    sync.RWMutex
	phase string
	lanIP string
	error string
}

func (st *setupState) snapshot() (phase, ip, errMsg string) {
	st.mu.RLock()
	defer st.mu.RUnlock()
	return st.phase, st.lanIP, st.error
}

func (st *setupState) set(phase, ip, errMsg string) {
	st.mu.Lock()
	st.phase = phase
	st.lanIP = ip
	st.error = errMsg
	st.mu.Unlock()
}

// SetupStatus returns the current Setup phase + LAN IP so the web client
// can poll progress through the AP→STA switch. When no Setup run has
// happened (phase=idle) but the device is already on home Wi-Fi from a
// previous session, fall back to the live wlan0 address so the web
// client can still detect "you're at the AP IP but the device lives at X"
// and redirect.
func (s *Service) SetupStatus() (phase, lanIP, errMsg string) {
	phase, lanIP, errMsg = s.setupState.snapshot()
	if lanIP == "" {
		if ip, err := s.networkService.GetCurrentIP(); err == nil {
			lanIP = ip
		}
	}
	return phase, lanIP, errMsg
}

func (s *Service) Setup(data domain.SetupRequest) error {
	slog.Info("starting setup", "component", "device")
	data.LLMBaseURL = urlnorm.NormalizeBaseURL(data.LLMBaseURL)
	data.STTBaseURL = urlnorm.NormalizeBaseURL(data.STTBaseURL)
	data.TTSBaseURL = urlnorm.NormalizeBaseURL(data.TTSBaseURL)
	s.setupState.set(SetupPhaseConnecting, "", "")

	// Blue-blink cue while wlan0 associates with the target Wi-Fi. Mirrors the
	// intern-v1 behavior (openclaw-lobster's led.ConnectionMode on setup entry).
	// Cleared on every return path below so a re-run after a failed setup starts
	// from the neutral status instead of a stuck blinking strip. No-op on devices
	// without the `light` capability (statusled short-circuits).
	s.statusLED.Set(statusled.StateWifiConnecting)
	defer s.statusLED.Clear(statusled.StateWifiConnecting)

	// Early LAN-IP capture: SetupNetwork() blocks up to 60s waiting for
	// internet, but the AP (192.168.100.1) tears down within ~2s of the
	// AP→STA switch — so by the time SetupNetwork returns and we'd normally
	// read the IP, the web client can no longer poll us over the AP. This
	// goroutine polls wlan0 while SetupNetwork runs and publishes the new STA
	// IP into setupState the instant it appears (before internet is even up),
	// giving the FE the largest possible window to read lan_ip during the
	// brief overlap where it's still polling. Phase stays "connecting" — a
	// LAN IP alone doesn't prove the join fully succeeded; SetupNetwork's
	// return flips it to connected/failed below.
	ipPollDone := make(chan struct{})
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ipPollDone:
				return
			case <-ticker.C:
				ip, ipErr := s.networkService.GetCurrentIP()
				// Ignore the AP's own static IP — we want the STA-side
				// address handed out by the home router's DHCP.
				if ipErr == nil && ip != "" && ip != apSetupIP {
					if _, prevIP, _ := s.setupState.snapshot(); prevIP != ip {
						s.setupState.set(SetupPhaseConnecting, ip, "")
						slog.Info("setup: early LAN IP captured", "component", "device", "lan_ip", ip)
					}
				}
			}
		}
	}()

	result, err := s.networkService.SetupNetwork(data.SSID, data.Password)
	close(ipPollDone)
	if err != nil {
		s.setupState.set(SetupPhaseFailed, "", err.Error())
		return fmt.Errorf("setup network: %w", err)
	}
	if !result {
		s.setupState.set(SetupPhaseFailed, "", "network setup failed")
		return fmt.Errorf("network setup failed")
	}
	// Capture the LAN IP immediately after WiFi associates so the web
	// client polling /api/setup/status can read it before AP shuts down.
	// Re-reading here can fail transiently while the AP tears down — in that
	// case keep whatever the early-capture goroutine already published rather
	// than clobbering a good IP with an empty string.
	ip, ipErr := s.networkService.GetCurrentIP()
	if ipErr != nil || ip == "" || ip == apSetupIP {
		_, prevIP, _ := s.setupState.snapshot()
		ip = prevIP
	}
	if ip != "" {
		s.setupState.set(SetupPhaseConnected, ip, "")
		slog.Info("setup: WiFi associated", "component", "device", "lan_ip", ip)
	} else {
		s.setupState.set(SetupPhaseConnected, "", "")
		slog.Warn("setup: WiFi associated but no IP detected", "component", "device", "error", ipErr)
	}

	// Persist the user's model selection so SetupAgent (run below, AFTER the full
	// config is saved) can fall back to it when the model API is unreachable.
	s.config.LLMModel = data.LLMModel

	llmAPIKey := data.LLMAPIKey
	llmBaseURL := data.LLMBaseURL
	channel := data.EffectiveChannel()

	s.config.LLMAPIKey = llmAPIKey
	s.config.LLMBaseURL = llmBaseURL
	// LLMModel already set above (and possibly overridden by SetupAgent from the
	// upstream default_model). Do not re-assign it from the raw request here.
	s.config.Channel = channel
	switch channel {
	case "slack":
		s.config.SlackBotToken = data.SlackBotToken
		s.config.SlackAppToken = data.SlackAppToken
		s.config.SlackUserID = data.SlackUserID
	case "discord":
		s.config.DiscordBotToken = data.DiscordBotToken
		s.config.DiscordUserID = data.DiscordUserID
	default:
		s.config.TelegramBotToken = data.TelegramBotToken
		s.config.TelegramUserID = data.TelegramUserID
	}
	s.config.DeviceID = data.DeviceID
	s.config.DeepgramAPIKey = data.DeepgramAPIKey
	s.config.STTAPIKey = data.STTAPIKey
	s.config.TTSAPIKey = data.TTSAPIKey
	s.config.STTBaseURL = data.STTBaseURL
	s.config.TTSBaseURL = data.TTSBaseURL
	s.config.STTLanguage = data.STTLanguage
	s.config.STTModel = sttModelForLanguage(data.STTLanguage)
	if data.TTSProvider != "" {
		s.config.TTSProvider = data.TTSProvider
	}
	if data.TTSVoice != "" {
		s.config.TTSVoice = data.TTSVoice
	}
	s.config.MQTTEndpoint = data.MQTTEndpoint
	s.config.MQTTUsername = data.MQTTUsername
	s.config.MQTTPassword = data.MQTTPassword
	s.config.MQTTPort = data.MQTTPort
	s.config.FAChannel = data.FAChannel
	s.config.FDChannel = data.FDChannel
	if data.LLMDisableThinking != nil {
		s.config.LLMDisableThinking = data.LLMDisableThinking
	}
	// Admin password is hashed once and never persisted in plaintext. Empty
	// is permitted so older clients that don't send it still complete setup;
	// the operator can set one later via PUT /api/device/config (TODO) or
	// re-run setup after factory reset.
	if data.AdminPassword != "" {
		hash, hashErr := bcrypt.GenerateFromPassword([]byte(data.AdminPassword), bcrypt.DefaultCost)
		if hashErr != nil {
			return fmt.Errorf("hash admin password: %w", hashErr)
		}
		s.config.AdminPasswordHash = string(hash)
	}
	if err := s.config.Save(); err != nil {
		slog.Error("save config failed", "component", "device", "error", err)
	}
	slog.Info("config saved", "component", "device")

	// Early presence ping — fire-and-forget: publish the freshly-acquired STA
	// IP to the backend the moment WiFi + config are ready, WITHOUT waiting for
	// the agent setup below (up to ~2 min). The page that opened the Setup
	// popup polls the backend for this IP and redirects the popup when neither
	// the AP-alive window nor mDNS could deliver it (see docs/setup-flow.md).
	// Must run after config.Save above: beclient derives the ping URL from the
	// just-assigned LLMBaseURL.
	if s.beClient != nil && llmAPIKey != "" {
		go func() { s.beClient.PingSafe(llmAPIKey, s.buildPingPayload("setting_up")) }()
	}

	// SetupAgent runs AFTER config.json is saved: a backend that materializes its
	// own config from config.json (Hermes presync) then sees the freshly-entered
	// llm_api_key/base_url + channel tokens immediately, instead of waiting for the
	// next os-server boot. OpenClaw writes openclaw.json from the request `data`, so
	// its result is unchanged; any LLMModel override it applies is persisted by the
	// SetUpCompleted save below.
	if err := s.agentGateway.SetupAgent(data); err != nil {
		return err
	}

	// Wait for agent gateway to be ready before marking device as working.
	if ok := s.WaitForAgentReady(120 * time.Second); !ok {
		return fmt.Errorf("agent gateway ready timeout, something went wrong")
	}

	s.config.SetUpCompleted = true
	if err := s.config.Save(); err != nil {
		slog.Error("save config failed", "component", "device", "error", err)
	}

	slog.Info("agent gateway is ready", "component", "device")
	if s.beClient != nil && llmAPIKey != "" {
		s.beClient.PingSafe(llmAPIKey, s.buildPingPayload("working"))
	}
	return nil
}
