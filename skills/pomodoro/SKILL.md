---
name: pomodoro
description: Desk timer. Use when they say pomodoro, focus timer, 25 minutes, start a work block. The OS owns the clock; you only start/stop and announce.
---

# Pomodoro

Requires `pomodoro: true`. The dashboard Start/Stop buttons hit the OS timer. Voice should do the same via curl (LAN):

```bash
curl -s -X POST http://127.0.0.1:5000/api/device/behaviors/pomodoro/start
curl -s -X POST http://127.0.0.1:5000/api/device/behaviors/pomodoro/stop
```

Those paths are admin-gated — if curl 401s, tell them to tap Start on Settings → Behaviors.

When a `[companion:pomodoro]` message arrives, the OS already flipped the phase. Speak one or two sentences (work done → break; break done → back to it). Emotion `happy` or `acknowledge`. Do not start another timer yourself.

Kids → skip, or treat as a short game timer if they asked.
