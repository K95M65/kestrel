"""Ask the companion Mac for a spoken reply instead of the Grok tool loop."""

from __future__ import annotations

import logging
import os
import time
from typing import Optional

import requests

logger = logging.getLogger("hal.voice")

_URL = os.environ.get("HAL_TALK_URL", "").strip()
_TOKEN = os.environ.get("HAL_ACTIVITY_TOKEN", "").strip() or os.environ.get(
    "HAL_TALK_TOKEN", ""
).strip()
_TIMEOUT = float(os.environ.get("HAL_TALK_TIMEOUT_S", "2"))
_COOLDOWN = float(os.environ.get("HAL_TALK_COOLDOWN_S", "20"))
_fail_until = 0.0


def try_talk(transcript: str) -> Optional[str]:
    """Return a spoken reply, or None to fall through to OpenClaw/Grok."""
    global _fail_until
    if not _URL or not (transcript or "").strip():
        return None
    if time.monotonic() < _fail_until:
        return None
    try:
        headers = {"Content-Type": "application/json"}
        if _TOKEN:
            headers["Authorization"] = f"Bearer {_TOKEN}"
        resp = requests.post(
            _URL,
            json={"input": transcript},
            headers=headers,
            timeout=_TIMEOUT,
        )
        if resp.status_code != 200:
            logger.warning("mac talk HTTP %s", resp.status_code)
            _fail_until = time.monotonic() + _COOLDOWN
            return None
        data = resp.json()
        if data.get("escalate"):
            logger.info("mac talk escalate source=%s", data.get("source"))
            return None
        speak = " ".join(str(data.get("speak") or "").split())
        if not speak:
            return None
        logger.info(
            "mac talk %sms source=%s %r",
            data.get("ms"),
            data.get("source"),
            speak[:120],
        )
        return speak
    except Exception as exc:
        _fail_until = time.monotonic() + _COOLDOWN
        logger.warning("mac talk failed: %s", exc)
        return None
