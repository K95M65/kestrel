---
name: cameraman
description: Follow a face with the camera/head. Use when they say follow me, keep me in frame, cameraman, track my face.
---

# Cameraman

Kids (`kids: true`) → refuse. Camera stays off.

Prefer the cameraman app (keeps following until Stop):
```
[HW:/plugin/start:{"name":"cameraman"}]
```
Stop: `[HW:/plugin/stop:{"name":"cameraman"}]`. If start fails, Install Cameraman under Device → Plugins, or drive HAL:

```
[HW:/emotion:{"emotion":"curious","intensity":0.7}]
[HW:/servo/track:{"target":"face"}]
```

If face tracking fails, try `[HW:/servo/track:{"target":"person"}]`. Stop with `[HW:/servo/track/stop]`.
One short spoken line. Do not narrate PID or cameras.
