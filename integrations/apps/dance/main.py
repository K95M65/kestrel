#!/usr/bin/env python3
"""Dance: cycle happy/excited/groove moves. Optional YouTube audio."""

from __future__ import annotations

import os
import time

import requests

HAL = os.environ.get("HAL_URL", "http://127.0.0.1:5001")
MOVES = ("excited", "happy", "laugh", "music_strong", "music_chill")


def post(path: str, body: dict | None = None) -> None:
    try:
        requests.post(f"{HAL}{path}", json=body or {}, timeout=30).raise_for_status()
    except Exception as exc:
        print(f"{path}: {exc}", flush=True)


def main() -> None:
    music = os.environ.get("DANCE_MUSIC", "").strip()
    try:
        seconds = int(os.environ.get("DANCE_SECONDS", "45"))
    except ValueError:
        seconds = 45
    if music:
        post("/voice/speak", {"text": "Let's dance!"})
        post("/audio/play", {"query": music})
    else:
        post("/voice/speak", {"text": "Watch me dance!"})
    deadline = time.time() + max(10, seconds)
    i = 0
    while time.time() < deadline:
        post("/emotion", {"emotion": MOVES[i % len(MOVES)], "intensity": 0.9})
        time.sleep(3.2)
        i += 1
    post("/audio/stop")
    post("/emotion", {"emotion": "idle", "intensity": 0.4})


if __name__ == "__main__":
    main()
