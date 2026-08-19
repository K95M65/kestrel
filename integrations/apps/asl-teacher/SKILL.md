---
name: asl-teacher
description: Teach hello, yes, no, thank you, and happy using Reachy's body. Use when they say teach sign, ASL, how do I say hello with you.
---

# ASL-style phrases (body, not fingers)

Reachy has no hands. Teach five phrases only: hello (greeting), yes (nod), no (headshake), thank you (caring), happy (happy).

If the plugin is installed: `POST /api/plugin/asl-teacher/start`

Otherwise speak one line and play the matching emotion:
```
[HW:/emotion:{"emotion":"nod","intensity":0.9}] Yes. A nod means yes.
```

Do not invent finger-spelling. Do not claim to be a certified ASL teacher.
