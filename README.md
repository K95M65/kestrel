<img src="docs/media/hero.gif" alt="Reachy Mini and Autonomous Lamp on one OS — antennas rise, the ring lights, both turn to look" width="720">

## Autonomous OS: The "Android" for Robots

Robots have been around for years but have never been autonomous — someone has to drive them with a remote, and they've stopped at scripted demos. Autonomous OS brings autonomy to robots: install it on your robot and it comes alive.

- **Your robot thinks.** Everything it sees and hears goes to an agentic reasoning engine running on the robot itself — [Hermes](runtimes/hermes/), [Claude Code](runtimes/claudecode/), or [whichever you choose](runtimes/) — that decides what to do next.
- **Your robot acts.** It [guards the house](skills/guard/), [knows your face](skills/face-enroll/), [follows you as you move](skills/servo-tracking/), [reads the mood on your face](skills/user-emotion-detection/), [sets the light](skills/scene/) — and does the desk work too: [Gmail, GitHub](skills/connectors/), [your Mac](skills/computer-use/). Each one is a [skill](skills/) — install more from the Skill Store, or write your own.
- **Your robot grows.** It has a built-in learning loop. It creates skills from experience, sharpens them as it uses them, keeps what it learns, searches its own past conversations, and builds a deeper picture of you with every session.

Autonomous OS is a fully customizable operating system for robots. Every component is swappable — [engine](runtimes/), [model](docs/hosted.md), [voice](hal/drivers/voice/), [skills](skills/), [board](hal/board/boards.json). Your robot declares what it has in a `DEVICE.md`, and the OS mounts exactly that. When a better one ships, your robot gets it the same day — and gets better without new hardware.

## Meet the first robots running Autonomous OS

We've installed Autonomous OS on three robots so far — Lamp, Intern and Pollen Robotics' Reachy Mini — and they think, act and grow on their own. Unitree's Go2-W is next. Here are the skills they have, and you can [install it on your own robot](#quick-start) too.

| PHYSICAL SKILLS | <a href="devices/lamp"><img src="devices/lamp/images/lamp-white.webp" width="150" alt="Lamp"><br>Lamp</a> | <a href="devices/intern-v2"><img src="devices/intern-v2/images/intern-tile.webp" width="150" alt="Intern"><br>Intern</a> | <a href="devices/reachy-mini"><img src="devices/reachy-mini/images/reachy-mini.webp" width="150" alt="Reachy Mini"><br>Reachy Mini</a> | <a href="devices/unitree-go2w"><img src="devices/unitree-go2w/images/go2-w-tile.webp" width="150" alt="Go2-W"><br>Go2-W</a> |
|---|:---:|:---:|:---:|:---:|
| [camera](skills/camera/)<br>See the room | ✅ |  | ✅ | ○ |
| [servo-tracking](skills/servo-tracking/)<br>Track an object | ✅ |  | ○ | ○ |
| [face-enroll](skills/face-enroll/)<br>Know your face | ✅ |  | ✅ |  |
| [speaker-recognizer](skills/speaker-recognizer/)<br>Know who is speaking | ✅ | ✅ | ✅ | ○ |
| [voice](skills/voice/)<br>Talk back | ✅ | ✅ | ✅ | ○ |
| [audio](skills/audio/)<br>Sound and volume | ✅ | ✅ | ✅ | ○ |
| [servo-control](skills/servo-control/)<br>Move and gesture | ✅ |  | ✅ | ○ |
| [emotion](skills/emotion/)<br>Show emotion | ✅ |  | ✅ |  |
| [led-control](skills/led-control/)<br>Colors and effects | ✅ | ✅ |  |  |
| [scene](skills/scene/)<br>Six lighting scenes | ✅ |  |  |  |
| [sensing](skills/sensing/)<br>Sense the room | ✅ | ✅ | ✅ | ○ |
| [sensing-track](skills/sensing-track/)<br>Remember what it sensed | ✅ | ✅ | ✅ | ○ |
| [guard](skills/guard/)<br>Guard the house | ✅ |  | ✅ |  |
| [music](skills/music/)<br>Play music | ✅ | ✅ | ✅ |  |
| [music-suggestion](skills/music-suggestion/)<br>Suggest a song | ✅ | ✅ | ✅ |  |
| [emotion-detection](skills/user-emotion-detection/)<br>Read your mood | ✅ | ✅ | ✅ | ○ |
| [mood](skills/mood/)<br>Track how you feel | ✅ | ✅ | ✅ | ○ |
| [wellbeing](skills/wellbeing/)<br>Posture and breaks | ✅ | ✅ | ✅ | ○ |
| [habit](skills/habit/)<br>Learn your routines | ✅ | ✅ | ✅ | ○ |

| DIGITAL SKILLS | <a href="devices/lamp"><img src="devices/lamp/images/lamp-white.webp" width="150" alt="Lamp"><br>Lamp</a> | <a href="devices/intern-v2"><img src="devices/intern-v2/images/intern-tile.webp" width="150" alt="Intern"><br>Intern</a> | <a href="devices/reachy-mini"><img src="devices/reachy-mini/images/reachy-mini.webp" width="150" alt="Reachy Mini"><br>Reachy Mini</a> | <a href="devices/unitree-go2w"><img src="devices/unitree-go2w/images/go2-w-tile.webp" width="150" alt="Go2-W"><br>Go2-W</a> |
|---|:---:|:---:|:---:|:---:|
| [Gmail](skills/connectors/)<br>Manage your email | ✅ | ✅ | ✅ | ○ |
| [Calendar](skills/connectors/)<br>Book and move meetings | ✅ | ✅ | ✅ | ○ |
| [Notion](skills/connectors/)<br>Search and write your notes | ✅ | ✅ | ✅ | ○ |
| [GitHub](skills/connectors/)<br>Issues and pull requests | ✅ | ✅ | ✅ | ○ |
| [Your Mac](skills/computer-use/)<br>Drive apps and the browser | ✅ | ✅ | ✅ |  |
| [Claude Buddy](skills/claude-buddy/)<br>Approve prompts by voice | ✅ | ✅ | ✅ | ○ |
| [Skill Creator](skills/skill-creator/)<br>Write its own skills | ✅ | ✅ | ✅ | ○ |

✅ runs today · ○ on the way

## Robot specifications

Personalizing Autonomous OS for your robot is simple. It's just four markdown files:

- **[DEVICE.md](devices/lamp/DEVICE.md)** — the body: what hardware this robot has
- **[SOUL.md](devices/lamp/SOUL.md)** — the self: who it is and how it talks
- **[SAFETY.md](devices/lamp/SAFETY.md)** — the bounds: how fast, how bright, how late
- **[SKILL.md](skills/guard/SKILL.md)** — the hands: one thing it can do

Check out the specifications of the first robots: [Lamp](devices/lamp/), [Intern](devices/intern-v2/), [Reachy Mini](devices/reachy-mini/), [Go2-W](devices/unitree-go2w/).

## Quick start

The fastest way in is a [Lamp](https://www.autonomous.ai/lamp) — our 5-DOF desk robot, $499, Autonomous OS already on it.

1. **Unbox it and open the app** — [iOS](https://apps.apple.com/app/id6744885683) · [Android](https://play.google.com/store/apps/details?id=ai.autonomous.connect.wifi).
2. **Tap Add robot → Lamp.** It asks for your Wi-Fi and handles keys and pairing.
3. **Say something.** It turns to look at you, the ring lights up, and it answers.
4. **Give it a job.** Tap a skill in the Skill Store, or type what you want it to do.
5. **Make it yours.** Edit [`SOUL.md`](devices/lamp/SOUL.md) and it is someone else on the next turn.

Full walkthrough, including how to swap the engine: [`devices/lamp/README.md`](devices/lamp/README.md).

### Other robots

| | |
|---|---|
| **[Autonomous Intern](devices/intern-v2/)** | Same image as Lamp, fewer capabilities declared |
| **[Reachy Mini](devices/reachy-mini/README.md)** | One command, installs beside Pollen's stack, nothing flashed |
| **[Build a Lamp](devices/lamp/BUILD.md)** | Raspberry Pi 5 or OrangePi 4 Pro, parts, wiring, CAD |
| **[Your own robot](docs/porting-a-robot.md)** | Three markdown files and a driver |
| **Simulation** | Not yet — the mock body is [#200](https://github.com/autonomous-ai/autonomous-os/issues/200) |

What we host and what stays on your network: [`docs/hosted.md`](docs/hosted.md).

## Skills are how it grows

Teaching it a new job is one file. Drop this on a Lamp or a Reachy Mini and both wave good morning:

```markdown
---
name: morning-wave
description: When someone says good morning, greet them by name and wave.
---
1. Reply with `[HW:/emotion:{"emotion":"greeting","intensity":0.9}]` — the arm plays its greeting move, the ring warms up.
2. Say good morning, using their name if you know their face. One sentence.
```

```
you    good morning
lamp   [HW:/emotion:{"emotion":"greeting","intensity":0.9}]   ← stripped here, POSTed to HAL
       head lifts, arm sweeps, ring warms
lamp   "Morning, Dee."                                        ← spoken while the move runs
```

Text becomes motion. The brain writes the `[HW:/…]` marker in its reply; the OS strips it out and sends it to the body; the body moves; the words are spoken. Same file on any body that declares the capability — a body that doesn't just ignores it. ([One turn, top to bottom](#every-layer-is-a-folder).)

**Two levels.** On your own robot: one folder in `/root/.openclaw/workspace/skills/<name>/`, live on the next conversation — no PR, no reboot, no Go. To ship it to *every* robot: the same folder plus one line in a Go catalog and a PR, until [#199](https://github.com/autonomous-ai/autonomous-os/issues/199) makes `skills/` the catalog. Level one is the OpenClaw workflow unchanged; level two is the part we still owe you.

A skill is one folder with one file: two front-matter keys, then markdown telling the agent what to do and when. It is the same `SKILL.md` OpenClaw and Claude skills use — a markdown skill folder drops in as-is (if it shells out to a CLI, that CLI has to be on the board too); a *robot* skill is one that also writes `[HW:/…]` markers or calls HAL. Here is the top of `guard`'s, trimmed:

```markdown
---
name: guard
description: Guard mode for security monitoring. Toggle on/off when a friend says "guard mode", "watch the house", "I'm going out" ...
---
# Guard Mode
1. Reply with `[HW:/emotion:{"emotion":"acknowledge","intensity":0.7}]` — the device nods and flashes green.
2. Enable guard mode: `curl -s -X POST http://127.0.0.1:5000/api/guard/enable`
3. Confirm verbally: "Guard mode on. I'll keep watch."
...
```

The `[HW:/path:{json}]` marker is the grammar: `{json}` is optional (`[HW:/led/off]` is fine), and the markdown-link mangling LLMs produce (`[Lights off](HW:/led/off)`) is rewritten to the canonical form — two regexes plus a normalizer in [`handler_hw.go`](system/server/agent/delivery/http/handler_hw.go). Each skill maps to the capabilities it needs ([`system/skills/skills.go`](system/skills/skills.go)), so the same file runs on any body that declares them.

Add one to your robot — one folder, live on the next conversation:

```bash
make push-skill SKILL=./my-skill TARGET=pi@lamp-xxxx.local   # live on the next conversation, no reboot
```

Same folder OpenClaw already uses. No reboot, no PR, no Go. (Or type what you want in the app, or tap one in the Skill Store.) Ship it to every robot:

1. `python skills/skill-creator/scripts/quick_validate.py skills/<name>` checks the format.
2. One line in `Catalog` in [`system/skills/skills.go`](system/skills/skills.go), plus one in `Capability` if it touches hardware; `go test ./system/skills/`.
3. Open the PR. After merge we push the skill feed (`make upload-skills` — a maintainer step for now, not CI); every body's skill watcher pulls it within 5 min and tells the agent to re-read.

That Go line and the maintainer step are the gap between this and "one folder, one PR, every robot" — [item 2](#not-built-yet--claim-one) on the list. [`skill-creator`](skills/skill-creator/) also ships an eval loop — with-skill vs baseline runs, a grader, a description optimizer — so you can measure a skill before you publish it.

## Port a robot: three files and one driver

If you make a robot, porting it is three markdown files and one driver, and what you get back the day it merges is a product: a one-line installer for your customers, every skill your hardware supports, six brains, the app's Add-robot flow, OTA and a live monitor — and every skill written from then on. Reachy Mini got all of it for ~2,900 lines — an 868-line driver, 1,875 lines of installer and unit scripts, 189 lines of declarations — over two weeks of commits (2026-07-21 → 08-04), with no change to Pollen's stack.

Skills are shared; a new body brings its own `DEVICE.md`, `SAFETY.md` and `SOUL.md` in `devices/<id>/`, plus one Python driver class if the hardware is new. `make new-device NAME=<id>` copies [`devices/_template/`](devices/_template/) to start you off. `DEVICE.md` is the whole idea in one file — the OS mounts only what you declare and refuses to boot on a board you didn't list:

```yaml
---
schema: autonomous.device.v1
id: my-robot
name: My Robot
type: mobile_robot
boards: [raspberry_pi_5]
gateway: { default: openclaw }
capabilities:
  audio:  { routes: [audio, speaker, voice], required: true }
  vision: { routes: [camera], driver: opencv, required: true }
  motion: { routes: [servo], driver: my_sdk, required: true, safety: SAFETY.md#motion }
  system: { routes: [system], required: true }
soul_ref: SOUL.md
safety_ref: SAFETY.md
---
```

Your compute needs 64-bit arm64 Linux with systemd, ~4 GB free (the installer brings its own Python 3.12), and a `/proc/device-tree/model` string matching a [`boards.json`](hal/board/boards.json) entry — one JSON entry per board; x86 is on the [list](docs/not-built-yet.md).

**Frozen:** the `autonomous.device.v1` schema (fields only added) and the capability names in [`capabilities.md`](devices/contract/capabilities.md) (never removed). **Not frozen yet:** the driver protocols (`MotionService`, `MediaOwner`) and the HAL route paths skills call (`/servo/aim`, `/emotion`) — both can move between releases, which is why ports live in-tree; port against a tag (`v0.1.4`, 2026-08-12). The motion contract is joint-space, proven on a 5–6 DOF head; an arm is the same `MotionService` with more joints (untested in-tree); wheels and legs need `LocomotionService`. `hal/` is GPL-3.0 — wrapping a permissive vendor SDK is fine (`reachy_service.py` imports Pollen's Apache-2.0 `reachy_mini`, and that is the whole driver); a closed SDK goes out of process ([#204](https://github.com/autonomous-ai/autonomous-os/issues/204)).

*Autonomous-compatible* is a written definition and a test, not our opinion: [`COMPATIBILITY.md`](devices/contract/COMPATIBILITY.md) is 16 numbered rules — 8 MUST, 4 SHOULD, 1 MAY, 3 MUST NOT — and [`devices/contract/cts/`](devices/contract/cts/) checks them in two halves — `make cts` reads every `DEVICE.md` on any laptop, `make cts-runtime TARGET=<ip>` probes a running body. No fee, no contract, no sign-off from us: pass both, open the PR. What the suite still can't check is listed by name in [`cts/README.md`](devices/contract/cts/README.md#not-covered-yet) — including rule 6, the immediate deterministic stop: no body in this repo has one yet (`/servo/release` travels to idle before cutting torque), so read that rule as where the contract is going, not as something we pass today. Fixing it is [#201](https://github.com/autonomous-ai/autonomous-os/issues/201).

Open an issue titled `port: <robot>` before code — we answer the interface questions there (which `type`, which routes, whether you need `owner:` or `LocomotionService`) and review the PR when both CTS halves are pasted in. Every step: [`docs/porting-a-robot.md`](docs/porting-a-robot.md).

## Contribute

PRs welcome, vibe-coded ones included — [`CONTRIBUTING.md`](CONTRIBUTING.md) has the norms, and the interface everyone builds on is [`devices/contract/`](devices/contract/), so open an issue before changing that. Questions, half-built ports and show-and-tell: [Discussions](https://github.com/autonomous-ai/autonomous-os/discussions).

| You want to… | You write… | Start from |
|---|---|---|
| Teach every robot something new | `skills/<name>/SKILL.md` | [`skills/guard/`](skills/guard/) · [`skill-creator`](skills/skill-creator/) |
| Run Autonomous on your robot | `devices/<id>/DEVICE.md` + `SAFETY.md` + `SOUL.md` | [`devices/reachy-mini/`](devices/reachy-mini/) — a third-party port, end to end |
| Support new hardware (open SDK) | a class in `hal/drivers/<subsystem>/` + one factory line | [`motors/reachy_service.py`](hal/drivers/motors/reachy_service.py) · [`camera/rpicam_capture_device.py`](hal/drivers/camera/rpicam_capture_device.py) |
| Support new hardware (closed SDK) | a small HTTP service speaking `MotionService` — [#204](https://github.com/autonomous-ai/autonomous-os/issues/204), not in-tree yet | [`base.py`](hal/drivers/motors/base.py) |
| Support a new board | one entry in `hal/board/boards.json` | [`boards.json`](hal/board/boards.json) |
| Add a brain | an `AgentGateway` implementation (76 methods, Go) in `runtimes/<name>/` + one factory case — the heaviest path | [`docs/agentic/adding-agent-runtime.md`](docs/agentic/adding-agent-runtime.md) · [`runtimes/opencode/`](runtimes/opencode/) |
| Ship an app people install with one click | a Python plugin against the plugin API | [`integrations/community-apps/plugin-template/`](integrations/community-apps/plugin-template/) · [plugin system](docs/plugin-system.md) |
| Add a voice — STT, TTS, or a realtime provider | a subclass in `hal/drivers/voice/` or `hal/realtime/voice_agent/` | [`voice_agent/qwen_realtime.py`](hal/realtime/voice_agent/qwen_realtime.py) |
| Turn a chat platform into a robot sense | a small Go program posting to `/api/sensing/event` (the web-chat bridge is ~230 lines) | [`integrations/chat-bridges/`](integrations/chat-bridges/) |
| Give the robot new eyes (a perception model) | a predictor in `integrations/perception-service/` or `hal/drivers/sensing/perceptions/` | [`perception-service/`](integrations/perception-service/) |
| Add a safety bound | a field + pure gate in `hal/safety/policy.py`, documented in [`SAFETY-SPEC.md`](devices/contract/SAFETY-SPEC.md) | [`policy.py`](hal/safety/policy.py) |
| Make the CTS stricter | a probe in `devices/contract/cts/` | [`test_runtime.py`](devices/contract/cts/test_runtime.py) |

### Running a fleet

Ten robots is the same install ten times. There is no fleet view, no per-device config and no inventory API — one robot per **Add robot**, every robot pulling the same skill feed and the same OTA floor. Point `OTA_METADATA_URL` at your own feed and you control exactly what ships and when. What will bite you, in order: the brains run as root beside `/dev/ttyACM0` ([#203](https://github.com/autonomous-ai/autonomous-os/issues/203)); OTA is unsigned zips over HTTPS every 5 min with no rollback ([#202](https://github.com/autonomous-ai/autonomous-os/issues/202)); the stop command commands a move — `/servo/release` travels to idle *then* cuts torque, and nothing aborts a move in flight, so no body here passes COMPATIBILITY rule 6 ([#201](https://github.com/autonomous-ai/autonomous-os/issues/201)). Until those land, keep a fleet on your own feed and off the public internet.

### Not built yet — claim one

Each is an open issue labelled [`claim-me`](https://github.com/autonomous-ai/autonomous-os/issues?q=is%3Aissue+is%3Aopen+label%3Aclaim-me); comment to take it. The first three flip the flywheel:

1. [**A mock body**](https://github.com/autonomous-ai/autonomous-os/issues/200) — `devices/sim/` on Pollen's `reachy-mini-daemon --sim` (same `:8000` API our driver already speaks) plus a `sim` board entry: a `DEVICE.md`, a board entry and glue, no new driver. **The day it merges: anyone with a laptop can run every skill in this repo — and Reachy Mini Lite works.**
2. [**Bring-your-own LLM endpoint for the OpenClaw brain**](https://github.com/autonomous-ai/autonomous-os/issues/198) — a base-URL + key override in `runtimes/openclaw/service_setup.go`, one config field and the call that reads it. The day it merges: a fully local robot, no account, Ollama on your LAN.
3. [**A skill catalog that reads `skills/`**](https://github.com/autonomous-ai/autonomous-os/issues/199) instead of a Go map (`Catalog` + `Capability` in one file), and CI publishing the feed on merge. The day it merges: "one folder, one PR, every robot" is literally true.

Seven more are open and labelled [`claim-me`](https://github.com/autonomous-ai/autonomous-os/issues?q=is%3Aissue+is%3Aopen+label%3Aclaim-me): a real `POST /servo/stop` ([#201](https://github.com/autonomous-ai/autonomous-os/issues/201)), signed OTA ([#202](https://github.com/autonomous-ai/autonomous-os/issues/202)), unprivileged runtimes ([#203](https://github.com/autonomous-ai/autonomous-os/issues/203)), an out-of-process motion driver ([#204](https://github.com/autonomous-ai/autonomous-os/issues/204)), the Go2-W port ([#205](https://github.com/autonomous-ai/autonomous-os/issues/205)), a LeRobot policy behind the marker ([#206](https://github.com/autonomous-ai/autonomous-os/issues/206)), and Reachy tracking, Hub moves and a dashboard app ([#207](https://github.com/autonomous-ai/autonomous-os/issues/207)).

More — a ROS 2 `MotionService`, x86 boards, route stability, a screen body, moving the contract's parsers out of GPL `hal/` — in [`docs/not-built-yet.md`](docs/not-built-yet.md).

Build locally:

```bash
make os-build && make os-test          # Go daemon, cross-compiled to linux/arm64
(cd hal && uv sync) && make hal-dev    # HAL on :5001 with reload
make web-install && make web-dev       # setup + monitor UI
make cts                               # is this a valid Autonomous device?
```

## Every layer is a folder

One turn, top to bottom:

1. **Sense** — a mic or camera event arrives (`hal/`).
2. **Route** — `intent` answers fixed commands from a local table, no model, no network (`system/intent/`); everything else goes to the brain and takes seconds. We have not put a stopwatch on either end to end — `make latency` is [wanted](docs/not-built-yet.md), and until it exists treat every timing here as an order of magnitude, not a spec.
3. **Or answer straight away** — if the turn arrived as speech, a realtime voice model in `hal/realtime/` has it too and runs beside this path: it either replies in under a second or hands the turn up to the brain ([`docs/realtime-voice.md`](docs/realtime-voice.md)).
4. **Think** — the brain reads its `SOUL.md` and the installed `SKILL.md`s and replies in plain text with `[HW:/path:{json}]` markers inside (`runtimes/`).
5. **Strip** — `os-server` pulls each marker out (`system/server/agent/delivery/http/handler_hw.go`), drops any whose capability this `DEVICE.md` doesn't declare, and POSTs the rest to HAL (:5001) before the words are spoken.
6. **Mount** — HAL serves a route only if `DEVICE.md` lists it under a declared capability (`hal/board/device.py`). Intern declares `light` but only the `led` route, so `/scene` never mounts on it.
7. **Clamp** — the safety gate, a pure function of `SAFETY.md` (`hal/safety/policy.py`), bounds brightness, quiet hours and explicit-move speed. Recorded moves — the `[HW:/emotion:…]` path on Lamp — and the 15 fps vision tracker are not speed-gated yet.
8. **Move** — the driver talks to the hardware (`hal/drivers/`), and the body acts.

If you run ROS 2, nothing here replaces your stack: no kernel, no bus, no planner, no real-time claims, and nothing above the drivers has a deadline — position control closes in the STS3215 firmware on Lamp and in Pollen's daemon on Reachy, never in HAL. What it adds is the part ROS never had a package for: a brain that reads markdown, a body file it can reason about, a safety clamp below the brain, a skill format shared with the agent world. A `MotionService` that publishes `JointTrajectory` makes your robot a driver here, not a rewrite ([wanted](docs/not-built-yet.md)).

Read the figure top down. Every row is a folder in this repo — the table below is the map — and four of them are governed by a markdown file. Left margin is *act* — a skill writes `[HW:/servo/aim]`, os-server strips it and POSTs it down through HAL to the body. Right margin is *sense* — hardware events → `intent` → an agent turn. Dashed boxes are open slots and declarations: Go2-W is declared and not running; `your skill`, `your brain`, `your board`, `your robot` are where you plug in. A body is a PR, not a fork.

![Autonomous OS stack, top down: 25 skills, six swappable agent runtimes, 14 Go system packages, the realtime voice agent, the 13 declared capabilities, a deterministic safety gate below them (brightness, quiet hours, explicit-move speed, thermal today), in-tree drivers and board profiles, the vendor Linux kernel, and the bodies — each row labelled with its repo folder.](docs/architecture/autonomous-stack.png)

| Row in the figure | Folder | Governed by |
|---|---|---|
| Skills | [`skills/`](skills/) | `SKILL.md` |
| Agentic runtime | [`runtimes/`](runtimes/) | `SOUL.md` |
| System managers | [`system/`](system/) | — |
| Realtime voice | [`hal/realtime/`](hal/realtime/) | — |
| Capabilities | [`devices/contract/capabilities.md`](devices/contract/capabilities.md) → [`hal/routes/`](hal/routes/) | `DEVICE.md` |
| Safety gate | [`hal/safety/`](hal/safety/) | `SAFETY.md` |
| Drivers | [`hal/drivers/`](hal/drivers/) | — |
| Boards | [`hal/board/boards.json`](hal/board/boards.json) | — |
| Linux | the vendor kernel, not ours | — |
| Bodies | [`devices/`](devices/) | `DEVICE.md` |

Five of those rows carry detail the figure can't hold:

- [`system/`](system/) — the Go daemon (`os-server`, :5000), one package per box in the figure: `server` strips markers and proxies HAL, `agent` switches brains, `intent` answers fixed commands, plus `network`, `monitor`, `healthwatch`, `ambient`, `device`, `skills`, `plugin`, `statusled`, `vision`, `buddy`; `bootstrap` is OTA, its own binary.
- [`devices/contract/capabilities.md`](devices/contract/capabilities.md) → [`hal/routes/`](hal/routes/) — the 13 capability names a `DEVICE.md` declares. Ten mount HAL routes (111 endpoints, live Swagger at `/api/hardware/docs`); `presence` and `lifelike` are routeless loops, `companion` is os-server's `buddy`.
- [`hal/safety/`](hal/safety/) — the clamp. SoC temperature polled every 10 s; over `max_temp_c` it stops the tracker — not moves, LEDs or audio. Stop is `POST /servo/release`: it moves to idle, then cuts torque — it does not abort a move in flight. No hardware e-stop; runtimes run as root. Confirmed on hardware vs unit-tested: [`docs/safety.md`](docs/safety.md).
- [`hal/drivers/`](hal/drivers/) — one folder per subsystem. Lamp's five STS3215s speak through LeRobot's Feetech bus (pinned in `hal/pyproject.toml`), Reachy through Pollen's `reachy_mini` SDK, and Lamp's 30 move recordings are leader-arm teleop (`hal/record.py`); `skills/servo-control/SKILL.md` names 23 of them for the agent.

Off the device: [`integrations/`](integrations/) and [`scripts/`](scripts/). Long form: [architecture](docs/architecture/overview.md) · [HAL](docs/architecture/hal.md) · [device spec](devices/contract/DEVICE-SPEC.md) · [capabilities](devices/contract/capabilities.md) · [safety](docs/safety.md) · [developer guide](docs/developer-guide.md).

## License

Everything outside `hal/` is Apache-2.0. `hal/` is GPL-3.0 — a handful of modules, the `follower/` package and the teleop recordings are inherited from [LeLamp Runtime](https://github.com/humancomputerlab/lelamp_runtime) ([`hal/UPSTREAM.md`](hal/UPSTREAM.md) lists them); the rest of `hal/` is ours, kept GPL by choice so the tree has one license per top-level folder. A driver you commit there is GPL, so a closed vendor SDK wraps out of process. Security issues: [`SECURITY.md`](SECURITY.md).
