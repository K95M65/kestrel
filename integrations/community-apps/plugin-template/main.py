"""Hello World plugin for Autonomous OS.

Pulses LED rainbow, speaks a greeting, waits, then turns off.
Fork this repo and modify to build your own plugin.

HAL API docs: see hal/routes/ in the main repo.
"""

import os
import time

import requests

HAL = os.environ.get("HAL_URL", "http://localhost:5001")


def main():
    # Rainbow LED effect
    requests.post(f"{HAL}/led/effect", json={"effect": "rainbow"}, timeout=5)

    # Voice greeting
    requests.post(
        f"{HAL}/voice/speak",
        json={"text": "Hello! I am a plugin running on Autonomous OS."},
        timeout=30,
    )

    # Keep running for 30 seconds (demo)
    time.sleep(30)

    # Clean up
    requests.post(f"{HAL}/led/off", timeout=5)


if __name__ == "__main__":
    main()
