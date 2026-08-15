# Skills Catalog

This catalog assigns every platform skill one primary category and one or more
search tags. Categories are intended for store navigation; tags let a skill be
found in multiple relevant contexts without duplicating it across categories.

System-only skills may be hidden from the default storefront or shown with a
`System` badge.

| Skill | Primary category | Tags | Compatible devices |
| --- | --- | --- | --- |
| `audio` | Utilities | speaker, microphone, volume, hardware | Lamp, Intern v2, Reachy Mini |
| `music` | Entertainment | youtube, playback, speaker | Lamp, Intern v2, Reachy Mini |
| `music-suggestion` | Entertainment | proactive, mood, recommendation | Lamp, Intern v2, Reachy Mini |
| `voice` | Communication | tts, mute, privacy, microphone | Lamp, Intern v2, Reachy Mini |
| `camera` | Camera & Vision | snapshot, streaming, privacy, vision | Lamp, Reachy Mini |
| `face-enroll` | Camera & Vision | face-recognition, identity, presence | Lamp, Reachy Mini |
| `user-emotion-detection` | Health | emotion, speech-emotion, mood, sensing | Lamp, Intern v2, Reachy Mini |
| `speaker-recognizer` | Communication | voice-id, speaker-recognition, identity | Lamp, Intern v2, Reachy Mini |
| `display` | Home | lcd, eyes, expression, hardware | No current device |
| `emotion` | Home | personality, expression, led, servo, display | Lamp, Reachy Mini |
| `led-control` | Home | lighting, rgb, effects, smart-home | Lamp, Intern v2 |
| `scene` | Home | lighting, ambiance, focus, relax, smart-home | Lamp, Intern v2 |
| `servo-control` | Home | motion, aiming, gestures, hardware | Lamp, Reachy Mini |
| `servo-tracking` | Camera & Vision | vision-tracking, object-tracking, motion | Lamp, Reachy Mini |
| `sensing` | Home | presence, sound, light, fire-safety, events | Lamp, Intern v2, Reachy Mini |
| `sensing-track` | Home | history, logs, motion, presence | Lamp, Intern v2, Reachy Mini |
| `skill-creator` | Productivity | create, test, evaluate, package, publish | Lamp, Intern v2, Reachy Mini |
| `guard` | Safety | monitoring, presence, alerts, smart-home | Lamp, Reachy Mini |
| `wellbeing` | Health | posture, hydration, breaks, coaching | Lamp, Intern v2, Reachy Mini |
| `habit` | Health | routines, behavior, personalization | Lamp, Intern v2, Reachy Mini |
| `mood` | Health | emotion, user-state, personalization | Lamp, Intern v2, Reachy Mini |
| `computer-use` | Productivity | macos, browser, desktop, companion | Lamp, Intern v2, Reachy Mini |
| `claude-buddy` | Productivity | claude-code, approvals, companion, agent | Lamp, Intern v2, Reachy Mini |
| `connectors` | Productivity | gmail, calendar, drive, notion, github | Lamp, Intern v2, Reachy Mini |
| `input-branching` | Not published | routing, realtime, internal | Lamp, Intern v2, Reachy Mini |

Compatibility is the automatic built-in installation gate from
`system/skills.Capability`: a skill with no entry is platform logic and runs on
every current device. It does not describe optional integrations; for example,
`computer-use` additionally requires an owner-paired Mac before it can act.

## Per-body notes

Why some cells in the README's capability table are blank or ○ (declared in
`DEVICE.md`, driver not landed):

- **Intern** — no camera, so Sense is sound-only, Mood is voice-only, and
  Look-after-you is breaks and habits, not posture. Glow is colors and effects,
  not the six scenes: its `light` declares `led`, not `scene`.
- **Reachy Mini** — the tracker (`servo-tracking`) still speaks Lamp's joint
  names, so it installs but does not track yet. Emotion moves are verified on
  hardware; `aim` directions are still being tuned. HAL plays moves through the
  SDK's `no_media` client while it owns the speaker, so each move's sound clip
  is dropped for now.
- **Go2-W** — no board entry yet, so HAL will not boot on it until the port
  lands. Its `motion` routes are `locomotion`, not `servo`, so `servo-control`
  and `servo-tracking` would install and have nothing to call — blank in the
  table; the `motion.drive` sub-capability under Not built yet keeps them off
  rolling bodies. `claude-buddy` would install (it only needs `audio`);
  `computer-use` needs `companion`, which Go2-W does not declare.

## Categories

```text
Home
Health
Entertainment
Productivity
Safety
Camera & Vision
Communication
Utilities
```

`input-branching` is routing infrastructure rather than a user-installable
feature, so it has no Store category and must stay out of the storefront.
`claude-buddy` cannot be published under the current Store contract because its
slug contains the reserved word `claude`.
