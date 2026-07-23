package http

import (
	"regexp"
	"strings"
)

// cameraSnapshotPathRE accepts JPEGs only from the active agent runtime's
// approved camera-output directories. The UI receives a server URL, never the
// runtime's filesystem path.
var cameraSnapshotPathRE = regexp.MustCompile(`/root/\.(openclaw|hermes|picoclaw|codex|claudecode)/(workspace|media/hal-snapshots)/([A-Za-z0-9][A-Za-z0-9._-]*\.(jpg|jpeg))\b`)

// cameraSnapshotURL returns the UI-safe URL for a snapshot produced by a
// camera tool call. Tool output is untrusted agent text, so both the camera
// command and an approved runtime path must match before exposing anything.
func cameraSnapshotURL(toolArgs, result string) string {
	if !strings.Contains(toolArgs, "/camera/snapshot") {
		return ""
	}
	matches := cameraSnapshotPathRE.FindStringSubmatch(result)
	if len(matches) != 5 {
		return ""
	}
	source := matches[2]
	if source == "media/hal-snapshots" {
		source = "media-hal-snapshots"
	}
	return "/api/sensing/agent-snapshot/" + matches[1] + "/" + source + "/" + matches[3]
}
