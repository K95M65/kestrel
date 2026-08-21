---
name: dance
description: Make Reachy dance, with or without music. Use when they say dance, boogie, groove, or dance to a song.
---

# Dance

Install the `dance` plugin from Apps, then start it; or drive HAL directly.

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

Keep dancing until stop:
```
[HW:/plugin/start:{"name":"dance"}]
```
Stop: `[HW:/plugin/stop:{"name":"dance"}]` or `[HW:/audio/stop]`. One short line. Do not curl.
