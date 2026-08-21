#!/usr/bin/env python3
"""Radio: play a station on the speaker until the plugin is stopped."""

from __future__ import annotations

import os
import signal
import sys
import time

import requests

HAL = os.environ.get("HAL_URL", "http://127.0.0.1:5001")
STATIONS = {
    "lofi": "lofi hip hop radio",
    "jazz": "jazz radio station",
    "news": "bbc world service radio",
    "classical": "classical music radio",
}


def post(path: str, body: dict | None = None) -> None:
    try:
        requests.post(f"{HAL}{path}", json=body or {}, timeout=30).raise_for_status()
    except Exception as exc:
        print(f"{path}: {exc}", flush=True)


def stop(*_args: object) -> None:
    post("/audio/stop")
    post("/emotion", {"emotion": "idle", "intensity": 0.4})
    sys.exit(0)


def query() -> str:
    q = os.environ.get("RADIO_QUERY", "").strip()
    if q:
        return q
    st = os.environ.get("RADIO_STATION", "lofi").strip().lower()
    return STATIONS.get(st, STATIONS["lofi"])


def main() -> None:
    signal.signal(signal.SIGTERM, stop)
    signal.signal(signal.SIGINT, stop)
    q = query()
    post("/emotion", {"emotion": "music_chill", "intensity": 0.7})
    post("/voice/speak", {"text": "Radio on."})
    post("/audio/play", {"query": q})
    while True:
        time.sleep(2)


if __name__ == "__main__":
    main()
