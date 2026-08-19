package http

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"go.autonomous.ai/os/system/domain"
	"go.autonomous.ai/os/system/grokauth"
	"go.autonomous.ai/os/system/server/serializers"
)

// GetLLMProviders returns the OpenCode-shaped provider catalog for setup/Settings.
func (h *DeviceHandler) GetLLMProviders(c *gin.Context) {
	c.JSON(http.StatusOK, serializers.ResponseSuccess(domain.ListProviders))
}

type llmOAuthStartRequest struct {
	Provider string `json:"provider"`
}

type llmOAuthPollRequest struct {
	Provider   string `json:"provider"`
	DeviceCode string `json:"device_code"`
}

type pendingLLMLogin struct {
	Provider string
	DC       grokauth.DeviceCode
	Expires  time.Time
}

var (
	llmLogins   sync.Map
	grokClient  = grokauth.New()
	grokTokFile = "/root/config/grok-oauth.json"
)

// StartLLMOAuth begins a device-code login (xAI SuperGrok today).
func (h *DeviceHandler) StartLLMOAuth(c *gin.Context) {
	var req llmOAuthStartRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, serializers.ResponseError(err.Error()))
		return
	}
	p, ok := domain.LookupLLMProvider(req.Provider)
	if !ok || p.Auth != domain.LLMAuthDeviceCode {
		c.JSON(http.StatusBadRequest, serializers.ResponseError("provider does not support device login"))
		return
	}
	if p.Key != "xai" {
		c.JSON(http.StatusNotImplemented, serializers.ResponseError("device login for this provider is not implemented yet"))
		return
	}
	dc, err := grokClient.RequestDeviceCode()
	if err != nil {
		slog.Warn("llm oauth start failed", "component", "device", "provider", p.Key, "error", err)
		c.JSON(http.StatusBadGateway, serializers.ResponseError(err.Error()))
		return
	}
	llmLogins.Store(dc.DeviceCode, pendingLLMLogin{
		Provider: p.Key,
		DC:       dc,
		Expires:  time.Now().Add(dc.ExpiresIn),
	})
	c.JSON(http.StatusOK, serializers.ResponseSuccess(gin.H{
		"provider":                  p.Key,
		"user_code":                 dc.UserCode,
		"device_code":               dc.DeviceCode,
		"verification_uri":          dc.VerificationURI,
		"verification_uri_complete": dc.VerificationURIComplete,
		"expires_in":                int(dc.ExpiresIn.Seconds()),
		"interval":                  int(dc.Interval.Seconds()),
		"base_url":                  p.BaseURL,
		"default_model":             p.DefaultModel,
	}))
}

// PollLLMOAuth is one token attempt. The browser calls this until pending=false.
func (h *DeviceHandler) PollLLMOAuth(c *gin.Context) {
	var req llmOAuthPollRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, serializers.ResponseError(err.Error()))
		return
	}
	raw, ok := llmLogins.Load(req.DeviceCode)
	if !ok {
		c.JSON(http.StatusNotFound, serializers.ResponseError("unknown or expired device login"))
		return
	}
	pending := raw.(pendingLLMLogin)
	if time.Now().After(pending.Expires) {
		llmLogins.Delete(req.DeviceCode)
		c.JSON(http.StatusGone, serializers.ResponseError("device code expired — start again"))
		return
	}
	tok, wait, err := grokClient.Exchange(pending.DC)
	if err != nil {
		llmLogins.Delete(req.DeviceCode)
		c.JSON(http.StatusBadGateway, serializers.ResponseError(err.Error()))
		return
	}
	if wait != "" {
		c.JSON(http.StatusOK, serializers.ResponseSuccess(gin.H{"pending": true}))
		return
	}
	llmLogins.Delete(req.DeviceCode)
	if err := writeGrokTokens(grokTokFile, tok); err != nil {
		slog.Warn("grok oauth persist failed", "component", "device", "error", err)
	}
	p, _ := domain.LookupLLMProvider(pending.Provider)
	c.JSON(http.StatusOK, serializers.ResponseSuccess(gin.H{
		"pending":       false,
		"provider":      pending.Provider,
		"access_token":  tok.AccessToken,
		"base_url":      p.BaseURL,
		"default_model": p.DefaultModel,
	}))
}

func writeGrokTokens(path string, tok grokauth.Tokens) error {
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	body, err := json.MarshalIndent(map[string]any{
		"access_token":  tok.AccessToken,
		"refresh_token": tok.RefreshToken,
		"expires_in":    int(tok.ExpiresIn.Seconds()),
		"token_type":    tok.TokenType,
		"saved_at":      time.Now().UTC().Format(time.RFC3339),
	}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, body, 0o600)
}
