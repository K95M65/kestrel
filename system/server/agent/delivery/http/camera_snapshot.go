package http

import (
	"regexp"
	"strings"
)

// cameraSnapshotNameRE accepts only the basename HAL creates for an explicit
// GET /camera/snapshot?save=true request. The UI receives a server URL, never
// the agent runtime's filesystem path.
var cameraSnapshotNameRE = regexp.MustCompile(`\bsnap_[0-9]+\.jpg\b`)

// cameraSnapshotURL returns the UI-safe URL for a snapshot produced by a
// camera tool call. Tool output is untrusted agent text, so both the camera
// command and HAL's fixed filename shape must match before exposing anything.
func cameraSnapshotURL(toolArgs, result string) string {
	if !strings.Contains(toolArgs, "/camera/snapshot") {
		return ""
	}
	name := cameraSnapshotNameRE.FindString(result)
	if name == "" {
		return ""
	}
	return "/api/sensing/agent-snapshot/" + name
}
