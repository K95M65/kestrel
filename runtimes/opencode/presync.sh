#!/usr/bin/env bash
# runtime-opencode-presync — run by switch-runtime right before opencode starts,
# once at the end of install.sh, and by EnsureOnboarding on every os-server boot
# / config change (hermes-style). It does NOT restart opencode.service — the Go
# side owns unit restarts (EnsureOnboarding hash-gates .env + opencode.json around
# this run). The bridge itself ships inside the os-server binary (`os-server
# opencode-gatewayd`) — nothing to materialize here. It OWNS everything stateful:
#
#   §1 MIGRATE — one-time persona/memory copy → workspace/, and skills copy →
#      ~/.config/opencode/skills (opencode's global discovery root), from the
#      openclaw workspace (marker /root/.opencode/.openclaw-migrated; a factory
#      reset that wipes /root/.opencode clears it so migrate re-runs next switch).
#   §2 CONFIG  — ~/.config/opencode/opencode.json provider/model wiring from
#      config.json llm_* (JSON via jq; the existing "mcp" object is preserved —
#      os-server runtimes/opencode/mcp.go owns those entries). The provider is an
#      @ai-sdk/openai custom provider (Responses API) whose apiKey is a
#      {env:LLM_API_KEY} reference resolved from .env at launch.
#   §3 ENV     — /root/.opencode/.env (systemd EnvironmentFile): gatewayd
#      token/port/workspace + LLM_API_KEY from llm_api_key.
#
# This file is EMBEDDED IN os-server (runtimes/opencode/presync.sh) and
# materialized to /usr/local/bin/runtime-opencode-presync on every switch.
set -euo pipefail

# Env-overridable (with production defaults) so tests can point at a temp dir.
CONFIG_JSON="${CONFIG_JSON:-/root/config/config.json}"   # device/project config (source of truth)
OPENCODE_DIR="${OPENCODE_DIR:-/root/.opencode}"
WS_DIR="$OPENCODE_DIR/workspace"
ENV_FILE="$OPENCODE_DIR/.env"
# opencode reads its global config + skills from XDG (~/.config/opencode).
OPENCODE_XDG_DIR="${OPENCODE_XDG_DIR:-/root/.config/opencode}"
OPENCODE_CONFIG="$OPENCODE_XDG_DIR/opencode.json"
SKILLS_DIR="$OPENCODE_XDG_DIR/skills"   # opencode global skill discovery root

# The campaign provider speaks the OpenAI RESPONSES API (not chat-completions —
# device-verified: {base}/chat/completions 404s, {base}/responses works). opencode
# reaches the Responses API via the @ai-sdk/openai package (per opencode docs:
# "for models using /v1/responses, use @ai-sdk/openai"), which appends /responses
# to baseURL.
DEFAULT_BASE_URL="https://campaign-api.autonomous.ai/api/v1/ai/v1"
DEFAULT_MODEL="Auto-AI"

log() { echo "[opencode-presync] $*"; }

command -v jq >/dev/null 2>&1 || { log "ERROR: jq not found — cannot sync opencode config" >&2; exit 1; }

mkdir -p "$WS_DIR" "$OPENCODE_DIR/attachments" "$OPENCODE_XDG_DIR"

# read a field from the device config.json ("" when absent/empty), newline-safe
# for the KEY=value .env format below.
dev() { jq -r ".${1} // empty" "$CONFIG_JSON" 2>/dev/null | tr -d '\n\r' || true; }

# ── §1 MIGRATE (restore persona/memory/skills from openclaw, once) ──────────────
# Gate on a sentinel marker under the opencode data dir, so a factory reset that
# wipes /root/.opencode clears it and migrate re-runs on the next switch. The
# marker is written ONLY after a clean copy, so a failed migrate is retried on
# the next run. Non-fatal: a device without openclaw just skips this.
MIGRATE_MARKER="$OPENCODE_DIR/.openclaw-migrated"
OC_WS="/root/.openclaw/workspace"
if [ ! -f "$MIGRATE_MARKER" ] && [ -d "$OC_WS" ]; then
  log "no migration marker — migrating persona/memory/skills from openclaw"
  # stop openclaw first so the copy doesn't race its live on-disk state (retry
  # 3x, proceed regardless — migrate is non-fatal).
  for attempt in 1 2 3; do
    systemctl stop openclaw 2>/dev/null || true
    if ! systemctl is-active --quiet openclaw; then
      log "openclaw stopped (attempt ${attempt}/3)"; break
    fi
    [ "$attempt" -eq 3 ] && log "WARN: openclaw still active after 3 attempts — continuing anyway"
    sleep 1
  done
  if (
    set -e
    # Persona files — copy openclaw's living copies verbatim (overwrite).
    # AGENTS.md included: opencode reads AGENTS.md natively, so openclaw's is
    # directly usable (the Go onboarding re-injects the OS block anyway).
    for f in IDENTITY.md SOUL.md KNOWLEDGE.md HEARTBEAT.md MEMORY.md USER.md AGENTS.md; do
      if [ -f "$OC_WS/$f" ]; then
        cp -f "$OC_WS/$f" "$WS_DIR/$f"
        log "copied $f from openclaw"
      fi
    done
    # memory/ → workspace/memory — only when absent, so a re-run never clobbers
    # local edits.
    if [ -d "$OC_WS/memory" ] && [ ! -d "$WS_DIR/memory" ]; then
      cp -a "$OC_WS/memory" "$WS_DIR/memory" && log "copied memory/ from openclaw"
    fi
    # skills/ → ~/.config/opencode/skills (opencode's global discovery root —
    # opencode auto-discovers every <name>/SKILL.md there). Only when the
    # destination is absent, so a re-run never clobbers local edits.
    if [ -d "$OC_WS/skills" ] && [ ! -d "$SKILLS_DIR" ]; then
      cp -a "$OC_WS/skills" "$SKILLS_DIR" && log "copied skills/ from openclaw → $SKILLS_DIR"
    fi
  ); then
    touch "$MIGRATE_MARKER"
    log "openclaw migration complete (marker written)"
  else
    log "WARN: openclaw migration failed — will retry on next presync run"
  fi
fi

# ── §2 CONFIG (~/.config/opencode/opencode.json) ────────────────────────────────
# Regenerate the provider/model head from config.json every run, preserving the
# existing "mcp" object (os-server runtimes/opencode/mcp.go owns those entries).
LLM_BASE_URL="$(dev llm_base_url)"; [ -n "$LLM_BASE_URL" ] || LLM_BASE_URL="$DEFAULT_BASE_URL"
LLM_BASE_URL="${LLM_BASE_URL%/}"   # strip trailing slash
LLM_MODEL="$(dev llm_model)"; [ -n "$LLM_MODEL" ] || LLM_MODEL="$DEFAULT_MODEL"

# Preserve an existing mcp block ({} when the file is absent / has none).
EXISTING_MCP="$(jq -c '.mcp // {}' "$OPENCODE_CONFIG" 2>/dev/null || echo '{}')"
[ -n "$EXISTING_MCP" ] || EXISTING_MCP='{}'

log "write $OPENCODE_CONFIG (model=campaign/$LLM_MODEL base=$LLM_BASE_URL)"
jq -n \
  --arg base "$LLM_BASE_URL" \
  --arg model "$LLM_MODEL" \
  --argjson mcp "$EXISTING_MCP" '
{
  "$schema": "https://opencode.ai/config.json",
  "model": ("campaign/" + $model),
  "provider": {
    "campaign": {
      "npm": "@ai-sdk/openai",
      "name": "Autonomous campaign-api",
      "options": { "baseURL": $base, "apiKey": "{env:LLM_API_KEY}" },
      "models": { ($model): { "name": $model } }
    }
  }
}
+ (if ($mcp | length) > 0 then { "mcp": $mcp } else {} end)
' >"$OPENCODE_CONFIG.tmp"
mv "$OPENCODE_CONFIG.tmp" "$OPENCODE_CONFIG"

# ── §3 ENV (config.json wins) ───────────────────────────────────────────────────
# systemd EnvironmentFile format: plain KEY=value lines, no quoting. LLM_API_KEY
# is referenced by opencode.json's provider options via {env:LLM_API_KEY}.
LLM_API_KEY="$(dev llm_api_key)"
log "write $ENV_FILE (key=$( [ -n "$LLM_API_KEY" ] && echo set || echo EMPTY ))"
umask 077
{
  echo "# Managed by runtime-opencode-presync — do not edit (synced from /root/config/config.json)."
  # OPENCODE_WS_TOKEN MUST match runtimes/opencode/constants.go Token; read by
  # `os-server opencode-gatewayd` (gatewayd package).
  echo "OPENCODE_WS_TOKEN=autonomous_opencode_token"
  echo "OPENCODE_PORT=18793"
  echo "OPENCODE_WORKSPACE=$WS_DIR"
  [ -n "$LLM_API_KEY" ] && echo "LLM_API_KEY=$LLM_API_KEY"
} >"$ENV_FILE.tmp"
mv "$ENV_FILE.tmp" "$ENV_FILE"
umask 022

# Channels: telegram/slack/discord are DEVICE-OWNED under opencode (os-server
# runs the receive loops itself — telegram_poll.go / slack.go / discord.go),
# driven by config.json tokens, so there is nothing runtime-side to write here.
log "channels: device-owned (telegram/slack/discord) — nothing to sync runtime-side"

# ── CLI LOGIN SHELL ENV ──────────────────────────────────────────────────────
# A bare `opencode` in an SSH/web-CLI login shell otherwise has no LLM_API_KEY
# (the .env is only injected into the systemd service). Drop a profile.d snippet
# that sources the ACTIVE runtime's .env into INTERACTIVE login shells — runtime
# resolved live from config.json so it stays correct across runtime switches.
# Guarded to interactive shells only (no leak into scripts/cron).
write_cli_login_env() {
  cat >/etc/profile.d/agent-cli-env.sh <<'PROFILE'
# Managed by os-server runtime presync — do not edit.
case "$-" in *i*) ;; *) return 2>/dev/null || exit 0 ;; esac
_rt="$(sed -n 's/.*"agent_runtime"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' /root/config/config.json 2>/dev/null | head -1)"
case "$_rt" in
  claudecode)
    if [ -f /root/.claudecode/.env ]; then set -a; . /root/.claudecode/.env; set +a; fi
    export IS_SANDBOX=1
    ;;
  codex)
    if [ -f /root/.codex/.env ]; then set -a; . /root/.codex/.env; set +a; fi
    export CODEX_HOME=/root/.codex
    ;;
  opencode)
    if [ -f /root/.opencode/.env ]; then set -a; . /root/.opencode/.env; set +a; fi
    ;;
esac
unset _rt
PROFILE
  chmod 0644 /etc/profile.d/agent-cli-env.sh
}
write_cli_login_env && log "wrote /etc/profile.d/agent-cli-env.sh (interactive CLI auto-login)"

log "done — opencode.json + env synced"
