package http

import (
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"go.autonomous.ai/os/system/googleauth"
	"go.autonomous.ai/os/system/server/serializers"
)

type googleClientBody struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

type pendingGoogleLogin struct {
	DC      googleauth.DeviceCode
	Expires time.Time
}

var googleLogins sync.Map

// GetGoogleStatus is GET /api/device/google.
func (h *DeviceHandler) GetGoogleStatus(c *gin.Context) {
	c.JSON(http.StatusOK, serializers.ResponseSuccess(h.service.GoogleStatus()))
}

// SetGoogleClient is PUT /api/device/google/client.
func (h *DeviceHandler) SetGoogleClient(c *gin.Context) {
	var body googleClientBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, serializers.ResponseError(err.Error()))
		return
	}
	if err := h.service.SetGoogleOAuthClient(body.ClientID, body.ClientSecret); err != nil {
		c.JSON(http.StatusBadRequest, serializers.ResponseError(err.Error()))
		return
	}
	c.JSON(http.StatusOK, serializers.ResponseSuccess(h.service.GoogleStatus()))
}

// StartGoogleOAuth is POST /api/device/google/start.
func (h *DeviceHandler) StartGoogleOAuth(c *gin.Context) {
	cli := h.serviceGoogleClient()
	if cli == nil {
		c.JSON(http.StatusBadRequest, serializers.ResponseError("add a Google OAuth client (TV / limited input) first"))
		return
	}
	dc, err := cli.RequestDeviceCode()
	if err != nil {
		slog.Warn("google oauth start failed", "component", "device", "error", err)
		c.JSON(http.StatusBadGateway, serializers.ResponseError(err.Error()))
		return
	}
	googleLogins.Store(dc.DeviceCode, pendingGoogleLogin{DC: dc, Expires: time.Now().Add(dc.ExpiresIn)})
	c.JSON(http.StatusOK, serializers.ResponseSuccess(gin.H{
		"user_code":                 dc.UserCode,
		"device_code":               dc.DeviceCode,
		"verification_uri":          dc.VerificationURI,
		"verification_uri_complete": dc.VerificationURIComplete,
		"expires_in":                int(dc.ExpiresIn.Seconds()),
		"interval":                  int(dc.Interval.Seconds()),
	}))
}

type googlePollBody struct {
	DeviceCode string `json:"device_code"`
}

// PollGoogleOAuth is POST /api/device/google/poll.
func (h *DeviceHandler) PollGoogleOAuth(c *gin.Context) {
	var body googlePollBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, serializers.ResponseError(err.Error()))
		return
	}
	raw, ok := googleLogins.Load(body.DeviceCode)
	if !ok {
		c.JSON(http.StatusNotFound, serializers.ResponseError("unknown or expired device login"))
		return
	}
	pending := raw.(pendingGoogleLogin)
	if time.Now().After(pending.Expires) {
		googleLogins.Delete(body.DeviceCode)
		c.JSON(http.StatusGone, serializers.ResponseError("device code expired — start again"))
		return
	}
	cli := h.serviceGoogleClient()
	if cli == nil {
		c.JSON(http.StatusBadRequest, serializers.ResponseError("Google OAuth client is not set"))
		return
	}
	tok, wait, err := cli.Exchange(pending.DC)
	if err != nil {
		if googleauth.TerminalDeviceError(err) {
			googleLogins.Delete(body.DeviceCode)
		}
		c.JSON(http.StatusBadGateway, serializers.ResponseError(err.Error()))
		return
	}
	if wait != "" {
		if wait == "slow_down" {
			pending.DC.Interval += googleauth.DeviceCodeSlowDownStep
			googleLogins.Store(body.DeviceCode, pending)
		}
		c.JSON(http.StatusOK, serializers.ResponseSuccess(gin.H{
			"pending":  true,
			"interval": int(pending.DC.Interval.Seconds()),
		}))
		return
	}
	info, _ := cli.LookupUser(tok.AccessToken)
	if err := h.service.ApplyGoogleTokens(tok, info); err != nil {
		c.JSON(http.StatusInternalServerError, serializers.ResponseError(err.Error()))
		return
	}
	googleLogins.Delete(body.DeviceCode)
	c.JSON(http.StatusOK, serializers.ResponseSuccess(gin.H{
		"pending":    false,
		"connected":  true,
		"user_email": info.Email,
	}))
}

func (h *DeviceHandler) serviceGoogleClient() *googleauth.Client {
	// device.Service.googleClient is unexported; reconstruct from status+config.
	st := h.service.GoogleStatus()
	if !st.Ready {
		return nil
	}
	return h.service.GoogleOAuthClient()
}
