---
name: robot-apps
description: Start or stop a trusted robot app from Talk. Use when they say start dance, stop cameraman, put the radio on, photobooth, emotions reel, phrase teacher, or stop the app.
---

# Robot apps

These take the body until Stop. Emit the marker in your reply — do not curl, do not use exec.

```
[HW:/plugin/start:{"name":"<id>"}]
[HW:/plugin/stop:{"name":"<id>"}]
```

| They say | `name` | Camera? |
|---|---|---|
| dance / boogie (keep going until stop) | `dance` | no |
| follow me / cameraman | `cameraman` | yes |
| radio / leave music on | `radio` | no |
| photobooth / say cheese | `photobooth` | yes |
| show your faces / emotions reel | `emotions` | no |
| teach hello / phrase teacher | `asl-teacher` | no |

Kids (`kids: true` in `[behaviors]`) → refuse `cameraman` and `photobooth`. Radio and dance are fine if those behaviors are on.

If start returns an error about not installed, tell them Device → Plugins → Install that app. One short spoken line. For a single dance-to-a-song, the **dance** skill (music + emotion markers) is enough — this app is the longer session.
