#!/usr/bin/env python3
"""Photobooth: one snapshot, spoken cue, then idle."""

from __future__ import annotations

import os
import time

import requests

HAL = os.environ.get("HAL_URL", "http://127.0.0.1:5001")


def post(path: str, body: dict | None = None) -> None:
    try:
        requests.post(f"{HAL}{path}", json=body or {}, timeout=30).raise_for_status()
    except Exception as exc:
        print(f"{path}: {exc}", flush=True)


def main() -> None:
    post("/emotion", {"emotion": "happy", "intensity": 0.9})
    post("/voice/speak", {"text": "Say cheese!"})
    time.sleep(1.6)
    path = ""
    try:
        r = requests.get(
            f"{HAL}/camera/snapshot",
            params={"save": "true", "width": "768", "quality": "75"},
            timeout=20,
        )
        r.raise_for_status()
        path = str(r.json().get("path") or "")
    except Exception as exc:
        print(f"snapshot: {exc}", flush=True)
        post("/voice/speak", {"text": "I couldn't get a photo."})
        post("/emotion", {"emotion": "confused", "intensity": 0.6})
        return
    print(path, flush=True)
    post("/voice/speak", {"text": "Got it."})
    post("/emotion", {"emotion": "idle", "intensity": 0.4})


if __name__ == "__main__":
    main()
