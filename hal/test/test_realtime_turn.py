"""Regression tests for realtime-to-main-agent turn routing."""

from hal.drivers.voice._internal.realtime_turn import (
    RealtimeTurnResult,
    should_dispatch_to_main,
)


def test_armed_turn_falls_back_when_realtime_is_unavailable_or_silent():
    """A connected flag must not swallow a wake-word command without a reply."""
    no_reply = RealtimeTurnResult()

    assert should_dispatch_to_main(True, True, True, no_reply)


def test_armed_turn_stays_with_realtime_only_after_it_spoke():
    assert not should_dispatch_to_main(
        True, True, True, RealtimeTurnResult(handled=True, transcript="Xin chào")
    )


def test_wakeword_gate_still_rejects_unarmed_ambient_speech():
    assert not should_dispatch_to_main(True, False, True, RealtimeTurnResult())


def test_wakeword_disabled_preserves_the_always_listening_main_agent_sync():
    """The legacy path always sends the finalized STT turn to the OS server."""
    assert should_dispatch_to_main(
        False, False, True, RealtimeTurnResult(handled=True, transcript="Xin chào")
    )
