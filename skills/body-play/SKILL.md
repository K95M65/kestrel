---
name: body-play
description: Physical play flags from Settings → Behaviors. Use when they say marionette, grab my head, radio, antenna tuner, hand tracking, mime, telepresence, idle breathing, turn toward my voice.
---

# Body play

These are dashboard flags. If the matching `[behaviors]` key is false, decline. If true but HAL has no matching endpoint, say so honestly — do not pretend to fetch keys or walk the room.

| Flag | They said | You do |
|---|---|---|
| `idle_motion` | (always on when idle) | Do not call `/emotion` idle. Breathing/antenna sway is HAL's job. You may stay still. |
| `doa` | "look at me when I talk" | Aim at `user` on a spoken turn. Sub-100ms turn-to-mic is HAL when the array supports it. |
| `layered_motion` | — | You may stack speech wobble + one emotion clip + face track. Do not queue three full dances. |
| `marionette` | "marionette", "record this move" | If HAL record/replay is available, record; else tell them to use the desktop marionette app. |
| `hand_track` | "follow my hand", "mime me" | `/servo/track` with hand if tracking works; else say the body cannot follow hands yet. |
| `radio` | "radio", "tune the antenna" | Play a radio-like YouTube/query via music skill; antenna-as-tuner is HAL-pending. |
| `telepresence` | "let me drive", "see what you see" | Point them at the camera page / official iOS app. Do not open an unauthenticated tunnel. |

Kids: marionette and radio OK; telepresence off unless a parent asked.
