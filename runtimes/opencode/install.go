package opencode

import (
	_ "embed"

	"go.autonomous.ai/os/system/domain"
	"go.autonomous.ai/os/system/lib/runtimereg"
)

// InstallScript is the device-side installer for the OpenCode backend, embedded in
// os-server so it ships + OTA-updates with the binary. os-server materializes
// it to disk and switch-runtime runs it the first time a device switches to
// opencode. See install.sh for the contract (downloads the pinned opencode CLI
// release, writes + starts opencode.service running `os-server opencode-gatewayd`,
// drops the verify hook, runs runtime-opencode-presync) and docs/agentic/opencode.md.
//
//go:embed install.sh
var InstallScript []byte

// PresyncScript is the device-side pre-start hook for OpenCode. os-server
// materializes it to /usr/local/bin/runtime-opencode-presync on every switch, and
// switch-runtime runs it right before opencode starts (EnsureOnboarding also runs
// it every boot). It OWNS opencode.json (provider/model wiring, preserving the
// "mcp" object mcp.go edits), the .env file (llm_* from config.json), and the
// one-shot persona/memory migration from OpenClaw. The WS bridge itself is
// compiled into os-server (gatewayd package) — nothing to materialize. See
// presync.sh.
//
//go:embed presync.sh
var PresyncScript []byte

// Register the embedded installer + presync so system/device can materialize
// them without importing this package (which would cycle via statusled → device).
func init() {
	runtimereg.Register(domain.AgentRuntimeOpenCode, InstallScript)
	runtimereg.RegisterPresync(domain.AgentRuntimeOpenCode, PresyncScript)
}
