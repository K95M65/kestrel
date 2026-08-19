"""Short-lived follow-up focus for wake-word conversations."""

import threading
import time
from collections.abc import Callable


class WakeWordFocus:
    """Track a monotonic idle deadline shared by successive mic sessions.

    Idle time does not include the stretch where the robot is thinking or
    speaking. ``hold()`` / ``release()`` freeze expiry for that window so a
    25s Grok turn cannot silently drop the user's reply.
    """

    def __init__(self, timeout_s: float, clock: Callable[[], float] = time.monotonic):
        self._timeout_s = max(0.0, timeout_s)
        self._clock = clock
        self._until = 0.0
        self._holds = 0
        self._hold_deadline = 0.0
        self._lock = threading.Lock()

    def is_active(self) -> bool:
        with self._lock:
            now = self._clock()
            if self._holds > 0:
                if now <= self._hold_deadline:
                    return True
                self._holds = 0
                self._hold_deadline = 0.0
            if self._until <= now:
                self._until = 0.0
                return False
            return True

    def refresh(self) -> bool:
        """Extend focus from now; false when follow-up focus is disabled."""
        if self._timeout_s <= 0:
            return False
        with self._lock:
            self._until = self._clock() + self._timeout_s
        return True

    def hold(self, max_s: float = 120.0) -> None:
        """Freeze expiry while the robot is thinking or speaking."""
        if self._timeout_s <= 0:
            return
        with self._lock:
            now = self._clock()
            if self._holds <= 0 and self._until <= now:
                return
            self._holds += 1
            self._hold_deadline = max(self._hold_deadline, now + max(1.0, max_s))

    def release(self) -> bool:
        """Resume idle countdown from a full timeout after the last hold."""
        if self._timeout_s <= 0:
            return False
        with self._lock:
            if self._holds <= 0:
                return False
            self._holds -= 1
            if self._holds > 0:
                return True
            self._hold_deadline = 0.0
            self._until = self._clock() + self._timeout_s
            return True
