#!/usr/bin/env bash
# spike-lib.sh — shared plumbing for the Reachy Mini spike scripts.
#
# Sourced, never executed. Everything the spike scripts have in common lives
# here exactly once: fetching OTA metadata, installing a component from it, and
# writing a systemd unit.
#
# Why a library at all: the first version of spike.sh duplicated what the
# per-component scripts did, drifted from them within a week, and ended up
# deploying os-server against the wrong config directory while every service
# still reported healthy. Nothing below is copied into a caller — a caller that
# needs different behaviour changes it here, for all of them.
#
# Everything is pulled from the OTA metadata, the same source the imager and
# scripts/provision/setup.sh read. Nothing is built on a developer machine and
# copied over: that would install whatever happened to be in someone's working
# tree, which is not what the fleet runs and not what a bug report is about.

# --- Layout ------------------------------------------------------------------
# The production layout, not a spike-only one. An earlier version installed
# under /opt/autonomous and had to patch DEVICES_DIR in the .env to match; the
# shipped .env already says /opt/devices, so using the real paths removes a
# whole class of "healthy but empty capability list" failures.
DEVICE_TYPE="${DEVICE_TYPE:-reachy-mini}"
HAL_DIR="${HAL_DIR:-/opt/hal}"
DEVICES_DIR="${DEVICES_DIR:-/opt/devices}"
CONFIG_DIR="${CONFIG_DIR:-/root/config}"
BIN_DIR="${BIN_DIR:-/usr/local/bin}"
WEB_ROOT="${WEB_ROOT:-/usr/share/nginx/html/setup}"

# The Pollen daemon: owns motion, and owns the camera and ALSA PCMs until HAL
# asks for them (DEVICE.md `owner: pollen_daemon`).
DAEMON_URL="${DAEMON_URL:-http://localhost:8000}"

# Default OTA feed. Overridden by an existing bootstrap.json, or by exporting
# OTA_METADATA_URL before running any spike script.
DEFAULT_METADATA_URL="https://cdn.autonomous.ai/os/ota/metadata.json"

# All progress output goes to STDERR, never stdout. ota_field and friends are
# called inside $( ), so anything they print on stdout is captured as part of
# the value — which silently turned a version string into an apt transcript.
say() { printf '\n========== %s ==========\n' "$1" >&2; }
info() { printf '[%s] %s\n' "${SPIKE_TAG:-spike}" "$1" >&2; }
die() { printf '[%s] ERROR: %s\n' "${SPIKE_TAG:-spike}" "$1" >&2; exit 1; }

ensure_root() {
  [ "$(id -u)" -eq 0 ] || die "run as root:  sudo bash $0 $*"
}

retry() {
  local cmd="$1" max="${2:-5}" delay="${3:-2}" n=0
  until [ "$n" -ge "$max" ]; do
    eval "$cmd" && return 0
    n=$((n + 1))
    info "retry $n/$max"
    sleep "$delay"
  done
  return 1
}

# --- Prerequisites -----------------------------------------------------------

ensure_tools() {
  local missing=()
  for t in curl unzip jq; do command -v "$t" >/dev/null || missing+=("$t"); done
  [ ${#missing[@]} -eq 0 ] && return 0
  info "installing: ${missing[*]}"
  # Output to stderr for the same reason as info(): this can run underneath a
  # command substitution, and an apt transcript on stdout becomes the "value".
  apt-get update -qq >&2
  DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends "${missing[@]}" >&2 \
    || die "could not install ${missing[*]}"
}

# --- OTA metadata ------------------------------------------------------------
# One fetch per script run, cached in a temp file. The URL comes from
# bootstrap.json when it exists, so a robot pointed at a staging feed keeps
# using it instead of silently falling back to production.

_METADATA_FILE=""

metadata_url() {
  if [ -n "${OTA_METADATA_URL:-}" ]; then
    echo "$OTA_METADATA_URL"
    return
  fi
  local from_bootstrap=""
  if [ -f "$CONFIG_DIR/bootstrap.json" ]; then
    from_bootstrap="$(jq -r '.metadata_url // empty' "$CONFIG_DIR/bootstrap.json" 2>/dev/null || true)"
  fi
  echo "${from_bootstrap:-$DEFAULT_METADATA_URL}"
}

fetch_metadata() {
  [ -n "$_METADATA_FILE" ] && [ -f "$_METADATA_FILE" ] && return 0
  ensure_tools
  local url
  url="$(metadata_url)"
  _METADATA_FILE="$(mktemp)"
  # no-cache: the CDN serves these with long-lived edge caching, and a stale
  # metadata read installs yesterday's build while reporting today's version.
  retry "curl -fsSL -H 'Cache-Control: no-cache' -H 'Pragma: no-cache' -o '$_METADATA_FILE' '$url'" 5 \
    || die "could not fetch OTA metadata from $url"
  info "OTA metadata: $url"
}

# ota_field <component> <field>
# Components are top-level keys ("hal", "os-server", "web", "bootstrap",
# "openclaw"); the device profile is nested under devices.<type>.
ota_field() {
  local component="$1" field="$2"
  fetch_metadata
  if [ "$component" = "device" ]; then
    jq -r --arg t "$DEVICE_TYPE" --arg f "$field" '.devices[$t][$f] // empty' "$_METADATA_FILE"
  else
    jq -r --arg c "$component" --arg f "$field" '.[$c][$f] // empty' "$_METADATA_FILE"
  fi
}

# ota_unpack <component> <dest_dir>
# Downloads the component zip and unpacks it INTO dest_dir. Prints the version.
ota_unpack() {
  local component="$1" dest="$2"
  local url version zip_tmp
  url="$(ota_field "$component" url)"
  version="$(ota_field "$component" version)"
  [ -n "$url" ] || die "OTA metadata has no url for '$component' (device_type=$DEVICE_TYPE)"

  zip_tmp="$(mktemp)"
  retry "curl -fsSL -H 'Cache-Control: no-cache' -H 'Pragma: no-cache' -o '$zip_tmp' '$url'" 5 \
    || die "download failed: $component from $url"
  mkdir -p "$dest"
  unzip -o -q "$zip_tmp" -d "$dest" || die "unzip failed: $component"
  rm -f "$zip_tmp"
  info "$component $version -> $dest"
}

# ota_install_binary <component> <dest_path>
# For components shipped as a zipped single binary (os-server, bootstrap).
ota_install_binary() {
  local component="$1" dest="$2"
  local dir_tmp bin
  dir_tmp="$(mktemp -d)"
  ota_unpack "$component" "$dir_tmp"
  # The zip has held the binary at the root and one level down at different
  # times; prefer an executable, fall back to the only file present.
  bin="$(find "$dir_tmp" -type f -perm -u+x 2>/dev/null | head -1)"
  [ -n "$bin" ] || bin="$(find "$dir_tmp" -type f 2>/dev/null | head -1)"
  [ -n "$bin" ] || die "no binary inside the $component zip"
  # Replacing a running binary in place gives ETXTBSY — the caller stops the
  # unit first, but a stray process can still hold it, so install by rename.
  install -m 0755 "$bin" "$dest.new" && mv -f "$dest.new" "$dest"
  rm -rf "$dir_tmp"
}

# --- systemd -----------------------------------------------------------------

# write_unit <name> <<'UNIT' ... UNIT
# Reads the unit body from stdin so callers keep it as a readable heredoc.
write_unit() {
  local name="$1"
  cat >"/etc/systemd/system/${name}.service"
  systemctl daemon-reload
  info "wrote /etc/systemd/system/${name}.service"
}

start_unit() {
  local name="$1"
  systemctl enable "$name" >/dev/null 2>&1
  systemctl restart "$name"
}

stop_unit() {
  local name="$1"
  systemctl stop "$name" 2>/dev/null || true
}

remove_unit() {
  local name="$1"
  systemctl disable --now "$name" 2>/dev/null || true
  rm -f "/etc/systemd/system/${name}.service"
  systemctl daemon-reload
}

# wait_http <url> <seconds> [label]
wait_http() {
  local url="$1" secs="${2:-60}" label="${3:-service}" i=0
  printf '[%s] waiting for %s ' "${SPIKE_TAG:-spike}" "$label"
  while [ "$i" -lt "$secs" ]; do
    if curl -sf -m 3 "$url" >/dev/null 2>&1; then echo " up"; return 0; fi
    printf '.'
    sleep 2
    i=$((i + 2))
  done
  echo " TIMEOUT"
  return 1
}
