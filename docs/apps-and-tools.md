# Apps and tools (Kestrel)

How community functions — the kind on
[Pollen’s Reachy Mini apps](https://pollen-robotics.com/reachy-mini/apps/) —
should live in Autonomous OS / Kestrel so **Hermes and OpenClaw both see them**.

The brain is a plugin. The catalog is the OS.

---

## What Pollen actually ships

The store mixes three kinds of thing. Treating them as one “plugin” is why
installs feel stuck.

| Pollen kind | Examples | Runs where | Talks to the LLM? |
|---|---|---|---|
| **App** | dance party, arcade, telepresence, marionette, mime | On the robot (or in the browser via WebRTC) | Usually no — it *takes over* the body |
| **Tool-space** | search, weather, time, surf report | Remote HF Space over MCP | Yes — one function the model can call |
| **JS app** | joystick, radio, photobooth, stories | Browser, talks to the robot | Sometimes |

Skills (`SKILL.md`) are a fourth thing we already have: markdown the model
reads, then it curls HAL. They are not exclusive apps and they are not
callable tools.

---

## What we already have

```
Agent (OpenClaw / Hermes / …)
        │  MCP (per-runtime config)
        │  SKILL.md (copied into workspace)
        ▼
os-server     — sensing, [HW:/…] markers, MCP reconcile on runtime switch
        ▼
HAL           — motion, speak, camera
        ▲
plugins v1    — systemd Python, HAL HTTP only, no tools, no SKILL
```

`system/agent/mcp_reconcile.go` already clones MCP entries when you switch
OpenClaw ↔ Hermes. Plugins v1 never write those entries. That is the hole.

Plugin-system.md already named the fix as **v2**: auto-register plugin methods
as MCP tools. Do that, and the brain no longer matters.

---

## The Kestrel package (one folder, three optional parts)

```
my-app/
  plugin.json     # required — kind + capabilities
  tools.json      # optional — MCP tool schemas
  SKILL.md        # optional — when/how the agent should use it
  main.py         # optional — long-running process (dance, radio, mime)
```

`plugin.json`:

```json
{
  "name": "radio",
  "version": "1.0.0",
  "kind": "tool",
  "description": "Play internet radio through the robot speaker",
  "capabilities": ["audio"],
  "exclusive": false,
  "entry": "main.py"
}
```

`kind`:

| kind | Meaning | Install does |
|---|---|---|
| `tool` | Stateless functions | Start a tiny MCP stdio/HTTP server; `WriteMCPEntry` on the **active** runtime (OpenClaw *and* persisted so reconcile clones it) |
| `app` | Full behavior, may own the body | systemd unit like plugins v1; optional `exclusive` parks HAL voice |
| `js` | Browser UI | Dash iframe / open URL; talks to HAL over the existing `/api/hardware` proxy |

Any combination is valid. Radio is `tool` + `main.py`. Dance party is `app` +
`exclusive`. Weather is `tool` only (or a remote MCP URL, no local process).
Emotions already exist as HAL — do not reimplement as an app.

---

## Why the agent stays out of the install path

```
dash / POST /api/plugin/install
        │
        ▼
os-server
  1. fetch package (git / HF Space)
  2. gate on ROBOT.md capabilities
  3. if tools.json → register MCP via AgentGateway.WriteMCPEntry
  4. if SKILL.md  → copy into the active workspace + the other runtime’s
                    workspace so a later switch still has it
  5. if main.py   → systemd os-plugin-<name>
```

Hermes and OpenClaw only **consume** MCP + skills. They do not own the catalog.
That is the same rule as Grok login: credentials and tools live in the OS.

Remote Pollen tool-spaces (`reachy-mini-weather-tool`) are just an MCP URL in
`tools.json` — no download, same `WriteMCPEntry`.

---

## Mapping the Pollen store onto this

**Tools (voice should call these):**

- weather / surf / time / search / office greeter facts  
- magic 8-ball, encyclopedia, bedtime story *generation*  
→ `kind: tool` (+ short SKILL.md). Fast path can even special-case a few
(as `/v1/talk` already does for weather).

**Apps (take over the desk):**

- dance duo, music quiz, arcade, marionette, mime, telepresence, cameraman  
→ `kind: app`, `exclusive: true`. One extra tool `start_<name>` / `stop_<name>`
so “Hey Reachy, start the dance party” still works from either brain.

**JS / dash:**

- joystick, photobooth, radio UI, stories composer  
→ `kind: js`, listed on Overview next to Buddy downloads.

**Already in the OS — do not wrap:**

- emotions → `/emotion`  
- conversation itself → HAL + talk / Grok  
- dashboard → our web UI

---

## Trust and safety

Same as Pollen apps: **install = full trust** for local `app`/`tool` code.
Remote MCP is weaker (no robot FS). `exclusive` apps must yield on
`POST /api/plugin/:name/stop` so voice can come back. Capability tags keep
Lamp-only or no-camera apps off Reachy and vice versa.

---

## First examples (in-tree)

Under `integrations/apps/`, installable from Setup **Apps** (`subdir` clone):

| App | Plugin name | Behavior |
|---|---|---|
| Dance | `dance` | Emotion groove; `DANCE_MUSIC` for a track, empty = silent |
| Emotions reel | `emotions` | Spoken tour of built-in faces |
| Cameraman | `cameraman` | `/servo/track` on face until stopped |
| Phrase teacher | `asl-teacher` | hello / yes / no / thank you / happy with body (no fingers) |

Voice: matching `skills/dance`, `skills/cameraman`, `skills/asl-teacher`.

## Ship order

1. These four apps + plugin `subdir` install (**done** on Kestrel).
2. MCP register on install so Hermes and OpenClaw get `start_dance` tools without SKILL.md.
3. Exclusive-mode park of HAL voice while dance/cameraman runs.

Do not put OAuth, model pickers, or the dash chrome in this system. Those stay
core OS (see `setup-integrations.md`).
