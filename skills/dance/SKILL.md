---
name: dance
description: Make Reachy dance, with or without music. Use when they say dance, boogie, groove, or dance to a song.
---

# Dance

Drive HAL directly. The Apps plugin is for the operator dash; do not POST `/api/plugin/...` (that endpoint needs admin auth).

With music:
```
[HW:/emotion:{"emotion":"excited","intensity":0.9}]
[HW:/audio/play:{"query":"upbeat dance pop"}]
```

Silent groove (no music):
```
[HW:/emotion:{"emotion":"happy","intensity":0.9}]
[HW:/emotion:{"emotion":"laugh","intensity":0.8}]
[HW:/emotion:{"emotion":"music_strong","intensity":0.9}]
```

Stop music with `[HW:/audio/stop]`. Say one short line ("Let's dance!") and do not call extra tools.
