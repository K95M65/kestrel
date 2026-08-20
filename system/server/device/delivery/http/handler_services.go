package http

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.autonomous.ai/os/system/device"
	"go.autonomous.ai/os/system/server/serializers"
)

// ListServices godoc
//
//	@Summary	secret-free status of Gmail, Calendar, Telegram
//	@Tags		device
//	@Success	200	{object}	serializers.ResponseSuccess
//	@Router		/device/services [get]
func (h *DeviceHandler) ListServices(c *gin.Context) {
	c.JSON(http.StatusOK, serializers.ResponseSuccess(h.service.ListServices()))
}

type setGmailBody struct {
	Email string `json:"email"`
	Key   string `json:"api_key"`
}

// SetGmail godoc
//
//	@Summary	store a Gmail app password (PAT) for overnight mail
//	@Tags		device
//	@Router		/device/connectors/gmail [put]
func (h *DeviceHandler) SetGmail(c *gin.Context) {
	var body setGmailBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, serializers.ResponseError(err.Error()))
		return
	}
	if err := h.service.SetGmailPAT(device.SetConnectorPAT{
		Code:  "gmail",
		Email: body.Email,
		Key:   body.Key,
	}); err != nil {
		c.JSON(http.StatusBadRequest, serializers.ResponseError(err.Error()))
		return
	}
	c.JSON(http.StatusOK, serializers.ResponseSuccess(h.service.ListServices()))
}

type setCalendarBody struct {
	URL string `json:"url"`
}

// SetCalendar godoc
//
//	@Summary	store a Google Calendar secret iCal URL
//	@Router		/device/connectors/google_calendar [put]
func (h *DeviceHandler) SetCalendar(c *gin.Context) {
	var body setCalendarBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, serializers.ResponseError(err.Error()))
		return
	}
	if err := h.service.SetCalendarICS(body.URL); err != nil {
		c.JSON(http.StatusBadRequest, serializers.ResponseError(err.Error()))
		return
	}
	c.JSON(http.StatusOK, serializers.ResponseSuccess(h.service.ListServices()))
}

// RemoveConnector godoc
//
//	@Router	/device/connectors/:code [delete]
func (h *DeviceHandler) RemoveConnector(c *gin.Context) {
	code := strings.TrimSpace(c.Param("code"))
	if code != "gmail" && code != "google_calendar" {
		c.JSON(http.StatusBadRequest, serializers.ResponseError("unknown connector"))
		return
	}
	if err := h.service.RemoveConnector(code); err != nil {
		c.JSON(http.StatusInternalServerError, serializers.ResponseError(err.Error()))
		return
	}
	c.JSON(http.StatusOK, serializers.ResponseSuccess(h.service.ListServices()))
}

type setTelegramBody struct {
	Token  string `json:"telegram_bot_token"`
	UserID string `json:"telegram_user_id"`
}

// SetTelegram godoc
//
//	@Router	/device/services/telegram [put]
func (h *DeviceHandler) SetTelegram(c *gin.Context) {
	var body setTelegramBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, serializers.ResponseError(err.Error()))
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()
	if err := h.service.SetTelegram(ctx, body.Token, body.UserID); err != nil {
		c.JSON(http.StatusBadRequest, serializers.ResponseError(err.Error()))
		return
	}
	c.JSON(http.StatusOK, serializers.ResponseSuccess(h.service.ListServices()))
}
