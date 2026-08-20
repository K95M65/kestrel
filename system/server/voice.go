package server

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"go.autonomous.ai/os/system/lib/hal"
	"go.autonomous.ai/os/system/server/serializers"
)

// voicePreview plays a TTS preview through HAL using server-side
// credentials. Body: {text, voice, provider}. The TTS API key + base URL
// come from cfg (with the same LLM-fallback the runtime voice pipeline
// uses) — they never leave the device. Audit web F13: previous flow
// shipped tts_api_key in the request body straight to /hw/voice/speak.
func (s *Server) voicePreview(c *gin.Context) {
	var body struct {
		Text     string `json:"text"`
		Voice    string `json:"voice"`
		Provider string `json:"provider"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || strings.TrimSpace(body.Text) == "" {
		c.JSON(http.StatusBadRequest, serializers.ResponseError("text required"))
		return
	}
	apiKey := s.config.GetTTSAPIKey()
	baseURL := s.config.GetTTSBaseURL()
	if err := hal.SpeakPreview(body.Text, body.Voice, body.Provider, apiKey, baseURL); err != nil {
		if errors.Is(err, hal.ErrSpeakerMuted) {
			c.JSON(http.StatusConflict, serializers.ResponseError("speaker muted"))
			return
		}
		if strings.Contains(err.Error(), "returned 409") {
			c.JSON(http.StatusConflict, serializers.ResponseError("robot is busy speaking — try again in a moment"))
			return
		}
		slog.Warn("voice preview failed", "component", "voice", "error", err)
		c.JSON(http.StatusBadGateway, serializers.ResponseError("preview failed: "+err.Error()))
		return
	}
	c.JSON(http.StatusOK, serializers.ResponseSuccess(true))
}

// voicePreviewAudio renders the same phrase as voicePreview and returns WAV
// bytes for playback in the operator's browser. Credentials stay on the
// device (same F13 rule as voicePreview). Mute does not apply.
func (s *Server) voicePreviewAudio(c *gin.Context) {
	var body struct {
		Text     string `json:"text"`
		Voice    string `json:"voice"`
		Provider string `json:"provider"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || strings.TrimSpace(body.Text) == "" {
		c.JSON(http.StatusBadRequest, serializers.ResponseError("text required"))
		return
	}
	apiKey := s.config.GetTTSAPIKey()
	baseURL := s.config.GetTTSBaseURL()
	audio, ctype, err := hal.SpeakPreviewAudio(body.Text, body.Voice, body.Provider, apiKey, baseURL)
	if err != nil {
		slog.Warn("voice preview-audio failed", "component", "voice", "error", err)
		c.JSON(http.StatusBadGateway, serializers.ResponseError("preview failed: "+err.Error()))
		return
	}
	if ctype == "" {
		ctype = "audio/wav"
	}
	c.Data(http.StatusOK, ctype, audio)
}
