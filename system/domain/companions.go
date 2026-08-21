package domain

import "os"

// CompanionApp is a downloadable client that pairs with the device.
type CompanionApp struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Platform    string `json:"platform"`
	Version     string `json:"version,omitempty"`
	Summary     string `json:"summary"`
	Hint        string `json:"hint,omitempty"`
	DownloadURL string `json:"download_url"`
	DirectURL   string `json:"direct_url,omitempty"`
	SourceURL   string `json:"source_url"`
	Kind        string `json:"kind,omitempty"`
	InstallURL  string `json:"install_url,omitempty"`
	Subdir      string `json:"subdir,omitempty"`
}

// githubRepo is the public repo that ships companion binaries/source.
func githubRepo() string {
	if v := os.Getenv("KESTREL_GITHUB_REPO"); v != "" {
		return v
	}
	return "K95M65/kestrel"
}

// LookupTrustedPlugin is a robot-app from CompanionApps (installable on the body).
func LookupTrustedPlugin(id string) (CompanionApp, bool) {
	for _, a := range CompanionApps() {
		if a.ID == id && a.Kind == "robot-app" && a.InstallURL != "" {
			return a, true
		}
	}
	return CompanionApp{}, false
}

// TrustedPlugins is the home-user install list (no raw git URL).
func TrustedPlugins() []CompanionApp {
	var out []CompanionApp
	for _, a := range CompanionApps() {
		if a.Kind == "robot-app" && a.InstallURL != "" {
			out = append(out, a)
		}
	}
	return out
}

// CompanionApps is the onboarding catalog (Buddy on Mac, Windows, and Linux).
func CompanionApps() []CompanionApp {
	repo := githubRepo()
	base := "https://github.com/" + repo
	git := base + ".git"
	robot := func(id, name, summary, hint, folder string) CompanionApp {
		return CompanionApp{
			ID:          id,
			Name:        name,
			Platform:    "Reachy Mini",
			Version:     "1.0.0",
			Summary:     summary,
			Hint:        hint,
			Kind:        "robot-app",
			DownloadURL: base + "/tree/main/integrations/apps/" + folder,
			SourceURL:   base + "/tree/main/integrations/apps/" + folder,
			InstallURL:  git,
			Subdir:      "integrations/apps/" + folder,
		}
	}
	return []CompanionApp{
		{
			ID:          "autonomous-buddy",
			Name:        "Kestrel Buddy",
			Platform:    "macOS 13+",
			Version:     "0.1.0",
			Summary:     "Lets the robot open apps, type, click, and screenshot on your Mac.",
			Hint:        "Install, allow Accessibility (and Screen Recording for screenshots), then pair with the code from this dash.",
			Kind:        "buddy",
			DownloadURL: base + "/releases/latest",
			DirectURL:   base + "/releases/latest/download/AutonomousBuddy.zip",
			SourceURL:   base + "/tree/main/integrations/companions/autonomous-buddy",
		},
		{
			ID:          "autonomous-buddy-windows",
			Name:        "Kestrel Buddy",
			Platform:    "Windows 10+",
			Version:     "0.1.0",
			Summary:     "Same pairing as Mac. Opens sites and apps on this PC.",
			Hint:        "Run the desktop binary, pair with the code from this dash. Click and screenshot still need the Mac app.",
			Kind:        "buddy",
			DownloadURL: base + "/releases/latest",
			DirectURL:   base + "/releases/latest/download/autonomous-buddy.exe",
			SourceURL:   base + "/tree/main/integrations/companions/autonomous-buddy/desktop",
		},
		{
			ID:          "autonomous-buddy-linux",
			Name:        "Kestrel Buddy",
			Platform:    "Linux",
			Version:     "0.1.0",
			Summary:     "Same pairing as Mac. Opens sites and apps on this computer.",
			Hint:        "Run the desktop binary. Typing needs xdotool (X11) or ydotool (Wayland).",
			Kind:        "buddy",
			DownloadURL: base + "/releases/latest",
			DirectURL:   base + "/releases/latest/download/autonomous-buddy-linux",
			SourceURL:   base + "/tree/main/integrations/companions/autonomous-buddy/desktop",
		},
		robot("dance", "Dance",
			"Groove with happy/excited moves. Optional music via DANCE_MUSIC.",
			"Install, then Start. Set DANCE_MUSIC to a song search for music, or leave empty for a silent dance.",
			"dance"),
		robot("emotions", "Emotions reel",
			"Walk through hello, happy, shy, sad, and more, out loud.",
			"A demo of built-in expressions. Everyday feelings still use the emotion skill.",
			"emotions"),
		robot("cameraman", "Cameraman",
			"Keep a face in frame until you stop the app.",
			"Needs the camera. Stop from the Plugins list when you're done.",
			"cameraman"),
		robot("asl-teacher", "Phrase teacher",
			"Teach hello, yes, no, thank you, and happy with Reachy's body (no hands).",
			"Honest about having no fingers. Five phrases only.",
			"asl-teacher"),
		robot("radio", "Radio",
			"Leave a station on the speaker until you Stop.",
			"Uses the speaker until Stop. Say stop in Talk or tap Stop here. Kids may use radio.",
			"radio"),
		robot("photobooth", "Photobooth",
			"Say cheese, one JPEG, done.",
			"Uses the camera for one shot. Kids profile keeps this off.",
			"photobooth"),
	}
}
