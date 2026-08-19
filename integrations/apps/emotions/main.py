#!/usr/bin/env python3
"""Show each core expression with a short spoken label."""

from __future__ import annotations

import os
import time

import requests

HAL = os.environ.get("HAL_URL", "http://127.0.0.1:5001")
REEL = (
    ("greeting", "Hello."),
    ("curious", "Curious."),
    ("happy", "Happy."),
    ("laugh", "That's funny."),
    ("shy", "A little shy."),
    ("confused", "Hmm, confused."),
    ("sad", "Sad."),
    ("caring", "I care."),
    ("goodbye", "Bye for now."),
)


def post(path: str, body: dict | None = None) -> None:
    try:
        requests.post(f"{HAL}{path}", json=body or {}, timeout=30).raise_for_status()
    except Exception as exc:
        print(f"{path}: {exc}", flush=True)


def main() -> None:
    post("/voice/speak", {"text": "Here are my faces."})
    time.sleep(1.5)
    for emotion, line in REEL:
        post("/emotion", {"emotion": emotion, "intensity": 0.85})
        post("/voice/speak", {"text": line})
        time.sleep(3.5)
    post("/emotion", {"emotion": "idle", "intensity": 0.4})


if __name__ == "__main__":
    main()
