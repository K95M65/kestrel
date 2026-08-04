from hal.drivers.voice._internal.wakeword_focus import WakeWordFocus


def test_focus_is_active_until_its_idle_deadline_and_refreshes():
    now = [10.0]
    focus = WakeWordFocus(20, clock=lambda: now[0])

    assert not focus.is_active()
    assert focus.refresh()
    assert focus.is_active()

    now[0] = 29.9
    assert focus.is_active()
    assert focus.refresh()

    now[0] = 49.1
    assert focus.is_active()
    now[0] = 50.0
    assert not focus.is_active()


def test_zero_timeout_disables_focus():
    focus = WakeWordFocus(0)

    assert not focus.refresh()
    assert not focus.is_active()
