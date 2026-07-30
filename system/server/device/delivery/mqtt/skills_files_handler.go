package mqtthandler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"unicode/utf8"

	"go.autonomous.ai/os/system/domain"
	"go.autonomous.ai/os/system/skills"
)

// mqttMaxFileText caps the inlined text of a single file on this uplink.
//
// The HTTP twin inlines up to 512 KB per file, which is fine over a LAN socket
// but not over MQTT: brokers commonly cap a message around 256 KB, and the whole
// payload also has to survive base64/JSON escaping. 64 KB leaves room for the
// envelope while still covering any realistic SKILL.md or reference doc; anything
// longer comes back flagged `truncated`.
const mqttMaxFileText = 64 << 10

// handleSkillsFiles handles kind="skills.files" — the MQTT twin of
// GET /api/agent/skills/files. That endpoint is LAN-only and admin-gated, so the
// backend (and through it a mobile app) has no way to inspect a skill the
// `skills` uplink advertised. This is that way in.
//
// Two modes, because MQTT is not a bulk transport and a full skill can be
// megabytes:
//
//	{"name":"music"}                         → the file LIST, no contents
//	{"name":"music","path":"music/SKILL.md"}  → that ONE file, contents inlined
//
// Synchronous: reading a skill dir is local disk, measured in milliseconds.
func (h *DeviceMQTTHandler) handleSkillsFiles(env domain.MQTTDataCommand) error {
	var req domain.MQTTSkillsFilesData
	if err := json.Unmarshal(env.Data, &req); err != nil {
		return h.publishDataResult(env.Kind, "failure", "invalid skills.files data: "+err.Error(), nil)
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Path = strings.TrimSpace(req.Path)
	if req.Name == "" {
		return h.publishDataResult(env.Kind, "failure", "name is required", nil)
	}

	runtimeName := h.agentGateway.Name()

	files, err := h.agentGateway.ReadSkillFiles(req.Name)
	if err != nil {
		step := "read"
		switch {
		case errors.Is(err, domain.ErrNotSupportedByRuntime):
			step = "unsupported_runtime"
		case errors.Is(err, skills.ErrInvalidSkillName):
			step = "validate_name"
		}
		slog.Error("skills.files: failed", "component", "mqtt",
			"skill", req.Name, "runtime", runtimeName, "step", step, "error", err)
		return h.publishDataResult(env.Kind, "failure", step+": "+err.Error(), map[string]interface{}{
			"name":        req.Name,
			"runtime":     runtimeName,
			"failed_step": step,
		})
	}

	if req.Path == "" {
		// List mode: strip every body so the payload stays bounded no matter how
		// big the skill is.
		return h.publishDataResult(env.Kind, "success", "", map[string]interface{}{
			"name":    req.Name,
			"runtime": runtimeName,
			"files":   stripSkillFileText(files),
		})
	}

	file, ok := findSkillFile(files, req.Path)
	if !ok {
		slog.Warn("skills.files: path not in skill", "component", "mqtt",
			"skill", req.Name, "path", req.Path)
		return h.publishDataResult(env.Kind, "failure", "not_found: "+req.Path, map[string]interface{}{
			"name":        req.Name,
			"path":        req.Path,
			"runtime":     runtimeName,
			"failed_step": "not_found",
		})
	}

	slog.Info("skills.files: success", "component", "mqtt",
		"skill", req.Name, "runtime", runtimeName, "path", file.Path, "bytes", len(file.Text))
	return h.publishDataResult(env.Kind, "success", "", map[string]interface{}{
		"name":    req.Name,
		"runtime": runtimeName,
		"file":    capSkillFileText(file),
	})
}

// stripSkillFileText drops every inlined body, leaving path/size/binary metadata.
// List mode must be bounded: a skill's combined text can far exceed what a broker
// will carry, and a caller asking for the list hasn't asked for contents.
func stripSkillFileText(files []domain.SkillBundleFile) []domain.SkillBundleFile {
	out := make([]domain.SkillBundleFile, 0, len(files))
	for _, f := range files {
		f.Text = ""
		f.Truncated = false
		out = append(out, f)
	}
	return out
}

// findSkillFile looks a file up by the exact path the list reported.
func findSkillFile(files []domain.SkillBundleFile, path string) (domain.SkillBundleFile, bool) {
	for _, f := range files {
		if f.Path == path {
			return f, true
		}
	}
	return domain.SkillBundleFile{}, false
}

// capSkillFileText re-truncates a file's text to the MQTT budget. The reader
// already caps at the (much larger) HTTP limit, so this only bites on a file
// between the two thresholds. Size keeps reporting the file's REAL length.
func capSkillFileText(f domain.SkillBundleFile) domain.SkillBundleFile {
	if len(f.Text) <= mqttMaxFileText {
		return f
	}
	text := f.Text[:mqttMaxFileText]
	// Don't cut a multi-byte rune in half — the payload has to stay valid UTF-8
	// or json.Marshal replaces the tail with U+FFFD.
	for len(text) > 0 && !utf8.ValidString(text) {
		text = text[:len(text)-1]
	}
	f.Text = text
	f.Truncated = true
	return f
}
