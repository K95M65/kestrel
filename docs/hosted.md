# What we host, what stays on your network, what happens offline

Autonomous OS phones home to three things and nothing else.

- **Skill Store** — our own catalog, behind one-tap install in the app. It
  carries more than this repo does: 70 skills as of 2026-08-16, of which the 25 in `skills/` are
  the robot skills that ship with the OS. The other 45 are first-party and
  user-published *workflow* skills (code review, standups, campaign plans) that
  need no hardware; they live only in the store, and between them they have
  been installed 14 times. Nothing in the store is paid — every entry is free. Plugins are not on this
  list: plugins install from the same store. (The plugin browser still queries
  Hugging Face Spaces — that was a prototype, and removing it is
  [#213](https://github.com/autonomous-ai/autonomous-os/issues/213).)
- **AI gateway** — the default OpenClaw brain and the voice, face and mood
  models call it. The key comes with the app account, self-built bodies
  included. No account? Swap the brain to Claude Code or Codex with your own key:
  you get chat and every skill; voice, face and mood still need the gateway.
  Pointing OpenClaw at your own endpoint (Ollama, any OpenAI-compatible server on
  your LAN) is not built yet — it is the first item under
  [Not built yet](../README.md#not-built-yet--claim-one).
- **Release feed** — the robot auto-updates from our CDN every 5 min
  (`bootstrap/`, staged by `min_version`, so we can hold a floor). Today: zips
  over HTTPS, no signature or checksum, no rollback. Running a fleet? Point
  `OTA_METADATA_URL` at your own feed and you control what ships; signed
  releases are on the list. There is no fleet view yet — one robot per **Add
  robot**, and every robot pulls the same skill feed and OTA floor.

## Offline

Local intents (~50 ms), recorded moves and the safety gate keep working;
conversation, voice, face and mood stop until the network is back, because the
brain and those models call the gateway. A fully local robot needs the
BYO-endpoint PR.

## On the network

HAL (:5001), the brain and OTA listen on localhost only; nginx exposes setup,
monitor and chat on the LAN behind the 4-character login. Nothing listens on
the internet — [`SECURITY.md`](../SECURITY.md) has the audit.

## Running more than one robot

Until signed OTA, a real `POST /servo/stop` and unprivileged runtimes merge,
keep the fleet on your own `OTA_METADATA_URL` and off the public internet.
