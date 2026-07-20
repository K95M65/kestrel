# Chat bridges

A chat bridge turns messages from an external chat platform into device **sensing events**:
it POSTs each message to the os-server's `/api/sensing/event` (default
`http://127.0.0.1:5000/api/sensing/event`), the same endpoint HAL's voice service uses for
transcripts. From there the message enters the exact same pipeline as voice or camera input —
local intent match first (instant TTS for fixed phrases), then an agent turn with the reply
flowing back through TTS / web chat / channels.

Each bridge tags its messages with a source prefix so SOUL.md can tell platform chat apart
from real microphone input:

| Bridge | Prefix | Sensing type (env override) |
|--------|--------|------------------------------|
| `twitch-chat-hook/` | `[source: twitch, twitch_user: <nick>] <text>` | `voice` (`TWITCH_SENSING_TYPE`) |
| `autonomous-chat-hook/` | `[source: autonomous_web, user: <name>] <text>` | `voice` (`AUTONOMOUS_SENSING_TYPE`) |

Both bridges are self-contained Go modules with their own `go.mod`, `.env.example`,
`VERSION_*` file, and OTA upload script. Forwarding is fire-and-forget (background goroutine,
2s HTTP timeout) so a slow device never blocks the receive loop.

## Flows

**Twitch** — EventSub webhook (production) with an anonymous-IRC fallback:

```
Twitch EventSub ──HTTPS POST──▶ cmd/webhook ──verify HMAC──▶ dedupe by ──▶ handleChatMessage
 (channel.chat.message)          (:8080)      signature       message ID         │
                                                                                 ▼
Twitch IRC gateway ──TLS 6697──▶ cmd/irc (justinfan nick,    twitch.ForwardChatMessage
 (fallback, read-only)            no app/token/2FA)  ────────────────┘           │
                                                                                 ▼
                                                        POST /api/sensing/event on device
```

**Autonomous web chat** — MQTT relay, runs on the device:

```
web user ──HTTP──▶ Autonomous BE ──MQTT publish──▶ broker ──▶ cmd/mqtt (autopaho, QoS 1)
                    {"user":..., "text":...}                        │
                                                                    ▼
                                              POST http://127.0.0.1:5000/api/sensing/event
```

## twitch-chat-hook

- `cmd/webhook/` — HTTPS EventSub receiver. Verifies `sha256=HMAC(secret, id||timestamp||body)`
  on the raw body, answers the challenge handshake, dedupes redeliveries by
  `Twitch-Eventsub-Message-Id` (in-memory, 10-min TTL), always ACKs 2xx fast.
  Env: `TWITCH_WEBHOOK_SECRET` (required), `PORT` (default 8080). Runs BE-side —
  Twitch requires a public HTTPS callback (terminate TLS in front).
- `cmd/subscribe/` — one-shot CLI that creates the `channel.chat.message` subscription via
  Helix (`-channel`, `-bot`, `-callback` flags; needs `TWITCH_CLIENT_ID`/`SECRET`,
  `TWITCH_BOT_USER_TOKEN` with scope `user:read:chat`).
- `cmd/irc/` — anonymous IRC fallback (`-channel foo,bar`). No app, no token, no HMAC/dedup
  metadata; read-only; can run on the device itself. This is the binary the OTA target ships.
- `twitch/` — EventSub types, HMAC verification, minimal Helix client, and `forward.go`
  (`DEVICE_SENSING_URL` — legacy `LAMP_SENSING_URL` still honored — and `TWITCH_SENSING_TYPE` env overrides).

See `twitch-chat-hook/README.md` and `HANDOFF.md` for full setup, token refresh, and limits.

## autonomous-chat-hook

- `cmd/mqtt/` — autopaho subscriber (auto-reconnect, exponential backoff). Expects BE to
  publish JSON `{"user": "...", "text": "..."}`; empty `user` becomes `anonymous`, empty
  `text` is dropped.
- `autonomous/forward.go` — HTTP POST to the sensing endpoint, same shape as the Twitch one.
- Env: `AUTONOMOUS_MQTT_URL` + `AUTONOMOUS_MQTT_TOPIC` (required),
  `AUTONOMOUS_MQTT_USERNAME`/`PASSWORD`, `AUTONOMOUS_MQTT_CLIENT_ID`, `DEVICE_SENSING_URL` (legacy `LAMP_SENSING_URL` honored),
  `AUTONOMOUS_SENSING_TYPE`.
- Runs **on the device** (defaults target loopback :5000), e.g. under systemd with
  `EnvironmentFile=/opt/autonomous-chat-hook/.env` — see its README for a unit example.

## Build & release

```bash
make twitch-build-irc          # cross-compile cmd/irc → twitch-irc (linux/arm64)
make autonomous-build-chat     # cross-compile cmd/mqtt → autonomous-chat (linux/arm64)

make upload-twitch-irc         # scripts/release/upload-twitch-irc.sh
make upload-autonomous-chat    # scripts/release/upload-autonomous-chat.sh
```

Each upload script: bumps the patch version in `twitch-chat-hook/VERSION_TWITCH_IRC` /
`autonomous-chat-hook/VERSION_AUTONOMOUS_CHAT`, builds with the version injected via
ldflags (`-X main.Version=...`), zips the binary, uploads it to GCS at
`${BUCKET_PREFIX}/ota/<component>/<semver>.zip`, and updates the shared OTA
`metadata.json` under its component key (`twitch-irc` / `autonomous-chat`). Bucket and
prefix come from `scripts/release/ota-config.sh`.

## Writing a new bridge

1. New self-contained Go module under `integrations/chat-bridges/<name>/` with `cmd/`,
   a forward package, `.env.example`, and a `VERSION_<NAME>` file.
2. Deliver every message as `{"type": ..., "message": ...}` to `/api/sensing/event`
   (`DEVICE_SENSING_URL`-style env override, default loopback :5000).
3. Prefix the message with a `[source: <platform>, ...user...]` tag so SOUL.md can route it.
4. Forward fire-and-forget from a goroutine with a short timeout — never block the
   platform receive loop on the device.
5. Dedupe redeliveries if the transport can retry (see the webhook's message-ID dedupe),
   and reconnect automatically if it's a long-lived connection (see autopaho usage).
6. Add a `make <name>-build` target plus `upload-<name>` + `scripts/release/upload-<name>.sh`
   mirroring the existing pair (version bump → zip → GCS → metadata.json key).
