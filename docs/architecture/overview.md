# Architecture

Autonomous is a layered stack. Each layer exposes an interface to the layer above and
depends only on the one below, so any layer can be replaced without touching the others.

![Autonomous architecture](autonomous-stack.svg)

## Layers

**Skills** — what the device does: 24 skills, each a `SKILL.md` the runtime invokes — apps
like `guard`, `mood`, `scene`, `habit`, `wellbeing`, plus capability wrappers
(`led-control`, `servo-control`, `camera`, `music`, …). A skill is an *ability*; the device's
*character* is its `SOUL.md`. First-party skills use the same public contract a third party
gets. *(`skills/`)*

**Tools** — how the runtime reaches beyond the device: **MCP** servers and the **CLI**. Skills
are the device's own abilities (through the HAL); tools are external capabilities the runtime
calls.

**System Managers** — the always-on Go daemon: `intent` (fast local commands), `network`,
`sensing` routing, `monitor` (flow event bus), `healthwatch`, `ambient`, and `device`.
Deterministic — they run with or without the runtime. OTA runs as its own worker
(`bootstrap/`). *(`system/internal`)*

**Agentic Runtime** — **OpenClaw**, **Hermes**, **PicoClaw**, **OpenAI Codex**, **Claude Code**,
or a custom runtime. Runs the skills, embodies the device's `SOUL.md`, and decides what to act
on. Swappable — and where Autonomous's differentiated value (the default brain, memory,
character) lives. *(`system/internal/agent/runtimes/{openclaw,hermes,picoclaw,codex,claudecode}`)*

**HAL — Capabilities** — the frozen, versioned interface, 12 capabilities: `audio`, `vision`,
`sensing`, `presence`, `motion`, `light`, `display`, `expression`, `media`, `connectivity`,
`companion`, `system`. Skills call capabilities (`motion.move`), never hardware models, so one
skill runs on any body that declares the capability — Lamp's servo arm and the Unitree Go2-W's
wheels both serve `motion`. A device's `DEVICE.md` declares which it has; the runtime mounts
only those. The HAL also hosts the **safety gate** (`hal/safety`): `SAFETY.md` bounds —
e-stop, motion limits, brightness, quiet hours — enforced deterministically below the brain,
never by the LLM. The realtime voice agent (`hal/drivers/realtime`) runs in-process here
too — runtime-layer code hosted in the HAL, marked purple in the diagram.
*(`contract/` + `hal` — see [hal.md](hal.md))*

**Linux Kernel** — the vendor kernel (Raspberry Pi OS / OrangePi, or a robot's onboard compute)
we run on; we don't ship one. Our **Drivers** (`motors`, `rgb`, `display`, `camera`, `voice`
(STT/TTS/VAD), `gpio`/`touch`, `bluetooth` — `hal/drivers`, with per-board wiring in
`hal/board`) are userspace programs talking to it through GPIO/SPI/ALSA/V4L2; **Power Management** is the foundation. *(see [kernel.md](kernel.md))*

## See also

[hal.md](hal.md) · [kernel.md](kernel.md) ·
[`DEVICE-SPEC.md`](../../contract/DEVICE-SPEC.md) ·
[`capabilities.md`](../../contract/capabilities.md)
