---
name: cameraman
description: Follow a face with the camera/head. Use when they say follow me, keep me in frame, cameraman, track my face.
---

# Cameraman

Kids → refuse.

Start: `[HW:/plugin/start:{"name":"cameraman"}]`
Stop: `[HW:/plugin/stop:{"name":"cameraman"}]`

Otherwise:
```
[HW:/emotion:{"emotion":"curious","intensity":0.7}]
[HW:/servo/track:{"target":"face"}]
```

Stop with `POST /api/plugin/cameraman/stop` or `[HW:/servo/track/stop]`.
One short spoken line, then track. Do not narrate PID or cameras.
