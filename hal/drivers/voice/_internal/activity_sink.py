"""Fire-and-forget activity events to the companion Mac HUD.

Does not affect the voice pipeline. Failures are swallowed.
"""

from __future__ import annotations

import logging
import os
import threading
from typing import Any

import requests

logger = logging.getLogger("hal.voice")

_URL = os.environ.get("HAL_ACTIVITY_URL", "").strip()
_TOKEN = os.environ.get("HAL_ACTIVITY_TOKEN", "").strip()
_TIMEOUT = 0.4


def emit(phase: str, **fields: Any) -> None:
    if not _URL:
        return
    payload = {"source": "hal", "phase": phase}
    for key, value in fields.items():
        if value is not None:
            payload[key] = value

    def _post() -> None:
        try:
            headers = {"Content-Type": "application/json"}
            if _TOKEN:
                headers["Authorization"] = f"Bearer {_TOKEN}"
            requests.post(_URL, json=payload, headers=headers, timeout=_TIMEOUT)
        except Exception:
            logger.debug("activity emit failed", exc_info=True)

    threading.Thread(target=_post, name="activity-emit", daemon=True).start()
