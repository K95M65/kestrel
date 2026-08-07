"""Focused behavior tests for the tracking servo follower."""

import threading
import time

import pytest

from hal.drivers.tracking import constants as C
from hal.drivers.tracking import servo_follow
from hal.drivers.tracking.servo_follow import JOINTS, ServoFollower


def _pose(value: float) -> dict[str, float]:
    return {joint: value for joint in JOINTS}


class _FakeRobot:
    def __init__(self):
        self.actions: list[dict[str, float]] = []

    def send_action(self, action: dict[str, float]) -> None:
        self.actions.append(dict(action))


class _FakeAnimationService:
    def __init__(self):
        self.bus_lock = threading.RLock()
        self.is_frozen = False
        self.robot = _FakeRobot()


def test_hold_retargets_the_current_pose_instead_of_leaving_a_stale_goal():
    follower = ServoFollower()
    follower.set_goal(_pose(10.0))
    follower._yaw = 1.0
    follower._base_pitch = 2.0
    follower._elbow_pitch = 3.0
    follower._wrist_pitch = 4.0

    follower.hold()

    assert follower._goal == {
        "base_yaw.pos": 1.0,
        "base_pitch.pos": 2.0,
        "elbow_pitch.pos": 3.0,
        "wrist_pitch.pos": 4.0,
    }


def test_follower_uses_elapsed_time_and_bounds_scheduler_stalls():
    assert ServoFollower._tick_dt(10.040, 10.000) == pytest.approx(0.040)
    assert ServoFollower._tick_dt(11.000, 10.000) == C.SERVO_SUBSTEP_MAX_DT_S


def test_small_setpoint_changes_are_not_sent_until_they_are_meaningful():
    last_sent = _pose(0.0)
    assert not ServoFollower._should_write(_pose(C.SERVO_COMMAND_MIN_DELTA - 0.001), last_sent)
    assert ServoFollower._should_write(_pose(C.SERVO_COMMAND_MIN_DELTA), last_sent)
    assert ServoFollower._should_write(_pose(0.001), last_sent, force=True)


def test_worker_sends_the_final_goal_even_when_it_is_below_write_threshold():
    follower = ServoFollower()
    service = _FakeAnimationService()
    running = threading.Event()
    final_goal = C.SERVO_COMMAND_MIN_DELTA / 2
    follower.set_profile(smooth_time=0.001, max_speed_dps=55.0)
    follower.set_goal(_pose(final_goal))
    running.set()

    follower.start(service, running)
    deadline = time.monotonic() + 0.5
    while not service.robot.actions and time.monotonic() < deadline:
        time.sleep(0.01)
    running.clear()
    follower.join(timeout=0.5)

    assert service.robot.actions == [_pose(final_goal)]


def test_worker_coalesces_small_steps_until_they_accumulate(monkeypatch):
    follower = ServoFollower()
    service = _FakeAnimationService()
    running = threading.Event()
    follower.set_goal(_pose(C.SERVO_COMMAND_MIN_DELTA * 4))
    running.set()

    ticks = iter([0.0] + [0.03 * i for i in range(1, 20)])
    monkeypatch.setattr(servo_follow.time, "perf_counter", lambda: next(ticks))
    monkeypatch.setattr(servo_follow.time, "sleep", lambda _: None)

    def send_and_stop(action: dict[str, float]) -> None:
        service.robot.actions.append(dict(action))
        running.clear()

    service.robot.send_action = send_and_stop
    follower._worker(service, running)

    assert len(service.robot.actions) == 1
    first = service.robot.actions[0]["base_yaw.pos"]
    assert C.SERVO_COMMAND_MIN_DELTA <= first < C.SERVO_COMMAND_MIN_DELTA * 4
