# Kestrel cycle log

Sequencer for the implement → test → push → “What would I do next?” loop.
New ideas may be appended only as a direct follow-on of the last next-step.

## Seed ideas

1. **In-flight slice** — Talk leak strip, Device Voice on-robot / in-browser, in-app Guide + `docs/wiki/`, kestrel-host setup copy, Uses “Needs a computer”.
2. **Find this robot** — Home card: LAN URL, mDNS host, copy, QR. Prefer IP; `.local` is extra.
3. **Guide: brains** — wiki page on switching OpenClaw / Hermes / others; coding CLIs are not the companion loop.
4. **Runtime switch copy** — Advanced Runtime warns Codex / Claude Code / OpenCode are coding bridges.
5. **Speaker mute without music** — poll speaker mute whenever audio is present, not only when music is mounted.
6. **Wi-Fi now** — Device → Wi-Fi shows the connected SSID.
7. **Guide: identity** — wiki page for name, soul, people, memory.
8. **Channels lead** — Device → Channels explains Telegram is how you text this robot, then credentials.

Deferred on purpose (not this loop): iOS/Android app, flash Reachy, six-runtime hop on the live desk, Matter/Zigbee/OT, ScoutHQ.

## Decisions

### Cycle 1 — 2ffd9fff
Done: Talk strip, Voice on-robot/in-browser, Guide + docs/wiki, kestrel-host copy, Uses “Needs a computer”.
What would I do next? Find this robot on Home (LAN IP, mDNS, copy, QR).
Why: last product question was finding the body on the network; IP is more reliable than `.local`.

### Cycle 2
Done: Home **This robot** card from `deviceReach` (LAN first, mDNS extra, loopback never shared). Guide page `find`.
What would I do next? Guide page on brains / runtime switch honesty (seed 3).
Why: the previous conversation asked if OpenClaw/Hermes/etc. are easy to swap; Home users should read that before Advanced Runtime.

### Cycle 3
Done: Guide **Brains** page — companion loop vs coding CLIs, what survives a switch.
What would I do next? Advanced Runtime warns Codex / Claude Code / OpenCode are coding bridges (seed 4).
Why: the wiki now says it; the switcher itself still looks like a harmless dropdown.

### Cycle 4
Done: Runtime dropdown shows a coding-CLI / PicoClaw warning before Switch.
What would I do next? Poll speaker mute whenever audio is present, not only when music is mounted (seed 5).
Why: kestrel-host may have a speaker without a music route; Home MUTED was already a desk bug.

### Cycle 5
Done: `/voice/status` reports `speaker_muted`; Home reads it even without music.
What would I do next? Device → Wi-Fi shows the connected SSID (seed 6).
Why: product queue P1 Device Wi-Fi; next small home-user honesty.

### Cycle 6
Done: Device → Wi-Fi shows live station SSID, signal, and link rate.
What would I do next? Guide page for identity layers (seed 7).
Why: the earlier identity question (name, soul, people) still has no handbook page.

### Cycle 7
Done: Guide **Name and memory** — IDENTITY, soul, USER, MEMORY, People.
What would I do next? Channels lead copy before tokens (seed 8).
Why: last remaining seed item; Device → Channels still opened on bot tokens.

### Cycle 8
Done: Channels card lead: text this robot from your phone, not iMessage.
What would I do next? Stop. Seed list is covered. Remaining product (storage prune, forget Wi-Fi, six-runtime hop) stays deferred as in Non-goals / seed deferrals.

### Cycle 9
Done: Sign in with Google (device-code OAuth), claim QR + household roles, ChatGPT-style ask levels, `@skill` in Talk, trusted plugin list. Apple Sign in, Matter/HomeKit, and a phone app stay deferred.
What would I do next? Not in this loop unless asked.

### Cycle 10
Done: sideload 0.1.31 / web 0.1.47 onto Lima and Bobert; OpenClaw on the VM with Grok from the desk unit; LAN hive (nginx `/api/buzz/ws`, restart on save, hive skill); Matter commission via Home Assistant; Sign in with Apple web flow (HTTPS return URL); Block Buzz compose script (not on the Pi — different wire).
What would I do next? Home-user onboarding cards for Google, hive, Matter, claim (Apple/Google add-device analog).

### Cycle 11
Done: onboarding cards — Google (TV sign-in, client behind a disclosure), hive (join QR like a pairing code), Matter (House, three steps), claim/Home copy as Add Accessory. Apple developer fields Advanced. Guide: compared with Siri AI and ChatGPT plugins. Ask levels named as ChatGPT’s. Join URL on hive host.
What would I do next? Paste a Google TV OAuth client; HA URL+token if a Matter accessory is on the desk.
