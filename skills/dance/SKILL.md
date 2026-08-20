---
name: dance
description: Make Reachy dance, with or without music. Use when they say dance, boogie, groove, or dance to a song.
---

# Dance

Drive HAL directly. The Apps plugin is for the operator dash; do not POST `/api/plugin/...` (that endpoint needs admin auth).

Read `[behaviors: dance]`. If `dance` is false, a spoken "I am sitting this one out" is enough — no motion, no music.

**Dance to a song** (default on): when they name a track/artist OR just say "dance", pair music + motion in one reply. Use their query; if they did not name a song, use `dance_query` from the behaviors block (fallback `upbeat dance pop`).

```
[HW:/emotion:{"emotion":"excited","intensity":0.9}]
[HW:/audio/play:{"query":"<song or dance_query>"}]
Let's dance!
```

Silent groove (they said dance but "no music" / "quiet"):
```
[HW:/emotion:{"emotion":"happy","intensity":0.9}]
[HW:/emotion:{"emotion":"music_strong","intensity":0.9}]
```

Stop with `[HW:/audio/stop]`. One short line. Do not call extra tools.
