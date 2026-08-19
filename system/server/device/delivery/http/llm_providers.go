package http

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
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
	pruneExpiredLLMLogins()
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
		if grokauth.TerminalDeviceError(err) {
			llmLogins.Delete(req.DeviceCode)
		}
		c.JSON(http.StatusBadGateway, serializers.ResponseError(err.Error()))
		return
	}
	if wait != "" {
		if wait == "slow_down" {
			pending.DC.Interval += grokauth.DeviceCodeSlowDownStep
			llmLogins.Store(req.DeviceCode, pending)
		}
		c.JSON(http.StatusOK, serializers.ResponseSuccess(gin.H{
			"pending":  true,
			"interval": int(pending.DC.Interval.Seconds()),
		}))
		return
	}
	llmLogins.Delete(req.DeviceCode)
	if err := writeGrokTokens(grokTokFile, tok); err != nil {
		slog.Warn("grok oauth persist failed", "component", "device", "error", err)
		c.JSON(http.StatusInternalServerError, serializers.ResponseError("could not save Grok login"))
		return
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

func pruneExpiredLLMLogins() {
	now := time.Now()
	llmLogins.Range(func(k, v any) bool {
		p, ok := v.(pendingLLMLogin)
		if ok && now.After(p.Expires) {
			llmLogins.Delete(k)
		}
		return true
	})
}

// GetCompanionApps returns downloadable pairing apps for onboarding.
func (h *DeviceHandler) GetCompanionApps(c *gin.Context) {
	c.JSON(http.StatusOK, serializers.ResponseSuccess(domain.CompanionApps()))
}

const grokRefreshInterval = 5 * time.Minute

type grokTokRecord struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	ExpiresAt    int64  `json:"expires_at"`
	SavedAt      string `json:"saved_at"`
	TokenType    string `json:"token_type"`
}

func writeGrokTokens(path string, tok grokauth.Tokens) error {
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	now := time.Now().UTC()
	expiresIn := int(tok.ExpiresIn.Seconds())
	if expiresIn <= 0 {
		expiresIn = int(time.Hour.Seconds())
	}
	body, err := json.MarshalIndent(map[string]any{
		"access_token":  tok.AccessToken,
		"refresh_token": tok.RefreshToken,
		"expires_in":    expiresIn,
		"expires_at":    now.Add(time.Duration(expiresIn) * time.Second).Unix(),
		"token_type":    tok.TokenType,
		"saved_at":      now.Format(time.RFC3339),
	}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, body, 0o600)
}

func readGrokTokens(path string) (grokTokRecord, error) {
	var rec grokTokRecord
	data, err := os.ReadFile(path)
	if err != nil {
		return rec, err
	}
	if err := json.Unmarshal(data, &rec); err != nil {
		return rec, err
	}
	return rec, nil
}

func (r grokTokRecord) expiry() time.Time {
	if r.ExpiresAt > 0 {
		return time.Unix(r.ExpiresAt, 0)
	}
	if r.SavedAt != "" && r.ExpiresIn > 0 {
		if t, err := time.Parse(time.RFC3339, r.SavedAt); err == nil {
			return t.Add(time.Duration(r.ExpiresIn) * time.Second)
		}
	}
	return time.Time{}
}

func (r grokTokRecord) needsRefresh(now time.Time) bool {
	if grokauth.AccessTokenIsExpiring(r.AccessToken, grokauth.AccessTokenRefreshSkew, now) {
		return true
	}
	if exp := r.expiry(); !exp.IsZero() {
		return now.After(exp.Add(-grokauth.AccessTokenRefreshSkew))
	}
	if r.SavedAt != "" {
		if t, err := time.Parse(time.RFC3339, r.SavedAt); err == nil {
			return now.Sub(t) > 50*time.Minute
		}
	}
	return false
}

// StartGrokRefreshLoop rotates SuperGrok tokens before the access token dies.
func (h *DeviceHandler) StartGrokRefreshLoop(ctx context.Context) {
	h.refreshGrokTokens()
	t := time.NewTicker(grokRefreshInterval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			h.refreshGrokTokens()
		}
	}
}

func (h *DeviceHandler) refreshGrokTokens() {
	defer func() {
		if rec := recover(); rec != nil {
			slog.Error("grok-refresh panic", "component", "device", "panic", rec)
		}
	}()
	rec, err := readGrokTokens(grokTokFile)
	if err != nil || strings.TrimSpace(rec.RefreshToken) == "" {
		return
	}
	if !rec.needsRefresh(time.Now()) {
		return
	}
	tok, err := grokClient.Refresh(rec.RefreshToken)
	if err != nil {
		slog.Warn("grok token refresh failed", "component", "device", "error", err)
		return
	}
	if err := writeGrokTokens(grokTokFile, tok); err != nil {
		slog.Warn("grok oauth persist failed", "component", "device", "error", err)
	}
	if h.service != nil {
		if err := h.service.ApplyRotatedLLMAPIKey(rec.AccessToken, tok.AccessToken); err != nil {
			slog.Warn("grok token apply failed", "component", "device", "error", err)
		}
	}
}
