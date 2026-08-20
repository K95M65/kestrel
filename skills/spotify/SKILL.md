---
name: spotify
description: Open or control Spotify on the paired computer via Kestrel Buddy. Use when they say Spotify, play this on Spotify. Do NOT play through the robot speaker — that is the music skill (YouTube). Do NOT use if no computer is paired.
---

# Spotify (computer via Kestrel Buddy)

Read `[behaviors: ...]`. If `kids: true`, refuse. Do not drive the Mac.

There is no Spotify account on the robot. Playback is the Spotify app on the paired computer.

## Path

1. No Buddy paired → “No computer is paired yet. House → Uses → Spotify, or Home → Buddy.” Do not fire markers.
2. Open the app: `[HW:/buddy/exec/open_app:{"app":"Spotify"}]`
3. If they named a track/artist/playlist, also search:

```
[HW:/buddy/exec/open_url:{"url":"https://open.spotify.com/search/<urlencoded query>"}]
```

4. Confirm in one short sentence. Markers at the **start** of the reply.

“Play music” / “play this out loud” with no Spotify mention → **music** skill (YouTube on the speaker), not this one.

## Examples

| They say | You do |
|---|---|
| “Open Spotify” | `open_app` Spotify |
| “Play Bohemian Rhapsody on Spotify” | `open_app` + `open_url` search |
| “Play Bohemian Rhapsody” (no Spotify) | **music** skill |
| “Stop the music” while the robot speaker is playing | **music** `/audio/stop` |

## Rules

- Flat JSON only. One action per marker. Same `computer-use` marker grammar.
- Do not claim the song is playing on the robot.
- Do not scrape Spotify, paste passwords, or call unofficial APIs.
- If Buddy is offline: “I couldn't reach your Mac — is Kestrel Buddy open?”
