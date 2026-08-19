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
}
