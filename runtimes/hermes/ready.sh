#!/usr/bin/env bash
# Hermes is ready only once its local HTTP gateway accepts an authenticated
# health request. This is intentionally stricter than systemctl is-active:
# systemd can report active while the gateway is still booting and has not bound
# its listener yet.
set -euo pipefail

curl --fail --silent --show-error --max-time 5 \
  -H 'Authorization: Bearer hermes-local-api-key' \
  http://127.0.0.1:8642/health >/dev/null
