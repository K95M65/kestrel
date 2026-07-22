# Plugin System (Future)

Design notes for a community-contributed plugin system, inspired by Pollen
Robotics' `reachy_mini_python_app` model. Plugins run **outside HAL** as
independent processes that call HAL's HTTP API (`:5001`) for hardware access.

Written July 2026. Status: **design only, not implemented.**

## Problem

Currently, adding a new behavior to Autonomous OS requires:

1. PR into HAL (Python driver/route changes)
2. PR into skills/ (SKILL.md for agent awareness)
3. Code review + merge + OTA push

This creates a high barrier for community contributors. Pollen solved this
with a 1-click app install from HF Spaces — 200+ apps from 150+ creators,
most with no robotics background.

## Architecture

HAL stays as-is. Plugins are standalone processes that use HAL as a service:

```
┌─────────────────────────────────────────────┐
│  Agent Runtime (OpenClaw / Hermes)           │
│  ┌──────────┐  ┌──────────┐  ┌───────────┐  │
│  │ Skills   │  │ MCP Tools│  │ Plugins   │  │
│  │ (local)  │  │ (remote) │  │ (local)   │  │
│  └──────────┘  └──────────┘  └─────┬─────┘  │
└────────────────────────────────────┼────────┘
                                     │ subprocess
┌────────────────────────────────────▼────────┐
│  Plugin Process                              │
│  - Own Python venv                           │
│  - Calls HAL API (localhost:5001)            │
│  - Registers skills with agent               │
└─────────────────────────┬───────────────────┘
                          │ HTTP
┌─────────────────────────▼───────────────────┐
│  HAL (:5001)                                 │
│  LED, servo, audio, camera, sensing          │
└─────────────────────────────────────────────┘
```

## Key Design Points

### 1. Plugin Format

A plugin is a Python package with a standard entry point:

```python
# plugin.json (metadata)
{
  "name": "dance-party",
  "version": "1.0.0",
  "description": "Syncs robot dance to music beats",
  "entry": "main.py",
  "skills": ["dance_to_music", "stop_dance"],
  "hal_endpoints": ["/servo/*", "/led/*", "/audio/*"]
}
```

### 2. HAL as SDK

Plugins access hardware through HAL's existing HTTP API — no internal imports:

```python
import requests

# Move servo
requests.post("http://localhost:5001/servo/move", json={"pan": 45, "tilt": 10})

# Set LED
requests.post("http://localhost:5001/led/set", json={"effect": "pulse", "color": "blue"})

# Play audio
requests.post("http://localhost:5001/audio/speak", json={"text": "Let's dance!"})
```

No new HAL code needed — the API already exists.

### 3. Skill Auto-Registration

When a plugin starts, it registers its skills with the agent runtime so the
agent knows when to call it:

```
Plugin starts → POST /api/device/plugins/:name/skills
  → agent runtime sees new tools available
  → user says "play some dance music"
  → agent calls plugin's skill endpoint
```

### 4. Lifecycle (OS Server manages)

```
POST   /api/device/plugins/install   — download from URL (HF Space / Git)
GET    /api/device/plugins           — list installed plugins
POST   /api/device/plugins/:name/start
POST   /api/device/plugins/:name/stop
DELETE /api/device/plugins/:name     — uninstall
```

### 5. Isolation

- Each plugin runs in its own subprocess + Python venv
- Crash in plugin does not affect HAL or agent
- OS Server monitors plugin health, restarts on failure
- Plugins only access HAL via HTTP — no filesystem access to HAL internals

### 6. Distribution

Same model as Pollen:
- Publish as HF Space (tagged for discovery)
- Install by URL from web UI (Settings > Plugins)
- Or from CLI: `POST /api/device/plugins/install {"url": "..."}`

## Comparison with Existing Extension Points

| Extension | Runs | Scope | Barrier |
|-----------|------|-------|---------|
| SKILL.md | Agent runtime | Prompt-based | Low (text file) |
| MCP Tools | Cloud (HF Space) | Stateless function | Low (URL only) |
| **Plugins** | Device (subprocess) | Full behavior | Medium (Python package) |
| HAL code | Device (in-process) | Hardware driver | High (PR + review) |

## Open Questions

- Should plugins have direct WebSocket access to the agent runtime, or only
  through the skill registration API?
- How to handle plugins that need persistent state (database, config files)?
- Should there be a plugin marketplace/registry beyond HF Spaces?
- Resource limits (CPU, memory) per plugin on constrained devices (CM4)?

## References

- Pollen's app model: `devices/reachy-mini/docs/pollen-ecosystem-analysis.md`
  §App Distribution
- [Make and Publish Reachy Mini Apps (HF Blog)](https://huggingface.co/blog/pollen-robotics/make-and-publish-your-reachy-mini-apps)
- [Robot App Store (VentureBeat)](https://venturebeat.com/technology/the-app-store-for-robots-has-arrived-hugging-face-launches-open-source-reachy-mini-app-store-with-200-apps)
