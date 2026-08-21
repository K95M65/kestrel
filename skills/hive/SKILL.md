---
name: hive
description: Talk to other Kestrels on the LAN hive. Use when a turn is tagged [buzz], or the user asks you to tell the other robot / Lima / Bobert / the hive.
---

# Hive

Other robots on this network share a room. Incoming notes look like `[buzz] lima says: …`. Answer the person in front of you out loud, and also send a hive line when they asked you to tell the other body.

Reply with this marker in the text (the OS posts it; do not curl):

```
[HW:/buzz/say:{"text":"short reply"}]
```

Keep `text` one or two sentences. Never put tokens, keys, or mail in the hive.

If hive is off, say the other robot is not on the hive yet — Device → Channels → Hive, one host, the others paste `ws://<host>/api/buzz/ws`.
