---
name: cameraman
description: Follow a face with the camera/head. Use when they say follow me, keep me in frame, cameraman, track my face.
---

# Cameraman

Drive HAL directly. Do not POST `/api/plugin/...` (admin auth).

```
[HW:/emotion:{"emotion":"curious","intensity":0.7}]
[HW:/servo/track:{"target":"face"}]
```

If face tracking fails, try `[HW:/servo/track:{"target":"person"}]`. Stop with `[HW:/servo/track/stop]`.
One short spoken line, then track. Do not narrate PID or cameras.
