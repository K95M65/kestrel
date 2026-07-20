# Chat bridges

Bridges that turn external chat messages into device **sensing events**: each one receives
messages from a chat platform and POSTs them to the device's `/api/sensing/event`, where they
enter the same pipeline as voice or camera input (intent match first, then an agent turn).

| Bridge | Source | Transport in |
|--------|--------|--------------|
| [`twitch-chat-hook/`](twitch-chat-hook/) | Twitch live chat | EventSub webhook (HTTPS) or IRC fallback |
| [`autonomous-chat-hook/`](autonomous-chat-hook/) | Autonomous web chat | MQTT subscribe |

Both are self-contained Go modules; build targets live in the top-level `Makefile`
(`upload-twitch-irc`, `upload-autonomous-chat`).
