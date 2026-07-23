package http

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"go.autonomous.ai/os/system/domain"
	"go.autonomous.ai/os/system/plugin"
	"go.autonomous.ai/os/system/server/serializers"
)

type PluginHandler struct {
	service *plugin.Service
}

func ProvidePluginHandler(ps *plugin.Service) PluginHandler {
	return PluginHandler{service: ps}
}

// Install handles POST /api/plugin/install. Clones, sets up venv, and creates
// a systemd unit. Runs async — returns 200 immediately. Poll List for status.
func (h *PluginHandler) Install(c *gin.Context) {
	var req domain.PluginInstallRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, serializers.ResponseError(err.Error()))
		return
	}
	if req.URL == "" {
		c.JSON(http.StatusBadRequest, serializers.ResponseError("url is required"))
		return
	}

	go func() {
		if _, err := h.service.Install(req.URL); err != nil {
			slog.Error("[plugins] install failed", "component", "plugin-http", "url", req.URL, "error", err)
		}
	}()

	c.JSON(http.StatusOK, serializers.ResponseSuccess(true))
}

// List handles GET /api/plugin. Returns all installed plugins.
func (h *PluginHandler) List(c *gin.Context) {
	c.JSON(http.StatusOK, serializers.ResponseSuccess(h.service.List()))
}

// Start handles POST /api/plugin/:name/start.
func (h *PluginHandler) Start(c *gin.Context) {
	name := c.Param("name")
	if err := h.service.Start(name); err != nil {
		c.JSON(http.StatusBadRequest, serializers.ResponseError(err.Error()))
		return
	}
	c.JSON(http.StatusOK, serializers.ResponseSuccess(true))
}

// Stop handles POST /api/plugin/:name/stop.
func (h *PluginHandler) Stop(c *gin.Context) {
	name := c.Param("name")
	if err := h.service.Stop(name); err != nil {
		c.JSON(http.StatusBadRequest, serializers.ResponseError(err.Error()))
		return
	}
	c.JSON(http.StatusOK, serializers.ResponseSuccess(true))
}

// Uninstall handles DELETE /api/plugin/:name.
func (h *PluginHandler) Uninstall(c *gin.Context) {
	name := c.Param("name")
	if err := h.service.Uninstall(name); err != nil {
		c.JSON(http.StatusBadRequest, serializers.ResponseError(err.Error()))
		return
	}
	c.JSON(http.StatusOK, serializers.ResponseSuccess(true))
}
