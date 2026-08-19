---
name: cameraman
description: Follow a face with the camera/head. Use when they say follow me, keep me in frame, cameraman, track my face.
---

# Cameraman

If the `cameraman` plugin is installed:
`POST http://127.0.0.1:5000/api/plugin/cameraman/start`

Otherwise:
```
[HW:/emotion:{"emotion":"curious","intensity":0.7}]
[HW:/servo/track:{"target":"face"}]
```

Stop with `POST /api/plugin/cameraman/stop` or `[HW:/servo/track/stop]`.
One short spoken line, then track. Do not narrate PID or cameras.
