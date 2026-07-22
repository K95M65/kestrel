# Pollen Ecosystem Reference

Technical reference on how the Pollen Robotics / Hugging Face ecosystem works
for Reachy Mini. Covers voice architecture, tool registry, app distribution, and
takeaways for improving Autonomous OS.

Written July 2026.

## Voice Architecture

Pollen's conversation app runs a single-pipeline voice loop:

```
Mic → Silero VAD → Parakeet STT → LLM (w/ tools via MCP) → Qwen3 TTS → Speaker
      └──── all in one process, one WebSocket ───────────────────────────┘
```

Key properties:
- **No pipeline split.** The LLM has full tool access (via MCP) in every turn,
  so there is no need for a separate "smart" path vs "fast" path.
- **Server-side VAD.** Silero runs in the same process as STT, zero network hop.
- **Swappable LLM backends:** llama.cpp (local 4B), vLLM, OpenAI, HF Inference.
- **Cascaded STT+TTS** (not native speech-to-speech). Voice quality is good but
  not on par with native Gemini/OpenAI Realtime.

### Takeaway

A single pipeline with full tool access per turn is architecturally simpler than
splitting fast-but-limited vs slow-but-smart paths. Worth prototyping on Reachy
(greenfield, no legacy voice path) to evaluate latency and quality trade-offs
before deciding on backport.

## Tool Registry

The conversation app has a three-tier tool system:

### Tier 1: Built-In Tools (Local)

Ship with the app in `src/.../tools/`. Each is a Python class inheriting from
`Tool`, registered into a global `ALL_TOOLS` dict. 17 built-in tools:
`move_head`, `dance`, `stop_dance`, `play_emotion`, `camera`,
`idle_do_nothing`, `head_tracking`, `remember`, `forget`, `go_to_sleep`, etc.

### Tier 2: Custom Local Tools (User-Authored)

Python files in `external_tools/`. Auto-loaded when `AUTOLOAD_EXTERNAL_TOOLS=1`.
Each must expose a unique `Tool.name`. App fails fast on name collision.

### Tier 3: Remote MCP Tools (HF Spaces)

Managed by `tool_spaces.py` and `mcp_client.py`. Tools run remotely in public
Gradio Spaces that expose MCP at `/gradio_api/mcp/`. No code downloads to the
robot — the LLM calls tools over HTTPS.

```bash
# Install a remote tool
reachy-mini-conversation-app tool-spaces add pollen-robotics/reachy-mini-search-tool

# List / remove
reachy-mini-conversation-app tool-spaces list
reachy-mini-conversation-app tool-spaces remove <owner/space>
```

**Namespacing** uses double-underscore to prevent collision:
```
Space:  pollen-robotics/reachy-mini-search-tool
Alias:  pollen_robotics_reachy_mini_search_tool
Tool:   pollen_robotics_reachy_mini_search_tool__search_web
```

**Preinstalled remote tools:**
- `pollen-robotics/reachy-mini-search-tool` (web search)
- `pollen-robotics/reachy-mini-time-tool` (time/timezone)
- `pollen-robotics/reachy-mini-weather-tool` (weather)

**Per-profile gating:** each profile has a `tools.txt` whitelist — a tool is
only available if its ID appears in the active profile's list.

### Takeaway

Adding an MCP client to the agent gateway would let Autonomous OS consume HF
Space tools alongside local `SKILL.md` skills. Publishing a few of our skills as
Gradio MCP endpoints would give them visibility in the HF ecosystem. Both can
coexist — local skills for offline/deep integration, remote MCP tools for
community contributions.

## App Distribution

Two channels for getting capabilities onto a Reachy Mini:

### Channel A: Apps (Full Robot Behavior)

- Python packages inheriting from `ReachyMiniApp` with a `run()` method
- Tagged `reachy_mini_python_app` on HF Spaces
- One-click install from robot dashboard or REST API:
  ```bash
  curl -X POST http://localhost:8000/api/apps/install \
    -H "Content-Type: application/json" \
    -d '{"url": "https://huggingface.co/spaces/<user>/<app>"}'
  ```
- Downloaded and run locally as subprocess
- Full access to robot hardware (motion, audio, vision)
- 200+ apps from 150+ creators as of mid-2026

### Channel B: Tool-Spaces (Stateless Remote Functions)

- Gradio apps tagged `reachy-mini-tool` + `mcp` on HF
- Run remotely on HF Space, never downloaded
- Extend the conversation app's LLM with new capabilities
- CLI install: `tool-spaces add <owner/space>`

| | Apps | Tool-Spaces |
|--|-----|-------------|
| Code runs | On robot (local) | On HF Space (remote) |
| Install | Dashboard / REST API | CLI |
| Scope | Full behavior | Single function |
| Trust model | Full (local exec) | Sandboxed (remote) |

### JS Apps (Browser-Based)

A third variant: JS apps tagged `reachy_mini_js_app` run entirely in the
browser. They connect to the robot via WebRTC signaling — zero install, open URL
and go.

Example: [reachy-dance-duo](https://huggingface.co/spaces/TwinPeaksTownie/reachy-dance-duo)
by Carson Maestas (community). Two robots dance synced to YouTube music. Uses
real-time beat detection, WebRTC audio routing, and Three.js IK visualization.
85 commits, 30 likes, 3 contributors — built entirely with JS + HF Spaces.

### Takeaway

The low barrier to entry (HF Space = distribution, WebRTC = connectivity, no
SSH/firmware required) enables community contributions. Worth considering how
Autonomous OS can lower its own barrier for external contributors — whether
through HF Spaces, a web-based skill editor, or a simpler local plugin format.

## Profile System

Each Reachy Mini can have multiple "profiles" — a profile is a personality
configuration:

```
profiles/<name>/
  instructions.txt    # system prompt for the LLM
  tools.txt           # whitelist of enabled tool IDs
```

Switching profiles changes the robot's personality and available tools without
restarting. Similar in concept to our `SOUL.md` persona system.

## Agent Development Support

Pollen provides AI coding agent guidance via:

- **AGENTS.md** — top-level behavioral protocol for Claude Code, Cursor, Copilot
- **agents.local.md** — per-session user context (robot type, preferences)
- **skills/*.md** — 12 domain-specific reference files (motion philosophy, safe
  torque, control loops, app creation, debugging, etc.)
- **CLAUDE.md** — Claude Code-specific instructions

Similar to our `CLAUDE.md` but with richer domain reference material.

## Key Repos

| Repository | Purpose |
|------------|---------|
| [reachy_mini_conversation_app](https://github.com/pollen-robotics/reachy_mini_conversation_app) | Voice app, tool registry, profiles |
| [reachy_mini](https://github.com/pollen-robotics/reachy_mini) | SDK, docs, AGENTS.md, community skills |
| [reachy-mini-os](https://github.com/pollen-robotics/reachy-mini-os) | OS image build (pi-gen based) |

## References

- [Adding MCP Tools to Reachy Mini (HF Blog)](https://huggingface.co/blog/adding-mcp-tools-to-reachy-mini)
- [Robot App Store Launch (VentureBeat)](https://venturebeat.com/technology/the-app-store-for-robots-has-arrived-hugging-face-launches-open-source-reachy-mini-app-store-with-200-apps)
- [Make and Publish Apps (HF Blog)](https://huggingface.co/blog/pollen-robotics/make-and-publish-your-reachy-mini-apps)
- [Reachy Mini Hardware Datasheet](https://huggingface.co/docs/reachy_mini/platforms/reachy_mini/hardware)
- [Jeff Geerling Review](https://www.jeffgeerling.com/blog/2026/testing-reachy-mini-hugging-face-robot/)