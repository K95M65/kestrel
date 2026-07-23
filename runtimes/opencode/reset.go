package opencode

import (
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"go.autonomous.ai/os/system/lib/osreset"
)

// Paths + unit set up by install.sh (see install.sh).
const (
	opencodeDataDir           = "/root/.opencode"             // bridge state dir (HOME=/root)
	opencodeXDGConfigDir      = "/root/.config/opencode"      // CLI global config (opencode.json, AGENTS.md, skills/)
	opencodeXDGDataDir        = "/root/.local/share/opencode" // CLI data (auth.json, sessions)
	opencodeUnit              = "opencode"                    // systemd unit name
	opencodeStopVerifyTimeout = 5 * time.Second
)

// ResetAgent is the OpenCode factory-reset wipe, called on the active gateway by
// server/system/factoryreset.go. OpenCode keeps nothing worth preserving:
// opencode.json/.env are regenerated from the project config.json by presync.sh
// on the next switch, and wiping the state + XDG dirs also drops CLI auth,
// sessions and the migrate marker — a true factory state.
func (s *OpenCodeService) ResetAgent() error {
	wipeOpenCodeState()
	return nil
}

// wipeOpenCodeState: stop+disable gateway → wipe /root/.opencode → recreate baseline dirs.
func wipeOpenCodeState() {
	// 1. Stop the gateway (Restart=always is overridden by an explicit stop) and
	//    confirm it is down — it holds the data dir open, so it must die before wipe.
	log.Printf("[factory-reset/opencode] step 1/4 — systemctl stop opencode")
	if out, err := exec.Command("systemctl", "stop", opencodeUnit).CombinedOutput(); err != nil {
		log.Printf("[factory-reset/opencode] step 1/4 — stop error: %v — %s", err, strings.TrimSpace(string(out)))
	}
	if waitForOpenCodeStop(opencodeUnit, opencodeStopVerifyTimeout) {
		log.Printf("[factory-reset/opencode] step 1/4 — confirmed inactive")
	} else {
		log.Printf("[factory-reset/opencode] step 1/4 — WARNING still active after %s", opencodeStopVerifyTimeout)
	}

	// 2. Disable — reboot defaults to openclaw; switch-runtime re-enables on switch back.
	log.Printf("[factory-reset/opencode] step 2/4 — systemctl disable opencode")
	if out, err := exec.Command("systemctl", "disable", opencodeUnit).CombinedOutput(); err != nil {
		log.Printf("[factory-reset/opencode] step 2/4 — disable error: %v — %s", err, strings.TrimSpace(string(out)))
	}

	// 3. Wipe the bridge state dir (/root/.opencode: .env, session.json,
	//    workspace/, attachments/, install.log, .openclaw-migrated) AND opencode's
	//    XDG dirs (opencode.json + AGENTS.md + skills/ under ~/.config/opencode;
	//    auth.json + sessions under ~/.local/share/opencode).
	for _, d := range []string{opencodeDataDir, opencodeXDGConfigDir, opencodeXDGDataDir} {
		log.Printf("[factory-reset/opencode] step 3/4 — wiping %s", d)
		osreset.WipePath("[factory-reset/opencode]", d)
	}

	// 4. Recreate the baseline dirs (OpenCode has no onboard subcommand — presync
	//    re-asserts opencode.json/.env on the next switch; the CLI recreates its
	//    own state under XDG on first run). Non-fatal.
	log.Printf("[factory-reset/opencode] step 4/4 — recreate baseline dirs")
	for _, d := range []string{opencodeDataDir + "/workspace", opencodeDataDir + "/attachments"} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			log.Printf("[factory-reset/opencode] step 4/4 — mkdir %s error: %v (non-fatal)", d, err)
		}
	}
}

// waitForOpenCodeStop polls is-active until the unit is inactive or timeout elapses.
func waitForOpenCodeStop(unit string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for {
		if exec.Command("systemctl", "is-active", "--quiet", unit).Run() != nil {
			return true // non-zero exit → not active
		}
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(200 * time.Millisecond)
	}
}
