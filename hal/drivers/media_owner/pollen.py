"""Media handover with Pollen Robotics' reachy_mini daemon (Reachy Mini).

The daemon owns the camera and both ALSA PCMs for its own app runtime, so every
HAL capture fails with "Device or resource busy" until it lets go. It exposes a
supported handover for exactly that: POST /api/media/release gives the devices
to whoever asks, POST /api/media/acquire takes them back. Motion is unaffected —
the daemon keeps running and keeps driving the motors either way.

Why HAL performs the handover itself rather than a launcher doing it before
startup: ordering. The Reachy SDK happens to call release when a client
connects, but that runs inside HAL's motion-init thread and races audio
detection. Losing that race is silent and total — with the daemon still holding
the card, PortAudio cannot probe a single sample rate, the configured ALSA
output never enumerates, and TTS settles on output device -1 and raises on every
utterance while all the status endpoints still report healthy.

Reached over the same host/port the motion driver uses, so a robot that moved
its daemon needs one setting changed, not two.
"""
from __future__ import annotations

import logging
import os
import time

import requests

logger = logging.getLogger(__name__)


class PollenDaemonMediaOwner:
    """MediaOwner backed by the reachy_mini daemon's /api/media endpoints."""

    _TIMEOUT_S = 5.0
    # The daemon is a systemd service starting alongside HAL, so its HTTP port
    # may not be listening yet on a cold boot. Retry rather than lose the camera
    # and mic for the whole session over a two-second race.
    _RELEASE_ATTEMPTS = 5
    _RETRY_DELAY_S = 2.0

    def __init__(self, host: str | None = None, port: int | None = None):
        # Same defaults and env names as hal/drivers/motors/reachy_service.py —
        # HAL runs on the robot's own Pi, so the daemon is local.
        self._host = host or os.getenv("REACHY_DAEMON_HOST", "localhost")
        self._port = int(port or os.getenv("REACHY_DAEMON_PORT", "8000"))
        self._base = f"http://{self._host}:{self._port}/api/media"

    def _post(self, action: str) -> bool:
        resp = requests.post(f"{self._base}/{action}", timeout=self._TIMEOUT_S)
        resp.raise_for_status()
        logger.info("[media-owner] pollen %s -> %s", action, resp.text.strip()[:200])
        return True

    def release(self) -> bool:
        """Take the camera and audio devices from the daemon."""
        for attempt in range(1, self._RELEASE_ATTEMPTS + 1):
            try:
                return self._post("release")
            except Exception as e:
                if attempt == self._RELEASE_ATTEMPTS:
                    logger.warning(
                        "[media-owner] pollen release failed after %d attempts (%s) — "
                        "camera and audio will report 'device busy'",
                        self._RELEASE_ATTEMPTS, e,
                    )
                    return False
                logger.info(
                    "[media-owner] pollen release attempt %d/%d failed (%s) — retrying in %.0fs",
                    attempt, self._RELEASE_ATTEMPTS, e, self._RETRY_DELAY_S,
                )
                time.sleep(self._RETRY_DELAY_S)
        return False

    def acquire(self) -> bool:
        """Give the camera and audio devices back to the daemon.

        Single attempt: this is the shutdown path, where systemd is already
        counting down to SIGKILL, and a missed acquire is repaired by the next
        release anyway.
        """
        try:
            return self._post("acquire")
        except Exception as e:
            logger.warning(
                "[media-owner] pollen acquire failed (%s) — daemon stays without media", e
            )
            return False
