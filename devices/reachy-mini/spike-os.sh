#!/usr/bin/env bash
# spike-os.sh — Deploy and run ONLY os-server on a Reachy Mini.
#
# Runs FROM YOUR MAC. Cross-compiles the Go binary, rsyncs it, seeds a minimal
# config, and runs it in tmux. Companion to spike-hal.sh (body) — this is the
# API/agent layer. Neither is production: no systemd, no nginx, no OTA.
#
# What this does NOT give you: the web UI. os-server binds 127.0.0.1:5000 and
# serves no static files — nginx is what serves the dist and proxies /api
# (scripts/provision/setup.sh). Until that exists for Reachy, reach the API from
# the Pi itself, or tunnel it:
#
#   ssh -L 5000:localhost:5000 pollen@reachy-mini.local
#   # then on the Mac: curl localhost:5000/api/health/live
#
# Why root + cwd=/root: config.Load reads the RELATIVE path "config/config.json"
# (system/server/config/config.go), so production's WorkingDirectory=/root makes
# it /root/config/config.json — the same file HAL looks for via OS_CONFIG_PATH.
# Running from anywhere else silently splits the two services onto different
# configs. The logger also writes /var/log/os-server.log, which needs root.
#
# Usage:
#   bash devices/reachy-mini/spike-os.sh              # build + deploy + run
#   bash devices/reachy-mini/spike-os.sh --no-build   # redeploy the existing binary
#   bash devices/reachy-mini/spike-os.sh --stop       # stop it
set -euo pipefail

REACHY_HOST="${REACHY_HOST:-pollen@reachy-mini.local}"
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
REMOTE_BASE="/opt/autonomous"
SESSION="os"

SKIP_BUILD=0
STOP_ONLY=0
for arg in "$@"; do
  case "$arg" in
    --no-build) SKIP_BUILD=1 ;;
    --stop)     STOP_ONLY=1 ;;
    *) echo "unknown flag: $arg" >&2; exit 2 ;;
  esac
done

say() { printf '\n========== %s ==========\n' "$1"; }

if [ "$STOP_ONLY" = "1" ]; then
  say "Stopping os-server on $REACHY_HOST"
  ssh "$REACHY_HOST" "tmux kill-session -t $SESSION 2>/dev/null || true; sudo pkill -f '$REMOTE_BASE/os-server' 2>/dev/null || true; echo stopped"
  exit 0
fi

echo "target: $REACHY_HOST"

if [ "$SKIP_BUILD" = "0" ]; then
  say "1/4  Build os-server (linux/arm64)"
  cd "$ROOT_DIR"
  make os-build
else
  say "1/4  Build skipped (--no-build)"
  [ -f "$ROOT_DIR/system/os-server" ] || { echo "no binary at system/os-server — drop --no-build"; exit 1; }
fi

say "2/4  Copy binary"
ssh "$REACHY_HOST" "sudo mkdir -p $REMOTE_BASE && sudo chown \$(id -u):\$(id -g) $REMOTE_BASE"
# Replacing a running binary in place gives ETXTBSY; stop first, then copy.
ssh "$REACHY_HOST" "tmux kill-session -t $SESSION 2>/dev/null || true; sudo pkill -f '$REMOTE_BASE/os-server' 2>/dev/null || true; sleep 1" || true
scp "$ROOT_DIR/system/os-server" "$REACHY_HOST:$REMOTE_BASE/os-server"
ssh "$REACHY_HOST" "chmod +x $REMOTE_BASE/os-server"

say "3/4  Seed /root/config/config.json"
# Minimal config: the only fail-loud startup guard is device_type
# (server.go: "device_type unresolved ... refusing to assume 'lamp'"). Everything
# else defaults, and the web setup flow fills the rest in. Never overwrite an
# existing config — it holds provisioning state.
ssh "$REACHY_HOST" bash <<'REMOTE_CONF'
set -e
if sudo test -f /root/config/config.json; then
  echo "[spike-os] /root/config/config.json already present — keeping it"
else
  sudo mkdir -p /root/config
  sudo tee /root/config/config.json >/dev/null <<'JSON'
{
  "httpPort": 5000,
  "device_type": "reachy-mini"
}
JSON
  echo "[spike-os] seeded /root/config/config.json"
fi
REMOTE_CONF

say "4/4  Start os-server in tmux"
ssh "$REACHY_HOST" bash <<REMOTE_START
set -e
tmux kill-session -t $SESSION 2>/dev/null || true
tmux new-session -d -s $SESSION -n os

# cwd=/root so the relative config path resolves the same way it does under
# systemd; DEVICE_TYPE also passed explicitly so a missing config still boots.
# The cd must happen INSIDE sudo — /root is 0700, so the login shell cannot
# enter it before elevating.
# DEVICES_DIR must be passed too: device.DevicesDir() falls back to /opt/devices
# (devicemd.go), which the spike layout does not create — os-server then finds no
# DEVICE.md and reports an empty capability list, so the web Overview renders
# blank tiles while every service looks healthy.
tmux send-keys -t $SESSION:os.0 \
  "sudo sh -c 'cd /root && DEVICE_TYPE=reachy-mini DEVICES_DIR=$REMOTE_BASE/devices exec $REMOTE_BASE/os-server' 2>&1 | tee /tmp/os-server.log" Enter

tmux split-window -t $SESSION:os -v
tmux send-keys -t $SESSION:os.1 \
  "echo 'waiting for os-server...'; for i in \\\$(seq 1 30); do curl -sf localhost:5000/api/health/live >/dev/null && break; sleep 2; done; echo '--- live ---'; curl -s localhost:5000/api/health/live; echo; echo '--- readiness ---'; curl -s localhost:5000/api/health/readiness; echo" Enter
REMOTE_START

cat <<EOF

========================================
  os-server started on $REACHY_HOST
    logs   : ssh $REACHY_HOST 'tmux attach -t $SESSION'   (or tail /tmp/os-server.log)
    API    : loopback only — from the Pi:  curl localhost:5000/api/health/live
    tunnel : ssh -L 5000:localhost:5000 $REACHY_HOST
    stop   : bash devices/reachy-mini/spike-os.sh --stop
========================================

Footprint outside $REMOTE_BASE: /root/config/config.json and /var/log/os-server.log.
Web UI still needs nginx — see spike-os.sh header.
EOF
