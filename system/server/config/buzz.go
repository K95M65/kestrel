package config

// Buzz is the LAN hive (Block Buzz-shaped). Host=true means this body is
// the relay other units join. RelayURL is a websocket (this OS, or later a
// real Buzz/Nostr relay).
type Buzz struct {
	Enabled  bool   `json:"enabled,omitempty"`
	Host     bool   `json:"host,omitempty"`
	RelayURL string `json:"relay_url,omitempty"`
}
