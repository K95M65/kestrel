package domain

// Plugin represents an installed plugin.
type Plugin struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
	Entry       string `json:"entry"`
	Status      string `json:"status"` // "stopped", "running", "failed", "installing"
	URL         string `json:"url"`
}

// PluginInstallRequest is the payload for POST /api/plugin/install.
// Home users send ID from the trusted catalog. URL is the advanced path.
type PluginInstallRequest struct {
	URL    string `json:"url"`
	Subdir string `json:"subdir,omitempty"`
	ID     string `json:"id,omitempty"`
}
