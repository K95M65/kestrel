package mqtthandler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"path/filepath"
	"strings"

	"go.autonomous.ai/os/system/domain"
	"go.autonomous.ai/os/system/skills"
)

// handleSkillsUpload handles kind="skills.upload" — the MQTT counterpart of
// uploading a bare .md file to POST /api/agent/skills/upload. Content is a full
// SKILL.md rather than the three authored fields accepted by skills.save.
func (h *DeviceMQTTHandler) handleSkillsUpload(env domain.MQTTDataCommand) error {
	content, errMsg := parseSkillsUploadData(env.Data)
	if errMsg != "" {
		return h.publishDataResult(env.Kind, "failure", errMsg, nil)
	}

	if !skillsInstallMu.TryLock() {
		return h.publishDataResult(env.Kind, "failure",
			"a skills install is in progress; try again later", nil)
	}
	defer skillsInstallMu.Unlock()

	runtimeName := h.agentGateway.Name()
	dir, err := h.agentGateway.InstallSkillMarkdown([]byte(content))
	if err != nil {
		step := classifySkillsUploadError(err)
		slog.Error("skills.upload: failed", "component", "mqtt",
			"runtime", runtimeName, "step", step, "error", err)
		return h.publishDataResult(env.Kind, "failure", step+": "+err.Error(), map[string]interface{}{
			"runtime":     runtimeName,
			"failed_step": step,
		})
	}

	name := filepath.Base(dir)
	slog.Info("skills.upload: success", "component", "mqtt",
		"skill", name, "runtime", runtimeName, "path", dir)
	return h.publishDataResult(env.Kind, "success", "", map[string]interface{}{
		"name":    name,
		"runtime": runtimeName,
		"path":    dir,
	})
}

func parseSkillsUploadData(raw json.RawMessage) (string, string) {
	var req domain.MQTTSkillsUploadData
	if err := json.Unmarshal(raw, &req); err != nil {
		return "", "invalid skills.upload data: " + err.Error()
	}
	if strings.TrimSpace(req.Content) == "" {
		return "", "content is required"
	}
	if int64(len(req.Content)) > skills.StoreMaxBytes {
		return "", "content exceeds maximum skill size"
	}
	return req.Content, ""
}

func classifySkillsUploadError(err error) string {
	switch {
	case errors.Is(err, domain.ErrNotSupportedByRuntime):
		return "unsupported_runtime"
	case errors.Is(err, skills.ErrInvalidFrontMatter):
		return "validate_front_matter"
	case errors.Is(err, skills.ErrInvalidSkillName):
		return "validate_name"
	default:
		return "install"
	}
}
