---
name: dance
description: Make Reachy dance, with or without music. Use when they say dance, boogie, groove, or dance to a song.
---

# Dance

**Keep dancing until they say stop** — start the dance app:
```
[HW:/plugin/start:{"name":"dance"}]
```
Stop with `[HW:/plugin/stop:{"name":"dance"}]`. If start fails, Install Dance under Device → Plugins, or use the one-song path below.

Drive HAL directly for a **single song**. Do not curl `/api/plugin/...`.

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
