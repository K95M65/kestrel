"""Regression tests for reconnecting a provider that was down at HAL startup."""

import threading

from hal.realtime.orchestrator import RealtimeOrchestrator


class _Agent:
    available = False

    def __init__(self, fail: bool) -> None:
        self._fail = fail
        self.disconnected = False

    def connect(self) -> None:
        if self._fail:
            raise RuntimeError("temporary upstream outage")
        self.available = True

    def disconnect(self) -> None:
        self.disconnected = True
        self.available = False


class _Context:
    def build_instructions(self) -> str:
        return "test instructions"


def _orchestrator_for_initial_retry() -> RealtimeOrchestrator:
    orchestrator = object.__new__(RealtimeOrchestrator)
    orchestrator._agent = _Agent(fail=True)
    orchestrator._started = threading.Event()
    orchestrator._started.set()
    orchestrator._connect_retry_stop = threading.Event()
    orchestrator._connect_retry_thread = None
    orchestrator._rebuild_lock = threading.Lock()
    orchestrator._rebuild_done = threading.Event()
    orchestrator._rebuild_done.set()
    orchestrator._context = _Context()
    orchestrator._consecutive_silent = 0
    orchestrator._idle_reset_pending = False
    orchestrator._turns_since_recycle = 0
    orchestrator._looked_this_turn = False
    orchestrator._last_look_sent_monotonic = 0.0
    attempts = [_Agent(fail=False)]
    orchestrator._make_agent = lambda provider, instructions: attempts.pop(0)
    return orchestrator


def test_initial_connection_failure_recovers_without_a_hal_restart(monkeypatch):
    """A fresh session is built immediately; it is not gated on a voice turn."""
    orchestrator = _orchestrator_for_initial_retry()
    monkeypatch.setattr(
        "hal.realtime.orchestrator.config.REALTIME_PROVIDER", "gemini"
    )

    orchestrator._start_connect_retry_loop()
    assert orchestrator._connect_retry_thread is not None
    orchestrator._connect_retry_thread.join(timeout=1.0)

    assert orchestrator.available
    assert not orchestrator._connect_retry_thread.is_alive()
