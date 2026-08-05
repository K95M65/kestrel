#!/usr/bin/env bash
# `gateway status --require-rpc` completes OpenClaw's authenticated gateway
# probe, including the device-specific connect handshake.
set -euo pipefail

HOME=/root exec /usr/bin/openclaw gateway status --require-rpc --timeout 5000
