# Companions

Companions pair a user's computer with an Autonomous device. The pairing runs in
both directions:

- **Device as the physical face of the computer** — activity on the computer
  (Claude Desktop, Claude Code) is mirrored onto the device's LED, round
  display, and voice, and the device can relay decisions (e.g. voice-approving
  a tool call) back to the computer.
- **Agent driving the computer** — the device's voice agent controls the
  desktop (open apps, navigate, type), TeamViewer-style but driven by AI. This
  is the `autonomous-buddy` direction.

| App | Direction | Platform / language | Transport |
|-----|-----------|---------------------|-----------|
| [`autonomous-buddy/`](autonomous-buddy/) | agent → computer control | macOS 13+ menu-bar app (Swift) | WebSocket to os-server `/api/buddy/ws` (planned; mock today) |
| [`claude-desktop-buddy/`](claude-desktop-buddy/) | device ← computer feedback + voice approvals | Go daemon on the device; peers on the user's Mac | BLE Nordic UART (Claude Desktop) + HTTP `:5002` (Claude Code plugin) |

## autonomous-buddy — remote computer use

**Status: Phase 1A scaffold.** A macOS menu-bar shell (Swift Package in
`autonomous-buddy/macos/`) that builds and runs but does no networking yet.
Discovery (mDNS), pairing, the WebSocket connection, and command execution are
planned as Phases 1B–1E; a Go `mock-device/` server lets you exercise the
pairing UI and command REPL end-to-end today.

On the device side the capability is declared as `companion.control` (see
`contract/capabilities.md` — capability group `companion`, backed by the buddy
service in os-server, target `computer`). Design and MVP plan:
[`autonomous-buddy/docs/autonomous-buddy.md`](autonomous-buddy/docs/autonomous-buddy.md)
and [`autonomous-buddy/docs/autonomous-buddy-mvp.md`](autonomous-buddy/docs/autonomous-buddy-mvp.md).

## claude-desktop-buddy — mirror Claude activity onto the device

A Go daemon (`claude-desktop-buddy`, systemd service on the device) that
bridges Claude on the user's Mac to the device:

- **BLE link to Claude Desktop** — a Nordic UART GATT server receives
  heartbeats, chat events, and permission prompts from Claude Desktop's
  "Hardware Buddy" feature; a state machine turns them into device behavior
  (`idle` / `busy` / `attention`, plus `heart` / `celebrate` flourishes) on the
  LED, display, and TTS.
- **HTTP API on `:5002`** with a split trust model (see `httpapi/server.go`):
  - `GET /health` — open (used by LAN discovery).
  - LAN pushes (`POST /claude-code/notify`, `/claude-code/usage`,
    `/claude-code/approval-request`) — gated by `Authorization: Bearer
    <device admin password>`; fails closed (401) when no password is configured.
  - Agent endpoints (`GET /status`, `POST /claude-desktop/approve|deny`,
    `GET /claude-code/pending`) — loopback-only, called by the on-device agent.
- **Voice approvals** — when Claude Desktop asks permission for a tool call,
  the on-device agent (via the [`skills/claude-buddy`](../../skills/claude-buddy/SKILL.md)
  skill) asks the user out loud and POSTs the answer to
  `/claude-desktop/approve` or `/claude-desktop/deny`; the daemon relays the
  decision back over BLE.

### claude-code-buddy — the Claude Code plugin

[`claude-desktop-buddy/claude-code-buddy/`](claude-desktop-buddy/claude-code-buddy/)
is the Claude Code / HTTP counterpart to the BLE path. It runs on the user's
Mac as a Claude Code plugin (Python 3 stdlib only) and POSTs events to the
daemon's `:5002` API:

- **Task Done** (`Stop` hook) — Claude finished a response.
- **Usage** — pushes 5-hour / 7-day usage when it crosses a threshold, or on
  demand via `/claude-code-buddy:usage`.
- **Notifications** (`Notification` hook + `/claude-code-buddy:notify`) —
  "Claude needs you" pings and one-off custom messages.
- **Reverse approval** (`PermissionRequest` hook) — the hook long-polls
  `POST /claude-code/approval-request` on `:5002`; the on-device agent asks the
  user by voice and resolves it via loopback `POST /claude-code/approve|deny`.

Install from the marketplace:

```bash
claude plugins marketplace add https://raw.githubusercontent.com/autonomous-ai/autonomous-os/main/integrations/companions/claude-desktop-buddy/claude-code-buddy/.claude-plugin/marketplace.json
claude plugins install claude-code-buddy
```

## Pairing with the device

Buddy pairing is owned by os-server (`services/internal/buddy`, routes in
`services/server/buddy/delivery/http/`):

1. `POST /api/buddy/pair/start` (admin-gated) — `IssuePairingCode` generates a
   single-use 6-digit code with a 60s TTL; the device shows it to the user.
2. The buddy app submits the code with its name/fingerprint via
   `POST /api/buddy/pair/confirm` (anonymous, code-based) — `ConfirmPairing`
   consumes the code and persists a pairing record, returning a long-lived
   bearer token + buddy ID.
3. The buddy then holds a persistent WebSocket at `GET /api/buddy/ws`
   (token-gated); the agent dispatches commands to it through localhost-only
   `POST /api/buddy/command` and `POST /api/buddy/exec/:action`. Admin can
   inspect via `GET /api/buddy/status` and revoke via `DELETE /api/buddy`
   (or the buddy itself via `DELETE /api/buddy/self`).

The claude-desktop-buddy BLE path pairs separately, via standard Bluetooth
passkey pairing (see [`claude-desktop-buddy/docs/setup.md`](claude-desktop-buddy/docs/setup.md)).
