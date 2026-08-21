package plugin

import "strings"

// CameraExclusive is a robot-app that owns the camera until Stop.
// Kids pack / kid role must not start these.
func CameraExclusive(name string) bool {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "cameraman", "photobooth":
		return true
	default:
		return false
	}
}
