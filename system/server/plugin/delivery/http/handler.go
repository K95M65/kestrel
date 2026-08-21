package http

import (
	// "encoding/json"  // Browse only — restore with it (#213)
	// "io"             // Browse only
	"log/slog"
	"net/http"
	// "time"           // Browse only

	"strings"

	"github.com/gin-gonic/gin"

	"go.autonomous.ai/os/system/domain"
	"go.autonomous.ai/os/system/plugin"
	"go.autonomous.ai/os/system/server/serializers"
	"go.autonomous.ai/os/system/skillcontext"
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
	if req.ID != "" {
		app, ok := domain.LookupTrustedPlugin(req.ID)
		if !ok {
			c.JSON(http.StatusBadRequest, serializers.ResponseError("unknown trusted plugin"))
			return
		}
		req.URL = app.InstallURL
		req.Subdir = app.Subdir
	}
	if req.URL == "" {
		c.JSON(http.StatusBadRequest, serializers.ResponseError("url or id is required"))
		return
	}

	go func() {
		if _, err := h.service.InstallFrom(req); err != nil {
			slog.Error("[plugins] install failed", "component", "plugin-http", "url", req.URL, "subdir", req.Subdir, "error", err)
		}
	}()

	c.JSON(http.StatusOK, serializers.ResponseSuccess(true))
}

// List handles GET /api/plugin. Returns all installed plugins.
func (h *PluginHandler) List(c *gin.Context) {
	c.JSON(http.StatusOK, serializers.ResponseSuccess(h.service.List()))
}

type pluginNameBody struct {
	Name string `json:"name"`
}

// StartBody is POST /api/plugin/start `{name}` — loopback from [HW:/plugin/start:…].
func (h *PluginHandler) StartBody(c *gin.Context) {
	var body pluginNameBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, serializers.ResponseError(err.Error()))
		return
	}
	h.startNamed(c, body.Name)
}

// StopBody is POST /api/plugin/stop `{name}` — loopback from [HW:/plugin/stop:…].
func (h *PluginHandler) StopBody(c *gin.Context) {
	var body pluginNameBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, serializers.ResponseError(err.Error()))
		return
	}
	if err := h.service.Stop(strings.TrimSpace(body.Name)); err != nil {
		c.JSON(http.StatusBadRequest, serializers.ResponseError(err.Error()))
		return
	}
	c.JSON(http.StatusOK, serializers.ResponseSuccess(true))
}

// Start handles POST /api/plugin/:name/start.
func (h *PluginHandler) Start(c *gin.Context) {
	h.startNamed(c, c.Param("name"))
}

func (h *PluginHandler) startNamed(c *gin.Context, name string) {
	name = strings.TrimSpace(name)
	if name == "" {
		c.JSON(http.StatusBadRequest, serializers.ResponseError("plugin name is required"))
		return
	}
	if plugin.CameraExclusive(name) && skillcontext.KidsBound() {
		c.JSON(http.StatusForbidden, serializers.ResponseError("kids profile: camera apps stay off"))
		return
	}
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

// Browse — PARKED, not deleted (#213).
//
// This listed plugins from Hugging Face Spaces by the `autonomous-os-plugin`
// tag. That was the prototype; plugins belong in our own catalog, beside
// skills. It is commented out rather than removed because the shape is right
// and only the source is wrong: when the catalog grows a `plugins` collection,
// uncomment this, swap the fetch for skills.StoreGet("/api/v1/plugins", …)
// (system/skills/store.go already speaks to apiv2.autonomous.ai), and
// re-register the route in server.go.
//
// Installing is unaffected — POST /api/plugin/install takes a git URL and does
// not go through here.
//
// func (h *PluginHandler) Browse(c *gin.Context) {
// 	const hfURL = "https://huggingface.co/api/spaces?filter=autonomous-os-plugin&full=true&sort=likes&direction=-1"
// 	client := &http.Client{Timeout: 10 * time.Second}
// 	resp, err := client.Get(hfURL)
// 	if err != nil {
// 		c.JSON(http.StatusBadGateway, serializers.ResponseError("failed to reach HuggingFace"))
// 		return
// 	}
// 	defer resp.Body.Close()
// 	body, err := io.ReadAll(resp.Body)
// 	if err != nil {
// 		c.JSON(http.StatusBadGateway, serializers.ResponseError("failed to read HuggingFace response"))
// 		return
// 	}
// 	var spaces []any
// 	if err := json.Unmarshal(body, &spaces); err != nil {
// 		c.JSON(http.StatusBadGateway, serializers.ResponseError("invalid JSON from HuggingFace"))
// 		return
// 	}
// 	c.JSON(http.StatusOK, serializers.ResponseSuccess(spaces))
// }
