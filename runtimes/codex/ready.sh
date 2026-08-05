#!/usr/bin/env bash
# The bridge is usable only after it accepts an authenticated WebSocket upgrade.
set -euo pipefail

headers="$(curl --http1.1 --silent --max-time 2 -D - -o /dev/null \
  -H 'Connection: Upgrade' \
  -H 'Upgrade: websocket' \
  -H 'Sec-WebSocket-Version: 13' \
  -H 'Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==' \
  -H 'Authorization: Bearer autonomous_codex_token' \
  http://127.0.0.1:18792/codex/ws/ 2>/dev/null || true)"
grep -q '^HTTP/1.1 101 ' <<<"$headers"
