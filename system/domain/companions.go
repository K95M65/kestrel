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

// CompanionApps is the onboarding catalog (Buddy Mac today).
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
			Name:        "Autonomous Buddy",
			Platform:    "macOS 13+",
			Version:     "0.0.18",
			Summary:     "Lets the robot open apps, type, click, and screenshot on your Mac.",
			Hint:        "Install, allow Accessibility (and Screen Recording for screenshots), then pair with the code from this dash.",
			Kind:        "buddy",
			DownloadURL: base + "/releases/latest",
			DirectURL:   base + "/releases/latest/download/AutonomousBuddy.zip",
			SourceURL:   base + "/tree/main/integrations/companions/autonomous-buddy",
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
	}
}
