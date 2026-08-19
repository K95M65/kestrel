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
	return []CompanionApp{
		{
			ID:          "autonomous-buddy",
			Name:        "Autonomous Buddy",
			Platform:    "macOS 13+",
			Version:     "0.0.18",
			Summary:     "Lets the robot open apps, type, click, and screenshot on your Mac.",
			Hint:        "Install, allow Accessibility (and Screen Recording for screenshots), then pair with the code from this dash.",
			DownloadURL: base + "/releases/latest",
			DirectURL:   base + "/releases/latest/download/AutonomousBuddy.zip",
			SourceURL:   base + "/tree/main/integrations/companions/autonomous-buddy",
		},
	}
}
