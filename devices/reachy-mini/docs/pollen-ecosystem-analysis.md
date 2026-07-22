# Pollen Ecosystem Analysis

Competitive analysis of Pollen Robotics / Hugging Face ecosystem vs Autonomous
OS, based on review of their conversation app, community apps, and tool
registry. Written July 2026.

## Sources Reviewed

1. [reachy_mini_conversation_app](https://huggingface.co/spaces/pollen-robotics/reachy_mini_conversation_app) — Pollen's official voice conversation app
2. [reachy-dance-duo](https://huggingface.co/spaces/TwinPeaksTownie/reachy-dance-duo) — community app (2 robots dance sync to YouTube)
3. Internal feedback from D on voice architecture and tool registry

## Boss D's Two Concerns

D identified two areas where Pollen is genuinely ahead:

### Concern 1: Voice Pipeline Complexity

**Autonomous OS — two pipelines, one decision point per turn:**

```
                    ┌─── Fast lane (hal/realtime) ────────────────┐
                    │ speech-in → Gemini Live / OpenAI Realtime   │
                    │ → speech-out (native, fast, but limited     │
                    │   tools/memory/skills)                      │
Mic → local VAD ──→│                                              │
                    │ EVERY TURN: "can realtime handle this,      │
                    │  or delegate_to_main?"                      │
                    │                                              │
                    ├─── Main path ───────────────────────────────┤
                    │ Deepgram STT → agent runtime → TTS          │
                    │ (full skills, memory, persona, perception)  │
                    └─────────────────────────────────────────────┘
```

**Pollen — one pipeline, zero decision points:**

```
Mic → Silero VAD → Parakeet STT → LLM (w/ tools via MCP) → Qwen3 TTS → Speaker
      └──── all in one process, one WebSocket ───────────────────────────┘
```

**Why theirs is simpler:**

| Aspect | Autonomous OS | Pollen |
|--------|--------------|--------|
| Pipelines | 2 (realtime + main) | 1 |
| Decision per turn | handle-vs-delegate | None |
| Coherence burden | Must keep 2 pipelines in sync | N/A |
| Latency workarounds | Many (break turn on delegate, express_emotion in-process, echo skip) | Few |
| Voice quality | Native s2s (Gemini) = excellent | Cascaded STT+TTS = good but not native |
| Tool access | Realtime: limited; Main: full | Full (LLM has MCP tools) |

**Nuance:** Gemini Realtime itself is very fast (native speech-to-speech). The
problem is not Gemini's speed — it's that Gemini Realtime lacks full
tools/memory/skills, so we need a second pipeline for "smart" turns. If/when
Gemini/OpenAI Realtime adds full tool use + memory, our two-pipeline
architecture becomes unnecessary and the fast lane becomes the only lane. But
we're not there yet.

**Pollen's trade-off:** their cascaded pipeline has worse voice quality than
native Gemini s2s (accent artifacts, lost prosody), but it's architecturally
simpler and the LLM has full tool access in every turn.

### Concern 2: Skill Distribution — Where Should Skills Live?

**The threat:** if HF Spaces + MCP becomes the standard way to distribute robot
capabilities, then our `SKILL.md` file-based system becomes invisible to the
community.

| Distribution channel | Pros | Cons |
|---------------------|------|------|
| **Local `SKILL.md`** (current) | Offline, fast, full control, deep integration (persona, memory, safety bounds) | No community distribution, no one outside the team sees them |
| **HF Spaces + MCP** (Pollen) | 200+ apps, 150+ creators, community network effect, zero-install | Dependent on HF infra, tools must be stateless, no deep integration |
| **agentskills.io** | Cross-platform registry, not locked to one vendor | Unclear adoption, adds a dependency |

**Current score:**

- Pollen: 200+ community apps, 150+ creators, 10k+ units, HF Hub as
  marketplace
- Autonomous OS: internal skills only, no public distribution channel

**D's point:** we write better skills (deeper persona, memory, safety
integration) but no one outside the team can see or use them. They write simpler
skills but have a marketplace with 10k+ users.

## Pollen Conversation App — Technical Details

### Architecture

- **Voice loop:** Silero VAD → Parakeet STT → LLM → Qwen3 TTS, all in one
  Python process
- **LLM backends:** llama.cpp (local 4B), vLLM, OpenAI, HF Inference
- **17 built-in tools:** `move_head`, `dance`, `stop_dance`, `play_emotion`,
  `camera`, `idle_do_nothing`, `head_tracking`, `remember`, `forget`,
  `go_to_sleep`, etc.
- **Profile system:** `profiles/<name>/instructions.txt` (system prompt) +
  `tools.txt` (tool whitelist)

### Three-Tier Tool Registry

| Tier | Location | Example |
|------|----------|---------|
| Built-in | `src/.../tools/` (ships with app) | `move_head`, `dance`, `camera` |
| Custom local | `external_tools/*.py` (user writes) | Any `Tool` subclass |
| Remote MCP | HF Spaces (`/gradio_api/mcp/`) | `search_web`, `get_weather` |

**Tool-spaces CLI:**

```bash
# Install a remote MCP tool from a HF Space
reachy-mini-conversation-app tool-spaces add pollen-robotics/reachy-mini-search-tool

# List installed
reachy-mini-conversation-app tool-spaces list

# Remove
reachy-mini-conversation-app tool-spaces remove pollen-robotics/reachy-mini-search-tool
```

**Namespacing:** double-underscore to prevent collisions:
```
Space: pollen-robotics/reachy-mini-search-tool
Tool:  pollen_robotics_reachy_mini_search_tool__search_web
```

**Collision rule:** `Tool.name` must be globally unique across all three tiers.
App fails fast on collision.

### Two Distribution Channels

| | Apps (full behavior) | Tool-Spaces (stateless functions) |
|--|---------------------|----------------------------------|
| HF tag | `reachy_mini_python_app` | `reachy-mini-tool` + `mcp` |
| Runs where | Downloaded to robot, subprocess | Remote on HF Space |
| Install | Dashboard one-click or REST API | CLI `tool-spaces add` |
| Scope | Full robot control (motion, audio, vision) | Single function (search, weather) |
| Code trust | Full (local execution) | Sandboxed (remote) |

### Preinstalled Remote Tools

- `pollen-robotics/reachy-mini-search-tool` (web search)
- `pollen-robotics/reachy-mini-time-tool` (time/timezone)
- `pollen-robotics/reachy-mini-weather-tool` (weather)

## Dance Duo App — Community Ecosystem Evidence

[TwinPeaksTownie/reachy-dance-duo](https://huggingface.co/spaces/TwinPeaksTownie/reachy-dance-duo)
by Carson Maestas. 85 commits, 30 likes, 3 contributors. Not official Pollen.

**What it does:** 2 Reachy Minis dance synchronized to YouTube music.

**Technical highlights:**
- Real-time beat detection from mic (robot 1) + YouTube audio analysis (robot 2)
- WebRTC audio routing hack: monkey-patch `getUserMedia()` to stream YouTube
  audio through robot speaker
- Forward/inverse kinematics computed in browser (Three.js)
- Zero install — open URL, connect via WebRTC signaling, done

**Why it matters for us:** evidence that Pollen's platform model works. One
community dev shipped a complex app (IK, beat detection, 3D visualization)
using only JS + HF Spaces. No firmware flashing, no SSH, no build toolchain.
Our equivalent barrier: know Go/Python, understand agent gateway, SSH into Pi,
understand SKILL.md schema.

## Possible Responses

### Voice Architecture

| Option | Description | Effort |
|--------|-------------|--------|
| A. Keep two pipelines | Wait for Gemini/OpenAI Realtime to add full tool use, then drop main path | Zero (wait) |
| B. Add s2s backend to HAL | Use cascaded VAD→STT→LLM→TTS as a third voice mode, single pipeline with full tools | Medium |
| C. Prototype on Reachy first | Reachy has no legacy voice path yet — try single-pipeline there, evaluate, decide on backport | Low-Medium |

**Recommendation:** Option C. Reachy is greenfield, no legacy to break.

### Skill Distribution

| Option | Description | Effort |
|--------|-------------|--------|
| A. MCP client in agent gateway | Consume HF Space tools alongside local SKILL.md | Medium |
| B. Publish skills as HF Spaces | Wrap our skills as Gradio MCP endpoints for visibility | Low |
| C. Dual registry | Local scan + HF Hub query | Medium-High |
| D. Do nothing | Differentiate on depth (persona, memory, sensing), ignore distribution | Zero |

**Recommendation:** A + B. Add MCP client to consume community tools. Wrap a
few of our skills as HF Spaces for ecosystem presence. Keep SKILL.md for
local/offline (our differentiated value).

## Key Repos

- [reachy_mini_conversation_app](https://github.com/pollen-robotics/reachy_mini_conversation_app) — voice app + tool registry
- [reachy_mini](https://github.com/pollen-robotics/reachy_mini) — SDK, docs, AGENTS.md
- [reachy-mini-os](https://github.com/pollen-robotics/reachy-mini-os) — OS image (pi-gen based)

## References

- [Adding MCP Tools to Reachy Mini (HF Blog)](https://huggingface.co/blog/adding-mcp-tools-to-reachy-mini)
- [Robot App Store Launch (VentureBeat)](https://venturebeat.com/technology/the-app-store-for-robots-has-arrived-hugging-face-launches-open-source-reachy-mini-app-store-with-200-apps)
- [Make and Publish Apps (HF Blog)](https://huggingface.co/blog/pollen-robotics/make-and-publish-your-reachy-mini-apps)
- [BLE Reset Guide (Seeed Studio)](https://wiki.seeedstudio.com/reachymini_platforms_reachy_mini_reset/)
- [Jeff Geerling Review](https://www.jeffgeerling.com/blog/2026/testing-reachy-mini-hugging-face-robot/)