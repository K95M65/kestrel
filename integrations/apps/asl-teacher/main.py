#!/usr/bin/env python3
"""Short body-language lesson. Reachy has no fingers — we teach five phrases."""

from __future__ import annotations

import os
import time

import requests

HAL = os.environ.get("HAL_URL", "http://127.0.0.1:5001")
# (spoken gloss, emotion, line)
LESSONS = (
    ("hello", "greeting", "Hello. I wave hello."),
    ("yes", "nod", "Yes. A nod means yes."),
    ("no", "headshake", "No. A head shake means no."),
    ("thank you", "caring", "Thank you. Soft and warm."),
    ("happy", "happy", "Happy. That's a wiggle."),
)


def post(path: str, body: dict | None = None) -> None:
    try:
        requests.post(f"{HAL}{path}", json=body or {}, timeout=30).raise_for_status()
    except Exception as exc:
        print(f"{path}: {exc}", flush=True)


def main() -> None:
    post(
        "/voice/speak",
        {
            "text": "I don't have hands, so we'll learn five phrases with my head and body."
        },
    )
    time.sleep(4)
    for _gloss, emotion, line in LESSONS:
        post("/emotion", {"emotion": emotion, "intensity": 0.9})
        post("/voice/speak", {"text": line})
        time.sleep(5)
    post("/voice/speak", {"text": "Your turn. Say hello, yes, no, thank you, or happy."})
    post("/emotion", {"emotion": "listening", "intensity": 0.7})


if __name__ == "__main__":
    main()
