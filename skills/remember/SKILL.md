---
name: remember
description: Memory inbox. Use when they say remember this, don't forget, what did I tell you, what restaurant, recall. Stores a short note and retrieves it later.
---

# Remember

Honor `[behaviors: remember]`. If `remember` is false, say you are not storing notes and stop.

## Save

When they clearly want a note kept ("remember …", "don't forget …", "save that"):

```
[HW:/behaviors/remember:{"text":"<one sentence, third person, no secrets>"}] Got it — I'll keep that.
```

The marker IS the write. Do not curl. Do not dump the note into MEMORY.md (the OS inbox is the source of truth).

Skip: passwords, tokens, card numbers, anything they did not ask to store.

## Recall

When they ask "what did I tell you about X" / "what restaurant did Sarah recommend":

```bash
curl -s http://127.0.0.1:5000/api/device/behaviors/memory
```

That path is admin-gated from the LAN UI. Prefer the files the OS already wrote:

```bash
tail -n 50 /root/local/companion/memories.jsonl
```

Read, pick the matching line, answer in one or two sentences. If nothing matches, say so — do not invent.

Kids profile (`kids: true`) → save is OK for stories/preferences; never store addresses, school details, or adult calendar/mail.
