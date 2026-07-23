package http

import "testing"

func TestCameraSnapshotURL(t *testing.T) {
	tests := []struct {
		name     string
		toolArgs string
		result   string
		want     string
	}{
		{
			name:     "saved camera snapshot",
			toolArgs: `{"command":"curl -s 'http://127.0.0.1:5001/camera/snapshot?save=true'"}`,
			result:   `{"path":"/root/.openclaw/media/hal-snapshots/snap_1710000000000.jpg"}`,
			want:     "/api/sensing/agent-snapshot/openclaw/media-hal-snapshots/snap_1710000000000.jpg",
		},
		{
			name:     "agent workspace JPEG",
			toolArgs: `curl -s http://127.0.0.1:5001/camera/snapshot?save=true`,
			result:   `{"path":"/root/.openclaw/workspace/cam_face3.jpg"}`,
			want:     "/api/sensing/agent-snapshot/openclaw/workspace/cam_face3.jpg",
		},
		{
			name:     "non camera result is not exposed",
			toolArgs: `{"command":"curl -s http://127.0.0.1:5001/servo/play"}`,
			result:   `{"path":"/root/.openclaw/media/hal-snapshots/snap_1710000000000.jpg"}`,
			want:     "",
		},
		{
			name:     "untrusted filename is not exposed",
			toolArgs: `curl -s http://127.0.0.1:5001/camera/snapshot?save=true`,
			result:   `{"path":"/etc/passwd"}`,
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := cameraSnapshotURL(tt.toolArgs, tt.result); got != tt.want {
				t.Fatalf("cameraSnapshotURL() = %q, want %q", got, tt.want)
			}
		})
	}
}
