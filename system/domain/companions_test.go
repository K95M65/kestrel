package domain

import "testing"

func TestCompanionApps(t *testing.T) {
	apps := CompanionApps()
	if len(apps) == 0 {
		t.Fatal("expected at least Kestrel Buddy")
	}
	b := apps[0]
	if b.ID != "autonomous-buddy" || b.DownloadURL == "" || b.SourceURL == "" {
		t.Fatalf("buddy = %+v", b)
	}
	var platforms int
	for _, a := range apps {
		if a.Kind == "buddy" {
			platforms++
			if a.DirectURL == "" {
				t.Fatalf("buddy %s missing direct_url", a.ID)
			}
		}
	}
	if platforms < 3 {
		t.Fatalf("expected Mac + Windows + Linux buddy, got %d", platforms)
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
	if _, ok := LookupTrustedPlugin("dance"); !ok {
		t.Fatal("dance should be trusted")
	}
	if _, ok := LookupTrustedPlugin("autonomous-buddy"); ok {
		t.Fatal("buddy is not a robot-app")
	}
	if len(TrustedPlugins()) < 4 {
		t.Fatal("expected trusted robot apps")
	}
}
