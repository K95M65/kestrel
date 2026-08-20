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
