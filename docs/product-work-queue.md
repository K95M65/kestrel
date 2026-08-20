# Product work queue

## Chosen next P0 — kinetic safety on wake/resume (2026-08-20)

Audit after recorded-play gating: the remaining snap was SDK `wake_up()` on
HAL start and `/servo/resume` — a fixed 2s goto from the sleep fold (~±175°)
to INIT (~±10°), ~82 deg/s, above Reachy's `motion.max_speed: 60`, plus a
sound that can abort with `media_backend="no_media"`. That path now uses
`_goto_awake_pose` / `min_move_duration` from the live pose (sleep-fold bound
if the pose is unreadable — not the vendor 2s snap). Recovery
(stop/halt/release/zero/hold) stays ungated. Other pillars deferred this slice.

## Previous P0 — kinetic safety on recorded playback (2026-08-20)

Audit of the seven standing pillars: the single highest-priority gap was that
`motion.max_speed` already stretched `/servo/move`, aim, and nudge, but **not**
the path the companion actually uses to move — recorded/emotion play
(`_continue_playback` / Reachy `play_move`). A Hub or library move could jerk
past the SAFETY.md ceiling. That gate is now in the play path
(`damp_recorded_actions`, Reachy `_stretch_move`); stop/release/zero/hold stay
ungated. Other pillars (README overhaul, persona picker, SKILL templates,
hardware-grey UI string, five full-audit loops) deferred this slice.

Notes from walking Guided Setup and the public rooms on the desk Reachy Mini
(`10.10.2.160`, 2026-08-19). Grounded in the current web 0.1.18 / os-server 0.1.20
code, not just the surface copy.

**P0 + P1 guide rewrite are in this tree** (os-server 0.1.20, web 0.1.18):
session reset on rename, live camera enroll, `guide=1` consumed once, Test Voice
toast + mute, name-family wake chips, intro/preset/mornings/privacy/done copy.
People-as-contact-book and Device polish stay later. Hardware notes:
[`robots/reachy-mini/docs/hardware.md`](../robots/reachy-mini/docs/hardware.md).
What we changed vs stock Autonomous OS (and how to watch upstream for
fixes to re-port): [`docs/divergence-from-stock.md`](divergence-from-stock.md).

This is the working list. P0 is broken or misleading *now*. P1 is the product
shape we already agreed. P2 is later polish and platform.

---

## P0 — broken or misleading on this unit

### 1. Talk-in-setup uses the old name
After step 2 (name), step 3 sends a real chat turn, but the brain still answers
as **Reachy**.

**Why:** `SetIdentity` writes `IDENTITY.md` and `i18n.SetDeviceName`, but it does
**not** reset the OpenClaw session. The live gateway already loaded the old
identity. WatchIdentity only pushes wake words, ~5s later.

**Fix:** After a successful rename, `NewSession` (or equivalent) so the next turn
reads the new `**Name:**`. Wait for ack before enabling Send. Prefer a spoken
greeting (“Hi, I’m {name}”) rather than a silent text box — step 3 is “talk”,
not “type in the dashboard”.

### 2. “Who is this?” is a stale snapshot, not an enroll ritual
Step 4 of 9 shows a JPEG, often black or yesterday’s frame. **New photo** looks
dead. Copy says “Nobody it knows yet.”

**Why:**
- Guide uses a single `/camera/snapshot?t=` `<img>`, not the live MJPEG stream.
- The `<img>` has no React `key`, and the snapshot response likely gets cached
  by the browser/nginx even with `?t=`.
- New photo only bumps `snapTs`. If `camOff` is true the button is disabled
  with no explanation.
- “Nobody it knows yet” is `GET /face/current-user` while nobody is enrolled —
  true, but the step should be *creating* that person.

**What it should be:** Reachy *asks* you to look at its face, gives audio cues
until you’re framed, takes the picture, then asks who you are (voice). You
answer; it enrolls that name. Dashboard fields are fallback, not the ritual.

### 3. House reopens Guided Setup
Clicking **House** (or its first child, Behaviors) pops the guide again.

**Why:** Home’s “Guided setup” navigates to `/setting?guide=1#behaviors`. That
query sticks. `BehaviorsSection` does `if (q.get("guide") === "1") setGuide(true)`
on every visit. Also, if you close the overlay without **Save this setup**,
`onboarded` stays false, so Home and Behaviors keep nagging.

**Fix:** Consume `guide=1` once (strip it). Don’t auto-open after `onboarded`.
House click should expand the group or go to People — not Behaviors-with-modal.

### 4. Test Voice appears to do nothing
`POST /api/voice/preview` exists and speaks through HAL. The button is
fire-and-forget: no toast, no spinner. Home showed **Speaker MUTED**, so the
preview is silent. No local sample clips for OpenAI / ElevenLabs.

**Fix:** Unmute for the test (or warn). Toast success/fail. Optional: play a
short sample in the browser for each listed voice so you can pick without
the body speaker.

### 5. Wake-word chips don’t follow the new name in the way people expect
Default `hey {name}` is treated as “not exclusive”, so HAL still merges
`hey autonomous` / `hey reachy-mini` / `hey {name}`. General’s list is the
whole family, and it only refreshes after Apply if the identity API returns
new phrases. If Apply failed or the page was already loaded, chips stay old.

**Fix:** Public UI should show **this robot’s** phrases (`wake up {name}`,
`hey {name}`, …). Permanent aliases belong behind Advanced. After naming,
reload General from `IdentityPublic.wake_phrases`.

---

## P1 — Guided Setup rewrite (the walk)

Current 9 steps: intro → name → text chat → snapshot → preset → mornings →
“how careful” → chips → “this is the setup”.

### Intro (“Let’s try this desk”)
“Desk” is leftover from the lamp/desk-companion metaphor. On Reachy it reads
as furniture.

**Copy:** “Let’s try this robot” / “Let’s set up {body}.” One sentence: name it,
talk to it, let it see you.

### Step 2 — Name
Keep. Persist **before** talk, and don’t continue until the session actually
knows the name (P0.1). Wake default `hey {name}` is fine if chips later match.

### Step 3 — Talk
Should be **voice** (or at least TTS of the reply). Prompt: “Say hi to {name}”
or the robot speaks first. Text box is a fallback when the mic is muted.

### Step 4 — Who is this? (face)
Replace snapshot+form with the framed enroll ritual (P0.2). “Nobody it knows
yet” → “We’ll add you as the first person.”

### Step 5 — Who is this for? (presets)
Today: Just me / Family / Kids around / Office as one-line chips. Selecting
one only highlights it.

**Need:** After a pick, a **second pane** (right, or stacked on phone) with
what actually turns on/off and the differentiators, e.g.

| | Just me | Family | Kids | Office |
|---|---|---|---|---|
| Morning brief | on | on | off | on |
| Greet everyone | yes | yes | yes | named only |
| Mail/calendar | on | on | **off** | on |
| Dance | on | on | on | **off** |
| Focus / pomodoro | off | off | off | on |

### Step 6 — Mornings
Current lead: “A spoken briefing before you open five apps — the thing people
actually keep.” That doesn’t mean anything to an owner.

**Copy:** “Do you want a morning brief — weather, calendar, overnight mail,
news — spoken at a time you pick?” Time picker on this step (not buried).
Skills/connectors that *feed* the brief (Gmail, calendar, news) are a later
**Jobs** screen, not this question.

### Step 7 — How careful?
Three independent toggles (draft-not-send, camera on demand, face-follow after
wake) with no frame. Unclear what “careful” is.

**Default: locked down.** Camera off until asked, no stranger tracking, drafts
never send. Skip this step in onboarding. Unlock later in the app under a
named **Privacy** page.

### Step 8 — When you’re around
Chip storm. Fine as optional “extras”, or fold into the preset pane.

### Step 9 — Done
Not “This is the setup”. Something like **“You’re set”** / **“Quick setup
complete.”** Unique summary of *this* robot (name, wake, who it’s for, brief
time, first person). One line: “Change any of this later under House and Device.”

---

## P1 — House

### Quiet hours / any time field
- 12h / 24h toggle, remembered.
- Clock picker, not a raw `type=time` where we can avoid it.
- Schedule: **per weekday** on/off, edit, remove. Not one window for all days
  with a chip row.

### People → contact book
Today: household list + Add a face + My Voice dumped on one page.

**Shape:**
- A **contact book**. Each person: face from enroll (photo at capture, not a
  stale load), voice samples, how they talk to the robot.
- **“Add a friend”** mode, also voice: “Reachy, let’s add a friend” → same
  framed capture as setup.
- Optional **sync contacts** (phone / Mac / Google) behind an explicit grant.
- **Add a face** and **My Voice** are not Device leaves. They’re actions on a
  person. Recording must target the selected contact; “Start Recording”
  disabled until a name is chosen is the current trap.

House must **not** reopen the guide (P0.3).

---

## P1 — Device

### Camera
Live MJPEG (or a snapshot that is forced fresh every tap). No cached still as
the main view. If the camera is off, say so and offer Turn on.

### Voice (TTS)
Local/browser samples for OpenAI + ElevenLabs (and Grok/Piper if listed).
Test must unmute or warn, then speak, with visible status.

### General
Wake phrases rebuilt from the assigned name (P0.5). “Wait for the wake word”
explained in one line: “It only listens after you say one of these.”

### Wi-Fi
Show **currently connected** SSID, signal, and **Forget / remove**. Changing
networks is secondary.

### Channels
Not a token form first. Explain: these are how **you** message **this robot**
when you’re not in the room (Telegram / Slack / Discord). List what’s
connected. Then credentials.

### Plugins
No catalog yet (HF browse is parked). Need:
- Our plugin format + a **trusted git org/repo** of examples.
- Install from that list.
- “Add your own URL” behind an extra confirmation / override.

---

## P1 — Advanced (keep gated)

### Logs
Robot / App / Brain tabs exist, but the body is a tail of raw lines. Need
levels as first-class, request-id grouping, and **ship to a log server**
(optional URL). Not a dump.

### Realtime
This is the **audio-native brain** (Gemini Live / OpenAI Realtime / Qwen Omni)
that hears continuously, separate from the turn-based **AI Brain**. Copy must
say that. Providers on the form: gemini, openai, qwen, none. Grok live is
missing if we want it. Default off or clearly “experimental”.

### AI Brain
This is the **base model for everything** (skills, Talk, briefs, jobs).
Say that. Don’t call it a generic “LLM URL”.

### Agent runtime
Implemented adapters: OpenClaw (default, full skills + voice path), Hermes,
PicoClaw, Codex, Claude Code, OpenCode.

**Honest split:** OpenClaw and Hermes are the ones that can run the companion
loop we actually ship (skills, IDENTITY, wake, TTS). Codex / Claude Code /
OpenCode are coding CLIs behind a bridge — they do **not** automatically get
morning brief, greeter, or HAL tools. Switching is a systemd event, not a
dropdown curiosity. Hide coding CLIs unless Advanced + a warning.

### System / storage (this robot)
14 GB eMMC, **12 GB used, 1.7 GB free (88%)**.

| Where | Size | Notes |
|---|---|---|
| `/opt/hal/.venv` | 2.4 GB | HAL Python (cv2, etc.) — expected |
| `/venvs` (Pollen) | 1.4 GB | `mini_daemon` + `apps_venv` — do not delete |
| `/usr` | 3.7 GB | Debian |
| `/opt/gst-plugins-rs` | 437 MB | Pollen/GStreamer |
| `/restore` | 475 MB | Pollen recovery image? |
| `/opt/hal/.uv-cache` | 260 MB | Safe to prune |
| `/root/.npm` | 131 MB | Safe to prune |
| `/var/log` | 52 MB | Rotate; os-server already splits 2 MB files |

**Need:** Device → Storage: breakdown, “what we can clean”, optional auto-prune
every N days (uv cache, old os-server logs, npm, snapshots). Never touch
`/venvs` or `/restore` without a labeled “Pollen” warning.

### Flow
Turn pipeline debugger: mic/chat → intent → agent → tools → TTS. Useful when
a turn fails. Not a product room. Keep Advanced; add a one-line “This is a
map of the last thing it heard and did.”

### Sensing
Presence, sound, light, fire — raw HAL. Needs plain labels: “Someone walked
in”, “Loud sound”, “Room dark”. Thresholds belong behind that, not instead of
it.

### Servo
Motion recordings / aim / track. On Reachy this is the head. Label it **Motion**
and say “nod, look, dance clips” instead of “servo”.

---

## P2 — later

- Jobs UI for brief sources (mail, calendar, news) after connectors exist.
- Voice enroll on the robot mic, bound to a contact.
- Contact sync providers.
- Trusted plugin registry.
- Remote syslog / OpenTelemetry.
- 12/24h preference as a single Device setting used everywhere.
- Code consolidations after the IA settles: one People surface, wake-word
  helper used by General + Guide, snapshot helper with `Cache-Control: no-store`
  + `key=`, strip `guide=1` in one place, identity `NewSession` next to
  `UpdateIdentityName`.

---

## Suggested build order

1. ~~P0: session reset on rename; live camera in setup; New photo cache-bust;
   consume `guide=1`; Test Voice feedback + mute warning; name-based wake chips.~~
   **Done in 0.1.20 / 0.1.18.**
2. ~~Guide copy + locked-down privacy default + done screen + preset detail pane.~~
   **Done in 0.1.20 / 0.1.18.** (12/24 and per-day quiet hours still later.)
3. ~~People as contact book; move face/voice off Device.~~ **Done in web 0.1.20.** (Add a friend = live camera; photo/voice are actions on a person. Device face/voice leaves stay Advanced-only.)
4. Device: live camera, Wi-Fi current/forget, channels explainer, TTS samples.
5. Storage drill-down + safe prune.
6. Advanced labels (Realtime, Brain, runtimes, Flow, Sensing, Motion, Logs).

Do not flash Reachy. HAL/Pollen stay put unless a change needs them.
