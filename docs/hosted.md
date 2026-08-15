# What we host, what stays on your network, what happens offline

Autonomous OS phones home to three things and nothing else.

- **Skill store** — the catalog behind one-tap install. Plugins are not on this
  list: the plugin store on every robot is the Hugging Face Hub — push a Space,
  tag it `autonomous-os-plugin`, it shows up under Settings → Plugins → Browse.
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
