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
| `input-branching` | System | routing, realtime, internal | Lamp, Intern v2, Reachy Mini |

Compatibility is the automatic built-in installation gate from
`system/skills.Capability`: a skill with no entry is platform logic and runs on
every current device. It does not describe optional integrations; for example,
`computer-use` additionally requires an owner-paired Mac before it can act.

## Categories

```text
Home
Health
Entertainment
Work
Safety
Camera & Vision
Communication
Utilities
```

`input-branching` is routing infrastructure rather than a user-installable
feature. Keep it out of the default storefront, or render it with a `System`
badge when it must be visible.
