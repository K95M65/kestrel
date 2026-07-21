package device

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"golang.org/x/crypto/bcrypt"

	"go.autonomous.ai/os/system/domain"
	"go.autonomous.ai/os/system/lib/i18n"
	"go.autonomous.ai/os/system/lib/urlnorm"
	"go.autonomous.ai/os/system/server/config"
)

// GetPublicConfig returns the device configuration with secrets replaced by
// presence booleans, suitable for browser bootstrap. The web UI renders
// write-only fields against the `Has*` flags so plaintext tokens never reach
// the DOM / sessionStorage / HAR captures.
func (s *Service) GetPublicConfig() domain.ConfigPublicResponse {
	disableThinking := false
	if s.config.LLMDisableThinking != nil {
		disableThinking = *s.config.LLMDisableThinking
	}
	deviceID := s.config.DeviceID
	if deviceID == "" {
		deviceID = GetDeviceMac()
	}
	return domain.ConfigPublicResponse{
		Channel:            s.config.Channel,
		TelegramUserID:     s.config.TelegramUserID,
		SlackUserID:        s.config.SlackUserID,
		DiscordGuildID:     s.config.DiscordGuildID,
		DiscordUserID:      s.config.DiscordUserID,
		WhatsappUserID:     s.config.WhatsappUserID,
		LLMModel:           s.config.LLMModel,
		LLMBaseURL:         s.config.LLMBaseURL,
		LLMDisableThinking: disableThinking,
		STTBaseURL:         s.config.STTBaseURL,
		TTSBaseURL:         s.config.TTSBaseURL,
		STTLanguage:        s.config.STTLanguage,
		STTModel:           s.config.STTModel,
		TTSProvider:        s.config.TTSProvider,
		TTSVoice:           s.config.TTSVoice,
		DeviceID:           deviceID,
		Mac:                GetDeviceMac(),
		NetworkSSID:        s.config.NetworkSSID,
		MQTTEndpoint:       s.config.MQTTEndpoint,
		MQTTUsername:       s.config.MQTTUsername,
		MQTTPort:           s.config.MQTTPort,
		FAChannel:          s.config.FAChannel,
		FDChannel:          s.config.FDChannel,

		HasTelegramBotToken: s.config.TelegramBotToken != "",
		HasSlackBotToken:    s.config.SlackBotToken != "",
		HasSlackAppToken:    s.config.SlackAppToken != "",
		HasDiscordBotToken:  s.config.DiscordBotToken != "",
		HasLLMAPIKey:        s.config.LLMAPIKey != "",
		HasDeepgramAPIKey:   s.config.DeepgramAPIKey != "",
		HasSTTAPIKey:        s.config.STTAPIKey != "",
		HasTTSAPIKey:        s.config.TTSAPIKey != "",
		HasNetworkPassword:  s.config.NetworkPassword != "",
		HasMQTTPassword:     s.config.MQTTPassword != "",
		HasAdminPassword:    s.config.AdminPasswordHash != "",
		Realtime: domain.RealtimePublic{
			Enabled:   s.config.RealtimeEnabled(),
			Provider:  s.config.RealtimeProvider(),
			Model:     s.config.RealtimeModel(),
			Voice:     s.config.RealtimeVoice(),
			Reasoning: s.config.RealtimeReasoning(),
			// Expose ONLY the explicit override (blank when deriving) so the web
			// form's "leave blank to derive" works and does not re-persist a bare
			// URL that breaks HAL's /ws/gemini handshake. See RealtimeBaseURL doc.
			BaseURL:   s.config.RealtimeBaseURLOverride(),
			HasAPIKey: s.config.RealtimeHasAPIKey(),
		},
	}
}

// VerifyAdminPassword returns nil when password matches the stored bcrypt hash.
// Returns an error when no password is set, when the hash is malformed, or when
// the password is wrong. Callers must not surface the specific error to clients
// (uniform "invalid credentials" message) to avoid leaking which case fired.
func (s *Service) VerifyAdminPassword(password string) error {
	if s.config.AdminPasswordHash == "" {
		return fmt.Errorf("admin password not configured")
	}
	return bcrypt.CompareHashAndPassword([]byte(s.config.AdminPasswordHash), []byte(password))
}

// UpdateConfig saves updated config fields. All fields are optional; empty strings are skipped.
// Side effects per field cluster: wifi → connect-wifi (wpa_supplicant reload),
// llm_model/thinking → openclaw, stt_language → openclaw NewSession + hal,
// voice-pipeline fields → hal. Other fields persist only; restart os-server for full effect.
func (s *Service) UpdateConfig(data domain.UpdateConfigRequest) error {
	// bcrypt is CPU-intensive; compute before acquiring the config lock.
	var adminHash string
	if data.AdminPassword != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(data.AdminPassword), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("hash admin password: %w", err)
		}
		adminHash = string(hash)
	}

	// Validate the realtime payload before the lock so an invalid request fails
	// without a partial save (same as bcrypt above).
	if data.Realtime != nil {
		if err := s.validateRealtimeSet(*data.Realtime); err != nil {
			return err
		}
	}

	// All field mutations happen inside WithLockSave so they are marshalled
	// atomically — the watcher goroutine's SetLLMModel cannot interleave with
	// a partial config snapshot. Side-effect flags are captured inside the
	// closure so callers outside always see the post-save values.
	var (
		modelChanged    bool
		thinkingChanged bool
		baseURLChanged  bool
		wifiChanged     bool
		langChanged     bool
		voiceChanged    bool
		realtimeChanged bool
		channelChanged  bool
		newModel        string
		newSSID         string
		newPassword     string
		prevLang        string
		newLang         string
		chanReq         domain.AddChannelRequest
	)
	if err := s.config.WithLockSave(func(c *config.Config) {
		prevModel := c.LLMModel
		prevLang = c.STTLanguage
		// Snapshot voice-pipeline fields hal reads at boot (hal/server.py
		// :317-388 + hal/config.py:103-104). Used to gate hal restart
		// so wifi/channel/MQTT/admin-only saves don't bounce TTS.
		prevLLMAPIKey := c.LLMAPIKey
		prevLLMBaseURL := c.LLMBaseURL
		prevDeepgramAPIKey := c.DeepgramAPIKey
		prevSTTAPIKey := c.STTAPIKey
		prevTTSAPIKey := c.TTSAPIKey
		prevSTTBaseURL := c.STTBaseURL
		prevTTSBaseURL := c.TTSBaseURL
		prevTTSProvider := c.TTSProvider
		prevTTSVoice := c.TTSVoice
		// Snapshot channel fields so we can tell whether the messaging channel /
		// its tokens actually changed and need re-pushing into the gateway.
		prevChannel := c.Channel
		prevTelegramBotToken := c.TelegramBotToken
		prevTelegramUserID := c.TelegramUserID
		prevSlackBotToken := c.SlackBotToken
		prevSlackAppToken := c.SlackAppToken
		prevSlackUserID := c.SlackUserID
		prevDiscordBotToken := c.DiscordBotToken
		prevDiscordGuildID := c.DiscordGuildID
		prevDiscordUserID := c.DiscordUserID

		if data.LLMAPIKey != "" {
			c.LLMAPIKey = data.LLMAPIKey
		}
		if data.LLMBaseURL != "" {
			c.LLMBaseURL = urlnorm.NormalizeBaseURL(data.LLMBaseURL)
		}
		if data.LLMModel != "" {
			c.LLMModel = data.LLMModel
		}
		modelChanged = data.LLMModel != "" && data.LLMModel != prevModel
		baseURLChanged = data.LLMBaseURL != "" && c.LLMBaseURL != prevLLMBaseURL
		newModel = c.LLMModel

		thinkingChanged = data.LLMDisableThinking != nil
		if thinkingChanged {
			c.LLMDisableThinking = data.LLMDisableThinking
		}

		// PATCH semantics: empty = leave existing value alone. Stops the
		// Settings page (which ships its full form body even when the operator
		// only edited one tab) from wiping STT/TTS/Deepgram fields it never showed.
		if data.DeepgramAPIKey != "" {
			c.DeepgramAPIKey = data.DeepgramAPIKey
		}
		if data.STTAPIKey != "" {
			c.STTAPIKey = data.STTAPIKey
		}
		if data.TTSAPIKey != "" {
			c.TTSAPIKey = data.TTSAPIKey
		}
		if data.STTBaseURL != "" {
			c.STTBaseURL = urlnorm.NormalizeBaseURL(data.STTBaseURL)
		}
		if data.TTSBaseURL != "" {
			c.TTSBaseURL = urlnorm.NormalizeBaseURL(data.TTSBaseURL)
		}
		// Operators pick a language; the matching Deepgram SKU is auto-derived
		// because end users don't know which model handles which language.
		if data.STTLanguage != "" {
			c.STTLanguage = data.STTLanguage
			c.STTModel = sttModelForLanguage(data.STTLanguage)
		}
		newLang = c.STTLanguage
		langChanged = prevLang != newLang

		if data.TTSProvider != "" {
			c.TTSProvider = data.TTSProvider
		}
		if data.TTSVoice != "" {
			c.TTSVoice = data.TTSVoice
		}
		// Realtime block (validated above the lock). Sent = apply + restart hal.
		if data.Realtime != nil {
			applyRealtimeSet(c, *data.Realtime)
			realtimeChanged = true
		}
		if data.DeviceID != "" {
			c.DeviceID = data.DeviceID
		}
		wifiChanged = data.SSID != "" && data.SSID != c.NetworkSSID
		if data.SSID != "" {
			c.NetworkSSID = data.SSID
		}
		if data.Password != "" {
			c.NetworkPassword = data.Password
		}
		// Capture for the WiFi goroutine (avoid reading config after lock release).
		newSSID = c.NetworkSSID
		newPassword = c.NetworkPassword

		if data.Channel != "" {
			c.Channel = data.Channel
		}
		switch c.Channel {
		case domain.ChannelSlack:
			if data.SlackBotToken != "" {
				c.SlackBotToken = data.SlackBotToken
			}
			if data.SlackAppToken != "" {
				c.SlackAppToken = data.SlackAppToken
			}
			if data.SlackUserID != "" {
				c.SlackUserID = data.SlackUserID
			}
		case domain.ChannelDiscord:
			if data.DiscordBotToken != "" {
				c.DiscordBotToken = data.DiscordBotToken
			}
			if data.DiscordGuildID != "" {
				c.DiscordGuildID = data.DiscordGuildID
			}
			if data.DiscordUserID != "" {
				c.DiscordUserID = data.DiscordUserID
			}
		case domain.ChannelWhatsapp:
			if data.WhatsappUserID != "" {
				c.WhatsappUserID = data.WhatsappUserID
			}
		default:
			if data.TelegramBotToken != "" {
				c.TelegramBotToken = data.TelegramBotToken
			}
			if data.TelegramUserID != "" {
				c.TelegramUserID = data.TelegramUserID
			}
		}
		if data.MQTTEndpoint != "" {
			c.MQTTEndpoint = data.MQTTEndpoint
		}
		if data.MQTTUsername != "" {
			c.MQTTUsername = data.MQTTUsername
		}
		if data.MQTTPassword != "" {
			c.MQTTPassword = data.MQTTPassword
		}
		if data.MQTTPort != 0 {
			c.MQTTPort = data.MQTTPort
		}
		if data.FAChannel != "" {
			c.FAChannel = data.FAChannel
		}
		if data.FDChannel != "" {
			c.FDChannel = data.FDChannel
		}
		// Admin password rotation. Empty = keep existing hash; non-empty = bcrypt
		// + replace. Existing sessions stay valid (signed by SessionSecret), so
		// rotating the password alone won't lock the active operator out.
		if adminHash != "" {
			c.AdminPasswordHash = adminHash
		}

		voiceChanged = c.LLMAPIKey != prevLLMAPIKey ||
			c.LLMBaseURL != prevLLMBaseURL ||
			c.DeepgramAPIKey != prevDeepgramAPIKey ||
			c.STTAPIKey != prevSTTAPIKey ||
			c.TTSAPIKey != prevTTSAPIKey ||
			c.STTBaseURL != prevSTTBaseURL ||
			c.TTSBaseURL != prevTTSBaseURL ||
			c.TTSProvider != prevTTSProvider ||
			c.TTSVoice != prevTTSVoice

		channelChanged = c.Channel != prevChannel ||
			c.TelegramBotToken != prevTelegramBotToken || c.TelegramUserID != prevTelegramUserID ||
			c.SlackBotToken != prevSlackBotToken || c.SlackAppToken != prevSlackAppToken || c.SlackUserID != prevSlackUserID ||
			c.DiscordBotToken != prevDiscordBotToken || c.DiscordGuildID != prevDiscordGuildID || c.DiscordUserID != prevDiscordUserID
		// Build the request from the post-save config (full current values, since
		// PATCH semantics mean a token the operator didn't touch keeps its value).
		chanReq = domain.AddChannelRequest{
			Channel:          c.Channel,
			TelegramBotToken: c.TelegramBotToken, TelegramUserID: c.TelegramUserID,
			SlackBotToken: c.SlackBotToken, SlackAppToken: c.SlackAppToken, SlackUserID: c.SlackUserID,
			DiscordBotToken: c.DiscordBotToken, DiscordGuildID: c.DiscordGuildID, DiscordUserID: c.DiscordUserID,
		}
	}); err != nil {
		return fmt.Errorf("save config: %w", err)
	}
	slog.Info("config updated", "component", "device")
	if wifiChanged {
		go func() {
			slog.Info("reconnecting to new WiFi", "component", "device", "ssid", newSSID)
			if _, err := s.networkService.SetupNetwork(newSSID, newPassword); err != nil {
				slog.Error("WiFi reconnect failed", "component", "device", "error", err)
			}
		}()
	}
	// Channel/token edits via the Settings form only land in config.json, but the
	// gateway keeps messaging tokens in its OWN config (written by AddChannel:
	// `openclaw channels add` + plugin enable) and reads from there first. So a
	// save — even a gateway restart — won't apply them; re-run AddChannel to push
	// the change into the gateway. WhatsApp needs interactive QR pairing (MQTT
	// add_channel only), so it is excluded here.
	if channelChanged && chanReq.Channel != domain.ChannelWhatsapp {
		go func() {
			if _, err := s.AddChannel(context.Background(), chanReq); err != nil {
				slog.Error("apply channel change to gateway failed", "component", "device", "channel", chanReq.Channel, "error", err)
			} else {
				slog.Info("channel change applied to gateway", "component", "device", "channel", chanReq.Channel)
			}
		}()
	}
	// Sync primary model into openclaw.json (os-server → OpenClaw direction).
	// config.mu is released by WithLockSave above; openclaw calls now acquire
	// primarySyncMu without risk of deadlock (consistent lock order).
	// When thinking also changed, RefreshModelsConfig handles primary update +
	// reasoning patch in a single write + restart — skip UpdatePrimaryModel to
	// avoid a redundant gateway restart.
	if modelChanged && !thinkingChanged && !baseURLChanged && s.agentGateway != nil {
		if err := s.agentGateway.UpdatePrimaryModel(newModel); err != nil {
			if errors.Is(err, domain.ErrNotSupportedByRuntime) {
				// hermes/picoclaw: the device model is not what the runtime runs
				// on, so there is nothing to apply — informational, not a failure.
				slog.Info("primary model sync not supported by runtime", "component", "device", "backend", s.agentGateway.Name())
			} else {
				slog.Warn("update primary model failed", "component", "device", "error", err)
			}
		}
	}
	if (thinkingChanged || baseURLChanged) && s.agentGateway != nil {
		// RefreshModelsConfig syncs agents.defaults.model.primary, per-model
		// reasoning, and providers.autonomous.baseUrl in one write + restart.
		if err := s.agentGateway.RefreshModelsConfig(); err != nil {
			if errors.Is(err, domain.ErrNotSupportedByRuntime) {
				// hermes/picoclaw can't be patched directly, but hermes's
				// EnsureOnboarding presync re-reads llm_base_url/llm_api_key from
				// the config.json we just saved — run it so a baseURL/key change
				// applies now instead of at the next boot. Idempotent on both
				// backends; restarts the gateway only when the config changed.
				slog.Info("models config not device-patchable, running onboarding self-heal", "component", "device", "backend", s.agentGateway.Name())
				go func() {
					if err := s.agentGateway.EnsureOnboarding(); err != nil {
						slog.Warn("onboarding self-heal after llm config change failed", "component", "device", "error", err)
					}
				}()
			} else {
				slog.Error("refresh models config failed", "component", "device", "error", err)
			}
		}
	}
	// When the operator switches stt_language explicitly, drop the in-session
	// chat history so the LLM doesn't keep replying in the previous language
	// out of inertia. SOUL.md tells it the latest turn wins, but a heavily
	// English/Vietnamese-biased history can still pull the next reply back —
	// a fresh session is the cleanest break.
	if langChanged && s.agentGateway != nil {
		if key := s.agentGateway.GetSessionKey(); key != "" {
			go func() {
				if err := s.agentGateway.NewSession(key); err != nil {
					slog.Warn("openclaw NewSession on stt_language change failed", "component", "device", "error", err)
				} else {
					slog.Info("openclaw session reset for stt_language change", "component", "device", "from", prevLang, "to", newLang)
				}
			}()
		}
	}
	// Restart hal only when a field it reads at boot actually changed.
	// stt_language is covered by langChanged (hal reads it via stt_language /
	// derived stt_model). Wifi/channel/MQTT/admin saves skip the restart.
	if voiceChanged || langChanged || realtimeChanged {
		s.restartHAL("voice config change")
	}
	return nil
}

// UpdateVoiceConfig updates only TTS provider/voice and STT language — safe to call from MQTT
// handlers since it does not touch API keys, MQTT credentials, or WiFi config.
func (s *Service) UpdateVoiceConfig(provider, voice, language string) error {
	prevLang := s.config.STTLanguage
	if provider != "" {
		s.config.TTSProvider = provider
	}
	if voice != "" {
		s.config.TTSVoice = voice
	}
	if language != "" {
		s.config.STTLanguage = language
		s.config.STTModel = sttModelForLanguage(language)
	}
	if err := s.config.Save(); err != nil {
		return fmt.Errorf("save config: %w", err)
	}
	slog.Info("voice config updated", "component", "device", "provider", s.config.TTSProvider, "voice", s.config.TTSVoice, "language", s.config.STTLanguage)
	if language != "" && prevLang != s.config.STTLanguage && s.agentGateway != nil {
		if key := s.agentGateway.GetSessionKey(); key != "" {
			go func() {
				if err := s.agentGateway.NewSession(key); err != nil {
					slog.Warn("NewSession on language change failed", "component", "device", "error", err)
				}
			}()
		}
	}
	s.restartHAL("voice config change")
	return nil
}

// sttModelForLanguage maps a BCP-47 language code to the Deepgram SKU exposed
// by the Autonomous STT proxy. Empty input → empty model so hal falls back
// to its built-in default (flux-general-en). English stays on Flux (Deepgram's
// conversational EN model — EN-only). Everything else rides on Nova-3:
// Vietnamese since Jan 2026, Mandarin (zh/zh-CN/zh-Hans/zh-TW/zh-Hant) since
// Deepgram's 2026-03-31 release — zh was pinned to Nova-2 before that.
func sttModelForLanguage(lang string) string {
	switch lang {
	case "":
		return ""
	case i18n.LangEN:
		return "flux-general-en"
	default:
		return "nova-3-general"
	}
}
