#!/usr/bin/env bash
# Shared signed OTA metadata helpers. Source from release scripts; do not run
# this file directly. The signing private key must remain outside the repo.

ota_base64_decode() {
  base64 --decode 2>/dev/null || base64 -D
}

# ota_metadata_unpack <envelope> <payload>
# Existing unsigned feeds cannot be safely extended. Operators must create the
# first signed metadata document with ota_metadata_sign over an explicit {}.
ota_metadata_unpack() {
  local envelope="$1" payload="$2"
  jq -er '.format == "autonomous-ota/v1" and .signature.algorithm == "ed25519" and (.payload | type == "string")' "$envelope" >/dev/null \
    || { echo "ERROR: existing OTA metadata is not a signed autonomous-ota/v1 envelope" >&2; return 1; }
  jq -r '.payload' "$envelope" | ota_base64_decode >"$payload" \
    || { echo "ERROR: decode signed OTA metadata payload" >&2; return 1; }
  jq -e . "$payload" >/dev/null || { echo "ERROR: OTA metadata payload is not JSON" >&2; return 1; }
}

# ota_metadata_sign <payload> <envelope>
# OTA_SIGNING_PRIVATE_KEY is a PEM Ed25519 key path; OTA_SIGNING_KEY_ID labels
# the key for operators and is not used as a trust decision on devices.
ota_metadata_sign() {
  local payload="$1" envelope="$2" signature payload_b64 signature_b64
  : "${OTA_SIGNING_PRIVATE_KEY:?OTA_SIGNING_PRIVATE_KEY (Ed25519 PEM path) is required}"
  : "${OTA_SIGNING_KEY_ID:?OTA_SIGNING_KEY_ID is required}"
  [ -r "$OTA_SIGNING_PRIVATE_KEY" ] || { echo "ERROR: cannot read OTA_SIGNING_PRIVATE_KEY" >&2; return 1; }

  signature=$(mktemp)
  trap 'rm -f "$signature"' RETURN
  openssl pkeyutl -sign -rawin -inkey "$OTA_SIGNING_PRIVATE_KEY" -in "$payload" -out "$signature" \
    || { echo "ERROR: sign OTA metadata" >&2; return 1; }
  payload_b64=$(base64 <"$payload" | tr -d '\n')
  signature_b64=$(base64 <"$signature" | tr -d '\n')
  # Keep the payload's component entries at the top level for already deployed
  # workers. New workers use only .signed after verifying it.
  jq --arg payload "$payload_b64" --arg signature "$signature_b64" --arg keyID "$OTA_SIGNING_KEY_ID" \
    '. + {signed:{format:"autonomous-ota/v1", payload:$payload, signature:{algorithm:"ed25519", key_id:$keyID, value:$signature}}}' \
    "$payload" >"$envelope"
}

# ota_artifact_sha256 <file>
ota_artifact_sha256() {
  shasum -a 256 "$1" | awk '{print $1}'
}
