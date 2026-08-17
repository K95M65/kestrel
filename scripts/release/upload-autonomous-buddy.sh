#!/usr/bin/env bash
set -e

source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/ota-config.sh"
source "${RELEASE_DIR}/ota-metadata.sh"

BUDDY_DIR="${ROOT_DIR}/integrations/companions/autonomous-buddy"
VERSION_FILE="${BUDDY_DIR}/VERSION_AUTONOMOUS_BUDDY"
DIST_DIR="${BUDDY_DIR}/dist"

# Bucket and path: ${BUCKET_PREFIX}/ota/autonomous-buddy/[semver].dmg

# Build target — `dmg` (unsigned, default) or `dmg-signed` (Developer ID + notarized).
# Override via env: BUDDY_DMG_TARGET=dmg-signed scripts/release/upload-autonomous-buddy.sh
DMG_TARGET="${BUDDY_DMG_TARGET:-dmg}"

# Auto-increment semver (patch) before build
if [[ -f "$VERSION_FILE" ]]; then
  current_version=$(tr -d '[:space:]' < "$VERSION_FILE")
  IFS='.' read -r major minor patch <<< "$current_version"
  patch=$((patch + 1))
  new_version="${major}.${minor}.${patch}"
  echo "$new_version" > "$VERSION_FILE"
  echo "========== Version bumped: ${current_version} -> ${new_version} =========="
else
  echo "1.0.0" > "$VERSION_FILE"
  new_version="1.0.0"
  echo "========== Version initialized: ${new_version} =========="
fi

DMG_NAME="AutonomousBuddy-${new_version}.dmg"
DMG_PATH="${DIST_DIR}/${DMG_NAME}"
GCS_PATH="${GCS_PATH:-${BUCKET_PREFIX}/ota/autonomous-buddy/${new_version}.dmg}"

echo "========== Building DMG via 'make ${DMG_TARGET}' (VERSION=${new_version}) =========="
(cd "$BUDDY_DIR" && make "$DMG_TARGET")

if [[ ! -f "$DMG_PATH" ]]; then
  echo "Error: expected DMG not found at $DMG_PATH"
  exit 1
fi

echo "========== Upload ${DMG_NAME} to Google Cloud Storage (no-cache) =========="
gsutil -h "Cache-Control:no-cache, no-store, must-revalidate" \
       -h "Content-Type:application/x-apple-diskimage" \
       cp "$DMG_PATH" "gs://${GCS_BUCKET}/${GCS_PATH}"

# Update metadata.json (${BUCKET_PREFIX}/ota/metadata.json) - autonomous-buddy key
METADATA_PATH="${BUCKET_PREFIX}/ota/metadata.json"
METADATA_TMP=$(mktemp)
PAYLOAD_TMP=$(mktemp)
BUDDY_URL="${BUDDY_URL:-https://storage.googleapis.com/${GCS_BUCKET}/${GCS_PATH}}"

echo "========== Fetch metadata from gs://${GCS_BUCKET}/${METADATA_PATH} =========="
if gsutil cp "gs://${GCS_BUCKET}/${METADATA_PATH}" "$METADATA_TMP" 2>/dev/null; then
  ota_metadata_unpack "$METADATA_TMP" "$PAYLOAD_TMP"
else
  printf '{}' >"$PAYLOAD_TMP"
fi

updated_metadata=$(python3 - "$PAYLOAD_TMP" "$new_version" "$BUDDY_URL" "$(date '+%Y-%m-%d %H:%M:%S %z')" <<'PY'
import json, sys
data = json.load(open(sys.argv[1]))
data.pop('claude-desktop-buddy', None)
data['autonomous-buddy'] = {'version': sys.argv[2], 'url': sys.argv[3], 'updated_at': sys.argv[4]}
print(json.dumps(data, indent=2))
PY
)

echo "$updated_metadata" > "$PAYLOAD_TMP"
ota_metadata_sign "$PAYLOAD_TMP" "$METADATA_TMP"
echo "========== Upload metadata (autonomous-buddy: v${new_version}) =========="
gsutil -h "Content-Type:application/json" -h "Cache-Control:no-cache, no-store, must-revalidate" cp "$METADATA_TMP" "gs://${GCS_BUCKET}/${METADATA_PATH}"
rm -f "$METADATA_TMP" "$PAYLOAD_TMP"

echo "Done: gs://${GCS_BUCKET}/${GCS_PATH} (v${new_version})"
echo "URL:  ${BUDDY_URL}"
