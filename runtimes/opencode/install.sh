#!/usr/bin/env bash
# runtimes/opencode/install.sh — installer for the OpenCode agentic backend.
#
# Published to the CDN at ${RUNTIMES_BASE_URL}/opencode/install.sh and fetched by
# /usr/local/bin/switch-runtime the first time a device switches to opencode. It
# is self-contained: nothing in the imager or os-server knows about opencode.
#
# This installer is self-sufficient: a direct `bash install.sh` fully configures
# AND starts the backend — it does not rely on switch-runtime to run the presync
# hook or enable the unit afterwards.
#
# What it does:
#   1. prerequisites: jq (presync config reads) + curl + tar (CLI download);
#   2. install the opencode CLI via the official pinned installer to
#      /usr/local/bin/opencode;
#   3. run the presync hook (materialized by os-server BEFORE this installer):
#      it OWNS the opencode config (~/.config/opencode/opencode.json — provider
#      wiring from config.json llm_*), the launch env (/root/.opencode/.env), and
#      the one-time openclaw persona migration. See presync.sh.
#   4. write + start the systemd unit. opencode runs per-turn via `opencode run`, so
#      the unit runs the Go gatewayd (`os-server opencode-gatewayd` — compiled into
#      the os-server binary, nothing to materialize), which drives the opencode
#      process and exposes the WebSocket os-server connects to.
#
# UNIT NAME: opencode.service (== runtime name). No service-name declaration
#    file is needed (switch_runtime.sh defaults the unit to the runtime name).
#
# ⚠️ VERIFY ON DEVICE: the gatewayd must listen on 127.0.0.1:18793 to match
#    runtimes/opencode/constants.go, and its bearer token must equal constants.go
#    Token (both compiled into the os-server binary).
set -euo pipefail

# Tee all output to a log under /root/.opencode (persistent rootfs), NOT
# /var/log — on these boards /var/log is a volatile zram mount wiped on reboot.
OPENCODE_LOG="${OPENCODE_LOG:-/root/.opencode/install.log}"
mkdir -p "$(dirname "$OPENCODE_LOG")"
exec > >(tee -a "$OPENCODE_LOG") 2>&1
echo "[install-opencode] ===== install start $(date -u '+%Y-%m-%dT%H:%M:%SZ') (log: $OPENCODE_LOG) ====="

export HOME=/root
OPENCODE_BIN="/usr/local/bin/opencode"

# Pin the release the device installs. Bump here (and re-OTA os-server) to upgrade.
# Latest published tag as of this build; see
# https://github.com/anomalyco/opencode/releases for newer.
OPENCODE_VERSION="${OPENCODE_VERSION:-1.18.4}"

echo "[install-opencode] prerequisites (jq, curl, tar)"
apt-get update || true
apt-get install -y jq curl tar || true

# Install the opencode CLI via the official installer, pinned to OPENCODE_VERSION
# and forced into /usr/local/bin. The official script handles arch detection
# (linux arm64/x64), the .tar.gz asset name + extraction, and is idempotent (it
# exits early when the pinned version is already installed).
#
# NOTE: OPENCODE_INSTALL_DIR MUST prefix `bash` (the process that runs the
# installer), NOT `curl` — in a pipeline `VAR=x curl … | bash`, VAR binds only to
# curl, so the installer would fall back to its ~/.opencode/bin default.
WANT_VERSION="${OPENCODE_VERSION#v}"
if [ -x "$OPENCODE_BIN" ] && "$OPENCODE_BIN" --version 2>/dev/null | grep -qw "$WANT_VERSION"; then
  echo "[install-opencode] opencode ${WANT_VERSION} already installed at $OPENCODE_BIN — skipping download"
else
  echo "[install-opencode] install opencode CLI ${WANT_VERSION} → /usr/local/bin"
  curl -fsSL https://opencode.ai/install \
    | OPENCODE_INSTALL_DIR=/usr/local/bin bash -s -- --version "$WANT_VERSION"
fi
# Belt-and-suspenders: if the installer ignored OPENCODE_INSTALL_DIR (its
# behavior has drifted before), locate the binary it produced and copy it into
# /usr/local/bin so the systemd unit + verify hook find it at a stable path.
if [ ! -x "$OPENCODE_BIN" ]; then
  echo "[install-opencode] $OPENCODE_BIN missing — locating installer output"
  SRC="$(command -v opencode 2>/dev/null || true)"
  [ -x "$SRC" ] || SRC="/root/.opencode/bin/opencode"
  if [ -x "$SRC" ]; then
    install -m 0755 "$SRC" "$OPENCODE_BIN"
    echo "[install-opencode] copied $SRC → $OPENCODE_BIN"
  fi
fi
"$OPENCODE_BIN" --version || {
  echo "[install-opencode] ERROR: opencode binary not runnable after install" >&2
  exit 1
}

mkdir -p /root/.opencode/workspace /root/.opencode/attachments /root/.config/opencode

# opencode.json + env + persona migration are owned ENTIRELY by the presync
# hook, NOT written here. The hook is materialized to
# /usr/local/bin/runtime-opencode-presync by os-server BEFORE this installer runs;
# switch-runtime re-runs it before every later start, and EnsureOnboarding
# re-runs it on every os-server boot — so the config self-heals (e.g. after a
# factory reset).
PRESYNC_HOOK="/usr/local/bin/runtime-opencode-presync"
if [ -x "$PRESYNC_HOOK" ]; then
  echo "[install-opencode] sync config/env now (via $PRESYNC_HOOK)"
  "$PRESYNC_HOOK" \
    || echo "[install-opencode] WARN: presync failed (config.json missing? non-fatal — retried on next switch/boot)"
else
  echo "[install-opencode] WARN: $PRESYNC_HOOK absent — os-server did not materialize it (standalone/offline run?); config/env NOT synced"
fi

# systemd unit — the gatewayd drives `opencode run` per turn so switch-runtime
# can enable/disable/verify it like any other backend. Unit name == runtime
# name (opencode.service), so no service-name declaration file is needed.
# MUST stay in sync with runtimes/opencode/gateway_unit.go opencodeUnitContent
# (EnsureOnboarding self-heals the same unit on a hand-switched device).
echo "[install-opencode] write systemd unit opencode.service"
cat >/etc/systemd/system/opencode.service <<UNIT
[Unit]
Description=OpenCode agent gateway (os-server opencode-gatewayd driving \`opencode run\` per turn)
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=root
Environment=HOME=/root
EnvironmentFile=-/root/.opencode/.env
WorkingDirectory=/root/.opencode
ExecStart=/usr/local/bin/os-server opencode-gatewayd
Restart=always
RestartSec=3

[Install]
WantedBy=multi-user.target
UNIT
systemctl daemon-reload
echo "[install-opencode] enable + start opencode.service"
systemctl enable --now opencode.service
systemctl status opencode.service --no-pager || true

# Verify hook: cheap + offline (CLI presence + os-server binary — the gatewayd
# ships inside os-server, and presync heals everything else, so a deeper
# structure check here would only force needless full reinstalls).
echo "[install-opencode] declare verify hook for switch-runtime (opencode + os-server)"
mkdir -p /usr/local/lib/os-runtimes/opencode
cat >/usr/local/lib/os-runtimes/opencode/verify <<'VERIFY'
#!/usr/bin/env bash
command -v opencode >/dev/null 2>&1 && [ -x /usr/local/bin/os-server ]
VERIFY
chmod +x /usr/local/lib/os-runtimes/opencode/verify

echo "[install-opencode] done — opencode gatewayd installed + started (opencode.service)."
echo "[install-opencode] ===== install finished $(date -u '+%Y-%m-%dT%H:%M:%SZ') ====="
