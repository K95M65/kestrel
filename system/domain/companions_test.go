package domain

import "testing"

func TestCompanionApps(t *testing.T) {
	apps := CompanionApps()
	if len(apps) == 0 {
		t.Fatal("expected at least Autonomous Buddy")
	}
	b := apps[0]
	if b.ID != "autonomous-buddy" || b.DownloadURL == "" || b.SourceURL == "" {
		t.Fatalf("buddy = %+v", b)
	}
	var robots int
	for _, a := range apps {
		if a.Kind != "robot-app" {
			continue
		}
		robots++
		if a.InstallURL == "" || a.Subdir == "" {
			t.Fatalf("robot app %s missing install_url/subdir: %+v", a.ID, a)
		}
	}
	if robots < 4 {
		t.Fatalf("expected dance/emotions/cameraman/phrase-teacher, got %d", robots)
	}
}
