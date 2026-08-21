package http

import (
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"go.autonomous.ai/os/system/appleauth"
	"go.autonomous.ai/os/system/server/config"
	"go.autonomous.ai/os/system/server/serializers"
)

var appleLogins sync.Map

func (h *DeviceHandler) GetBuzz(c *gin.Context) {
	c.JSON(http.StatusOK, serializers.ResponseSuccess(h.service.BuzzStatus()))
}

type buzzSetBody struct {
	Enabled  bool   `json:"enabled"`
	Host     bool   `json:"host"`
	RelayURL string `json:"relay_url"`
}

func (h *DeviceHandler) SetBuzz(c *gin.Context) {
	var body buzzSetBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, serializers.ResponseError(err.Error()))
		return
	}
	if err := h.service.SetBuzz(config.Buzz{Enabled: body.Enabled, Host: body.Host, RelayURL: strings.TrimSpace(body.RelayURL)}); err != nil {
		c.JSON(http.StatusBadRequest, serializers.ResponseError(err.Error()))
		return
	}
	c.JSON(http.StatusOK, serializers.ResponseSuccess(h.service.BuzzStatus()))
}

type buzzSayBody struct {
	Text string `json:"text"`
}

func (h *DeviceHandler) SayBuzz(c *gin.Context) {
	var body buzzSayBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, serializers.ResponseError(err.Error()))
		return
	}
	if err := h.service.SayBuzz(body.Text); err != nil {
		c.JSON(http.StatusBadRequest, serializers.ResponseError(err.Error()))
		return
	}
	c.JSON(http.StatusOK, serializers.ResponseSuccess(true))
}

func (h *DeviceHandler) BuzzWS(c *gin.Context) {
	hub := h.service.BuzzHub()
	if hub == nil {
		c.JSON(http.StatusServiceUnavailable, serializers.ResponseError("this robot is not hosting the hive"))
		return
	}
	hub.ServeWS(c.Writer, c.Request)
}

func (h *DeviceHandler) GetMatter(c *gin.Context) {
	c.JSON(http.StatusOK, serializers.ResponseSuccess(h.service.MatterStatus()))
}

type matterBody struct {
	Code string `json:"code"`
}

func (h *DeviceHandler) CommissionMatter(c *gin.Context) {
	var body matterBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, serializers.ResponseError(err.Error()))
		return
	}
	if err := h.service.CommissionMatter(body.Code); err != nil {
		c.JSON(http.StatusBadRequest, serializers.ResponseError(err.Error()))
		return
	}
	c.JSON(http.StatusOK, serializers.ResponseSuccess(true))
}

func (h *DeviceHandler) GetApple(c *gin.Context) {
	c.JSON(http.StatusOK, serializers.ResponseSuccess(h.service.AppleStatus()))
}

type appleClientBody struct {
	ServicesID string `json:"services_id"`
	TeamID     string `json:"team_id"`
	KeyID      string `json:"key_id"`
	PrivateKey string `json:"private_key"`
	ReturnURL  string `json:"return_url"`
}

func (h *DeviceHandler) SetAppleClient(c *gin.Context) {
	var body appleClientBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, serializers.ResponseError(err.Error()))
		return
	}
	if err := h.service.SetAppleClient(body.ServicesID, body.TeamID, body.KeyID, body.PrivateKey, body.ReturnURL); err != nil {
		c.JSON(http.StatusBadRequest, serializers.ResponseError(err.Error()))
		return
	}
	h.GetApple(c)
}

func (h *DeviceHandler) StartAppleOAuth(c *gin.Context) {
	cli := h.service.AppleOAuthClient()
	if cli == nil {
		c.JSON(http.StatusBadRequest, serializers.ResponseError("add a Sign in with Apple Services ID first"))
		return
	}
	if err := cli.Ready(); err != nil {
		c.JSON(http.StatusBadRequest, serializers.ResponseError(err.Error()))
		return
	}
	state := appleauth.NewState()
	u, err := cli.AuthorizeURL(state)
	if err != nil {
		c.JSON(http.StatusBadRequest, serializers.ResponseError(err.Error()))
		return
	}
	appleLogins.Store(state, time.Now().Add(10*time.Minute))
	c.JSON(http.StatusOK, serializers.ResponseSuccess(gin.H{"url": u, "state": state}))
}

func (h *DeviceHandler) AppleCallback(c *gin.Context) {
	code := strings.TrimSpace(c.PostForm("code"))
	if code == "" {
		code = strings.TrimSpace(c.Query("code"))
	}
	state := strings.TrimSpace(c.PostForm("state"))
	if state == "" {
		state = strings.TrimSpace(c.Query("state"))
	}
	errDesc := strings.TrimSpace(c.PostForm("error"))
	if errDesc == "" {
		errDesc = strings.TrimSpace(c.Query("error"))
	}
	if errDesc != "" {
		c.String(http.StatusBadRequest, "Apple sign-in was cancelled.")
		return
	}
	if _, ok := appleLogins.Load(state); !ok {
		c.String(http.StatusBadRequest, "Apple sign-in expired. Start again from this robot.")
		return
	}
	appleLogins.Delete(state)
	cli := h.service.AppleOAuthClient()
	if cli == nil {
		c.String(http.StatusBadRequest, "Apple client is not configured.")
		return
	}
	id, err := cli.Exchange(code)
	if err != nil {
		slog.Warn("apple oauth exchange", "component", "device", "error", err)
		c.String(http.StatusBadGateway, "Apple sign-in failed.")
		return
	}
	if name := strings.TrimSpace(c.PostForm("user")); name != "" && id.Name == "" {
		id.Name = name
	}
	if err := h.service.ApplyAppleIdentity(id); err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, "<!doctype html><p>Signed in with Apple. You can close this tab.</p>")
}
