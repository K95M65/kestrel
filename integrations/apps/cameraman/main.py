#!/usr/bin/env python3
"""Cameraman: track a face (fallback: person) until the plugin is stopped."""

from __future__ import annotations

import os
import signal
import sys
import time

import requests

HAL = os.environ.get("HAL_URL", "http://127.0.0.1:5001")


def post(path: str, body: dict | None = None) -> requests.Response | None:
    try:
        r = requests.post(f"{HAL}{path}", json=body or {}, timeout=30)
        r.raise_for_status()
        return r
    except Exception as exc:
        print(f"{path}: {exc}", flush=True)
        return None


def stop(*_args: object) -> None:
    post("/servo/track/stop")
    post("/emotion", {"emotion": "idle", "intensity": 0.4})
    sys.exit(0)


def main() -> None:
    signal.signal(signal.SIGTERM, stop)
    signal.signal(signal.SIGINT, stop)
    post("/emotion", {"emotion": "curious", "intensity": 0.7})
    post("/voice/speak", {"text": "I'll keep you in the shot."})
    started = post("/servo/track", {"target": "face"})
    if started is None or started.status_code >= 400:
        post("/servo/track", {"target": "person"})
    while True:
        time.sleep(2)


if __name__ == "__main__":
    main()
