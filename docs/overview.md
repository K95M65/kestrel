# Architecture Overview — Autonomous

## 3-Layer Architecture

```
Agentic Runtime (AI/LLM) → OS Server (Go, :5000) → HAL (Python, :5001) → Hardware
```

| Layer | Language | Port | Role |
|-------|----------|------|------|
| Agentic Runtime | Go | WS | AI brain, LLM, SKILL.md, memory, channels |
| OS Server | Go | 5000 | System (network, OTA, MQTT, reset), sensing event routing, local intent |
| HAL | Python | 5001 | Hardware drivers (servo, LED, camera, audio, display), FastAPI |

## Project Directory

```
system/
├── cmd/os-server/main.go              — OS Server entry point
├── cmd/bootstrap/main.go         — OTA bootstrap worker
├── server/
│   ├── server.go                 — Gin HTTP server, route setup
│   ├── config/                   — JSON config management
│   ├── health/delivery/http/     — Health, system info, dashboard
│   ├── network/delivery/http/    — WiFi scan, connect
│   ├── device/delivery/          — Setup (HTTP + MQTT handlers)
│   ├── sensing/delivery/http/    — Sensing event → intent match / agent gateway
│   └── openclaw/delivery/sse/    — Agent gateway status, SSE events
├── agent/  ambient/  beclient/  buddy/  device/  healthwatch/
├── intent/  monitor/  network/  skills/  statusled/  vision/
│                                 — System managers, one folder per diagram chip
├── lib/mqtt/                     — MQTT client (Eclipse Paho autopaho)
├── domain/                       — Shared structs
├── bootstrap/                    — OTA worker
└── web/                          — React 19 + Vite + Tailwind CSS 4 SPA

runtimes/                   — Swappable brains: openclaw/ hermes/ picoclaw/ codex/ claudecode/ opencode/

hal/
├── server.py                     — FastAPI server
├── config.py                     — Runtime constants (sensing thresholds, timeouts, URLs)
├── board/                        — Device profiles, board pin maps, and overlays
├── drivers/                      — Hardware services (camera, motors, RGB, sensing, voice, display)
├── routes/                       — FastAPI capability route modules
├── safety/                       — Parsed safety policy and deterministic gates
├── realtime/                     — Realtime voice agent and context managers
├── server_support/               — Shared HTTP/security support
└── pyproject.toml                — Python dependencies (opencv-python, insightface)

devices/                          — Per-device configs and overlays
  contract/                       — Shared API contracts (+ cts/ compliance suite)
skills/                           — Built-in SKILL.md files for agent runtime, including
                                    skill-creator for owner-authored skills
integrations/                     — Off-device: companions/, chat-bridges/, perception-service/
```

## Principles

- **Hardware is a plugin** — plug in and it works, unplug and it's skipped
- **System layer runs WITHOUT the runtime** — device always responds
- **Code is the source of truth** — docs reflect code
- **HAL is the hardware driver** — no AI logic
- **SKILL.md native** — no MCP, LLM reads skills and calls curl directly
- **Owners can create skills** — the built-in `skill-creator` guides an owner
  through drafting, testing, and packaging a skill for the Autonomous Skill Store.

## Voice Pipeline

```
Mic (always on) → Local VAD (RMS energy, free)
    → Speech detected → Connect Deepgram STT
        → "hey lamp, turn off light" → voice_command → local intent → execute
        → "hey wanna grab lunch?" → voice (ambient) → OpenClaw
    → Silence 3s → Disconnect Deepgram
```

## Sensing Flow

```
HAL sensing loop (every 2s) → Read 1 camera frame, run all detectors:
    ├─ Motion detection (frame diff) → event if >8% pixels changed
    ├─ Face recognition (InsightFace buffalo_sc) → friend/stranger classification
    │     → presence.enter (annotated JPEG with colored bboxes: green=friend, red=stranger)
    │     → presence.leave (3 consecutive ticks without face)
    ├─ Light level (mean brightness, every 30s) → event if change >30/255
    └─ Sound detection (mic RMS) → event if > threshold

Event has image? (large motion, face enter) → encode frame full-resolution JPEG q85
Face enter image: original frame annotated with bounding boxes + labels

POST /api/sensing/event {type, message, image?}
    → OS server (Go):
        1. Voice event + local intent match? → execute directly (~50ms)
        2. No match → forward to OpenClaw:
           - Has image → SendChatMessageWithImage (text + vision content block)
           - No image → SendChatMessage (text only)
        3. OpenClaw AI sees image + reads context → decides action → calls SKILL API
```

Cooldowns to protect LLM costs: motion/sound 60s, presence 10s, light.level 30s.
