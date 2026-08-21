#!/usr/bin/env bash
# Optional Block Buzz Nostr relay (Postgres + Redis + MinIO + ghcr.io/block/buzz).
# Not the Kestrel LAN hive. Do not run this on the Reachy (disk). Needs Docker
# Compose on a host with several GB free. See docs/wiki/hive.md.
set -euo pipefail
if ! command -v docker >/dev/null 2>&1; then
  echo "Docker is required. Start Docker Desktop, or install docker on a VM — not on Bobert." >&2
  exit 1
fi
if ! docker info >/dev/null 2>&1; then
  echo "Docker is installed but not running." >&2
  exit 1
fi
ROOT="${BUZZ_DIR:-$HOME/buzz-relay}"
mkdir -p "$ROOT"
if [[ ! -d "$ROOT/buzz/.git" ]]; then
  git clone --depth 1 https://github.com/block/buzz.git "$ROOT/buzz"
fi
cd "$ROOT/buzz/deploy/compose"
if [[ ! -f .env ]]; then
  cp .env.example .env
  python3 - <<'PY'
import os, secrets, pathlib
p = pathlib.Path(".env")
text = p.read_text()
def hx():
    return secrets.token_hex(32)
repl = {
    "CHANGE_ME_OWNER_PUBKEY_HEX": hx(),
    "CHANGE_ME_64_HEX_PRIVATE_KEY": hx(),
    "CHANGE_ME_RANDOM_64_HEX": hx(),
    "CHANGE_ME_RANDOM_PASSWORD": secrets.token_urlsafe(18),
    "CHANGE_ME_RANDOM_ACCESS_KEY": secrets.token_urlsafe(12),
    "CHANGE_ME_RANDOM_SECRET_KEY": secrets.token_urlsafe(18),
}
for a, b in repl.items():
    text = text.replace(a, b, 1)
p.write_text(text)
print("Wrote deploy/compose/.env with generated secrets. Edit RELAY_URL if this is not localhost.")
PY
fi
echo "Starting Block Buzz compose in $PWD"
if [[ -x ./run.sh ]]; then
  ./run.sh start
else
  docker compose up -d
fi
echo
echo "Relay (typical): ws://127.0.0.1:3000"
echo "Point Buzz desktop / buzz-cli here. Do not paste this URL into Kestrel Hive — different wire."
echo "Kestrel hive stays Device → Channels → Hive (JSON websocket on the robot)."
