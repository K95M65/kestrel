package claudecode

import (
	_ "embed"

	"go.autonomous.ai/os/system/domain"
	"go.autonomous.ai/os/system/lib/runtimereg"
)

// InstallScript is the device-side installer for the Claude Code backend, embedded in
// os-server so it ships + OTA-updates with the binary (no CDN round-trip needed).
// os-server materializes it to disk and switch-runtime runs it the first time a
// device switches to claudecode. See install.sh for the contract (installs the
// claude CLI — no bun/channel plugins, telegram + discord are device-owned
// loops — writes + starts claudecode.service, drops the verify hook, runs
// runtime-claudecode-presync) and docs/agentic/claudecode.md.
//
//go:embed install.sh
var InstallScript []byte

// PresyncScript is the device-side pre-start hook for Claude Code. os-server
// materializes it to /usr/local/bin/runtime-claudecode-presync on every switch,
// switch-runtime runs it right before claudecode starts, and EnsureOnboarding
// re-runs it on every os-server boot / config change (hermes-style). It OWNS the
// launch env (/root/.claudecode/.env — ANTHROPIC_* from config.json llm_*,
// read by the claudecode-gatewayd bridge inside os-server) and the headless
// flags (~/.claude.json, workspace settings.json) — no channel config anymore
// (telegram + discord are device-owned loops; presync just removes stale
// ~/.claude/channels state) — so the config self-heals on every switch/boot,
// including after a factory reset. Persona/memory migration is NOT here: claudecode has a Go adapter
// (system/agent/migrate_persona/runtime_claudecode.go). Materializing this
// from os-server (not from install.sh, which only re-runs on a first install /
// failed verify) is what makes a plain os-server OTA refresh it on disk.
//
//go:embed presync.sh
var PresyncScript []byte

// ReadyScript verifies the authenticated Claude Code bridge WebSocket upgrade.
//
//go:embed ready.sh
var ReadyScript []byte

// Register the embedded installer + presync so system/device can materialize
// them without importing this package (which would cycle via statusled → device).
func init() {
	runtimereg.Register(domain.AgentRuntimeClaudeCode, InstallScript)
	runtimereg.RegisterPresync(domain.AgentRuntimeClaudeCode, PresyncScript)
	runtimereg.RegisterReadiness(domain.AgentRuntimeClaudeCode, ReadyScript)
}
