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
    requests.post(f"{HAL}/led/set", json={"effect": "rainbow"}, timeout=5)

    # Voice greeting
    requests.post(
        f"{HAL}/audio/speak",
        json={"text": "Hello! I am a plugin running on Autonomous OS."},
        timeout=10,
    )

    # Keep running for 30 seconds (demo)
    time.sleep(30)

    # Clean up
    requests.post(f"{HAL}/led/off", timeout=5)


if __name__ == "__main__":
    main()
