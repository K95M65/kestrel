"""Short-lived follow-up focus for wake-word conversations."""

import threading
import time
from collections.abc import Callable


class WakeWordFocus:
    """Track a monotonic idle deadline shared by successive mic sessions."""

    def __init__(self, timeout_s: float, clock: Callable[[], float] = time.monotonic):
        self._timeout_s = max(0.0, timeout_s)
        self._clock = clock
        self._until = 0.0
        self._lock = threading.Lock()

    def is_active(self) -> bool:
        with self._lock:
            if self._until <= self._clock():
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
