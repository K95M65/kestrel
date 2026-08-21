package plugin

import "testing"

func TestCameraExclusive(t *testing.T) {
	if !CameraExclusive("cameraman") || !CameraExclusive("Photobooth") {
		t.Fatal("expected camera apps")
	}
	if CameraExclusive("radio") || CameraExclusive("dance") || CameraExclusive("") {
		t.Fatal("speaker/motion apps are not camera-exclusive")
	}
}
