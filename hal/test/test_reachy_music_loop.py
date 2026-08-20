"""Reachy music groove repeats until music_stop (hal/drivers/motors/reachy_service.py).

The Pollen SDK's play_move() runs a recorded move exactly once, so without the
repeat in _play_recording the robot danced a few seconds and then sat still for
the rest of the track — where the feetech backend loops the groove
(animation_service._continue_playback).

The reachy_mini SDK is not installed on dev machines, so it is stubbed here;
numpy/scipy come from hal's venv (imported at module level by the driver).
"""
import math
import os
import sys
import threading
import time
import types
import unittest

sys.path.append(os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__)))))


def _install_sdk_stub():
    """Register a minimal fake `reachy_mini` package before importing the driver."""
    if "reachy_mini" in sys.modules:
        return
    pkg = types.ModuleType("reachy_mini")
    pkg.ReachyMini = object          # replaced per-test by a fake instance
    utils = types.ModuleType("reachy_mini.utils")
    utils.create_head_pose = lambda **kwargs: None
    motion = types.ModuleType("reachy_mini.motion")
    recorded = types.ModuleType("reachy_mini.motion.recorded_move")
    recorded.RecordedMoves = object
    core = types.ModuleType("reachy_mini.reachy_mini")
    core.SLEEP_ANTENNAS_JOINT_POSITIONS = [-3.05, 3.05]
    core.SLEEP_HEAD_POSE = "sleep-head"
    sys.modules.update({
        "reachy_mini": pkg,
        "reachy_mini.utils": utils,
        "reachy_mini.motion": motion,
        "reachy_mini.motion.recorded_move": recorded,
        "reachy_mini.reachy_mini": core,
    })


_install_sdk_stub()

import hal.presets as P  # noqa: E402
import hal.drivers.motors.reachy_service as reachy_service  # noqa: E402
from hal.drivers.motors.reachy_service import ReachyMotionService, _TimeScaledMove  # noqa: E402

reachy_service._SLEEP_SETTLE_S = 0.0

_MOVE_DURATION_S = 0.02


class FakeMini:
    """Daemon client stand-in: records every played move, honours cancel_move."""

    def __init__(self):
        self.played = []
        self.ramps = []
        self.gotos = []
        self.sleeps = 0
        self.wakes = 0
        self.enables = 0
        self.cancels = 0
        self.head_pose = None
        self.head_joints = None
        self.antennas = None
        self.max_concurrent_plays = 0
        self._playing = 0
        self._lock = threading.Lock()
        self._cancel = threading.Event()

    def play_move(self, move, initial_goto_duration=0.0, **kwargs):
        with self._lock:
            self.played.append(move)
            self.ramps.append(initial_goto_duration)
            self._playing += 1
            self.max_concurrent_plays = max(self.max_concurrent_plays, self._playing)
        try:
            # A real move blocks for its duration; cancel_move cuts it short.
            self._cancel.wait(_MOVE_DURATION_S)
            self._cancel.clear()
        finally:
            with self._lock:
                self._playing -= 1

    def cancel_move(self):
        self.cancels += 1
        self._cancel.set()

    def enable_motors(self):
        self.enables += 1

    def disable_motors(self):
        pass

    def wake_up(self):
        self.wakes += 1

    def get_current_head_pose(self):
        if self.head_pose is None:
            raise RuntimeError("no pose read in tests — driver falls back to _target")
        return self.head_pose

    def get_current_joint_positions(self):
        if self.head_joints is None:
            raise RuntimeError("no joint read in tests")
        return list(self.head_joints), list(self.antennas)

    def goto_sleep(self):
        self.sleeps += 1

    def goto_target(self, **kwargs):
        self.gotos.append(kwargs)

    def set_target(self, **kwargs):
        self.gotos.append(kwargs)

    class _Client:
        def disconnect(self):
            pass

    @property
    def client(self):
        return self._Client()

    def play_count(self, name=None):
        with self._lock:
            if name is None:
                return len(self.played)
            return sum(1 for m in self.played if m == name)


class FakeMoves:
    """RecordedMoves stand-in — a move IS its HF name, so calls are inspectable.

    Unknown names raise like the real library does (it has `dance1`, not `dance`).
    """

    _KNOWN = ("dance1", "dance2", "dance3", "curious1", "cheerful1", "thoughtful1",
              "sad1", "welcoming1", "amazed1")

    def get(self, name):
        if name not in self._KNOWN:
            raise RuntimeError(
                f"Move {name} not found in recorded moves library "
                "pollen-robotics/reachy-mini-emotions-library"
            )
        return name

    def list_moves(self):
        return list(self._KNOWN)


def _wait_until(predicate, timeout=3.0):
    deadline = time.monotonic() + timeout
    while time.monotonic() < deadline:
        if predicate():
            return True
        time.sleep(0.01)
    return predicate()


class TestReachyPlaybackLoop(unittest.TestCase):
    def setUp(self):
        self.svc = ReachyMotionService()
        self.mini = FakeMini()
        self.svc._mini = self.mini
        self.svc._moves = FakeMoves()   # skip the lazy HF loader

    def tearDown(self):
        self.svc._music_playing = False
        self.svc._music_recording = None
        self.svc._suppressed = True
        self.mini.cancel_move()
        thread = self.svc._play_thread
        if thread:
            thread.join(timeout=2.0)

    def test_single_play_runs_once(self):
        """A plain emotion play must NOT repeat — only music does."""
        self.svc.dispatch(P.SERVO_CMD_PLAY, P.SERVO_CURIOUS)
        self.svc._play_thread.join(timeout=2.0)
        self.assertEqual(self.mini.play_count("curious1"), 1)

    def test_music_start_repeats_until_stop(self):
        self.svc.dispatch(P.SERVO_CMD_MUSIC_START, P.SERVO_MUSIC_GROOVE)
        self.assertTrue(
            _wait_until(lambda: self.mini.play_count("dance1") >= 3),
            f"groove did not repeat: {self.mini.played}",
        )

        self.svc.dispatch(P.SERVO_CMD_MUSIC_STOP, None)
        self.svc._play_thread.join(timeout=2.0)
        settled = self.mini.play_count()
        time.sleep(_MOVE_DURATION_S * 5)
        self.assertEqual(self.mini.play_count(), settled, "groove kept playing after stop")

    def test_music_start_without_style_falls_back_to_groove(self):
        self.svc.dispatch(P.SERVO_CMD_MUSIC_START, None)
        self.assertTrue(_wait_until(lambda: self.mini.play_count("dance1") >= 1))

    def test_emotion_during_music_returns_to_groove(self):
        """Mirrors feetech: a finished one-shot hands the servo back to music."""
        self.svc.dispatch(P.SERVO_CMD_MUSIC_START, P.SERVO_MUSIC_JAZZ)   # dance2
        self.assertTrue(_wait_until(lambda: self.mini.play_count("dance2") >= 1))

        self.svc.dispatch(P.SERVO_CMD_PLAY, P.SERVO_CURIOUS)
        self.assertTrue(_wait_until(lambda: self.mini.play_count("curious1") >= 1))
        # The emotion play thread keeps grooving once its one-shot is done.
        played_after = self.mini.play_count("dance2")
        self.assertTrue(
            _wait_until(lambda: self.mini.play_count("dance2") > played_after),
            f"groove did not resume after emotion: {self.mini.played}",
        )

    def test_unknown_recording_during_music_keeps_the_groove(self):
        """The agent sends names the HF library lacks (`dance`, not `dance1`).

        That must not end the dance for the rest of the track — it did, which
        is how the robot went still mid-song on the first live run.
        """
        self.svc.dispatch(P.SERVO_CMD_MUSIC_START, P.SERVO_MUSIC_GROOVE)
        self.assertTrue(_wait_until(lambda: self.mini.play_count("dance1") >= 1))

        self.svc.dispatch(P.SERVO_CMD_PLAY, "dance")   # not in the HF library
        played_after = self.mini.play_count("dance1")
        self.assertTrue(
            _wait_until(lambda: self.mini.play_count("dance1") > played_after),
            f"groove died on an unknown recording: {self.mini.played}",
        )
        self.assertEqual(self.mini.play_count("dance"), 0, "bogus move must not play")

    def test_groove_move_that_always_fails_does_not_spin(self):
        """A broken groove must end the thread, not busy-loop on failures."""
        class DeadMoves(FakeMoves):
            def get(self, name):
                raise RuntimeError(f"Move {name} not found")

        self.svc._moves = DeadMoves()
        self.svc.dispatch(P.SERVO_CMD_MUSIC_START, P.SERVO_MUSIC_GROOVE)
        self.svc._play_thread.join(timeout=2.0)
        self.assertFalse(self.svc._play_thread.is_alive(), "play thread spun on a failing move")
        self.assertEqual(self.mini.play_count(), 0)

    def test_hold_stops_the_groove(self):
        """suppress-then-cancel ordering: hold must not be outrun by a repeat."""
        self.svc.dispatch(P.SERVO_CMD_MUSIC_START, P.SERVO_MUSIC_GROOVE)
        self.assertTrue(_wait_until(lambda: self.mini.play_count("dance1") >= 1))

        self.svc.hold()
        self.svc._play_thread.join(timeout=2.0)
        settled = self.mini.play_count()
        time.sleep(_MOVE_DURATION_S * 5)
        self.assertEqual(self.mini.play_count(), settled, "groove survived hold()")

    def test_stop_ends_the_groove(self):
        self.svc.dispatch(P.SERVO_CMD_MUSIC_START, P.SERVO_MUSIC_GROOVE)
        self.assertTrue(_wait_until(lambda: self.mini.play_count("dance1") >= 1))

        self.svc.stop(timeout=2.0)
        self.assertFalse(self.svc._music_playing)
        settled = self.mini.play_count()
        time.sleep(_MOVE_DURATION_S * 5)
        self.assertEqual(self.mini.play_count(), settled, "groove survived stop()")


class TestPlayRamp(unittest.TestCase):
    """Every move must be entered through a ramp, never at initial_goto_duration=0."""

    class _RampMoves(FakeMoves):
        """Moves whose frame 0 sits far from the current pose (big yaw jump)."""

        def get(self, name):
            name = super().get(name)
            import numpy as np
            from scipy.spatial.transform import Rotation

            pose = np.eye(4)
            pose[:3, :3] = Rotation.from_euler("xyz", [0, 0, 90], degrees=True).as_matrix()
            move = types.SimpleNamespace(name=name)
            move.evaluate = lambda t: (pose, np.array([0.0, 0.0]), 0.0)
            return move

    def _svc(self, policy=None):
        svc = ReachyMotionService(safety_policy=policy)
        svc._mini = FakeMini()
        svc._moves = FakeMoves()
        return svc

    def test_play_passes_a_nonzero_ramp(self):
        svc = self._svc()
        svc.dispatch(P.SERVO_CMD_PLAY, P.SERVO_CURIOUS)
        svc._play_thread.join(timeout=2.0)
        self.assertTrue(svc._mini.ramps, "no move played")
        self.assertGreater(svc._mini.ramps[0], 0.0, "move entered with the SDK's 0.0 snap")

    def test_far_frame_zero_is_stretched_by_the_speed_gate(self):
        from hal.safety.policy import MotionBounds, SafetyPolicy

        policy = SafetyPolicy(schema="autonomous.safety.v1", motion=MotionBounds(max_speed=60))
        svc = self._svc(policy)
        svc._moves = self._RampMoves()
        svc.dispatch(P.SERVO_CMD_PLAY, P.SERVO_CURIOUS)
        svc._play_thread.join(timeout=2.0)
        # 90° of yaw at 60 deg/s cannot be done in the 0.5s default ramp.
        self.assertGreaterEqual(svc._mini.ramps[0], 90.0 / 60.0 - 0.01)

    class _SweepMoves(FakeMoves):
        """A 90° yaw sweep in 0.2s — 450 deg/s, well above a 30 deg/s ceiling.

        evaluate raises on t >= duration, matching Pollen RecordedMove
        (HF emotions start at 0, so duration == timestamps[-1]).
        """

        duration = 0.2
        delta_deg = 90.0

        def get(self, name):
            name = super().get(name)
            import numpy as np
            from scipy.spatial.transform import Rotation

            dur = self.duration
            delta = self.delta_deg
            evaluated_at = []

            def evaluate(t):
                t = float(t)
                evaluated_at.append(t)
                if t < 0 or t >= dur:
                    raise ValueError(
                        f"t={t} outside [0, {dur}) — RecordedMove.evaluate contract"
                    )
                frac = t / dur
                pose = np.eye(4)
                pose[:3, :3] = Rotation.from_euler(
                    "xyz", [0, 0, delta * frac], degrees=True
                ).as_matrix()
                return pose, np.array([0.0, 0.0]), 0.0

            move = types.SimpleNamespace(
                name=name, duration=dur, evaluate=evaluate, evaluated_at=evaluated_at
            )
            return move

    def test_fast_recorded_play_is_stretched_by_the_speed_gate(self):
        """dispatch → play_move is the emotion path. Peak 90° in 0.2s at 30 deg/s
        needs >= 3.0s; the wrapper's duration is that stretch. Sampling must
        not evaluate t == duration (that raise used to fail-open the gate)."""
        from hal.safety.policy import MotionBounds, SafetyPolicy

        max_speed = 30
        policy = SafetyPolicy(
            schema="autonomous.safety.v1", motion=MotionBounds(max_speed=max_speed)
        )
        svc = self._svc(policy)
        svc._moves = self._SweepMoves()
        svc.dispatch(P.SERVO_CMD_PLAY, P.SERVO_CURIOUS)
        svc._play_thread.join(timeout=2.0)
        self.assertTrue(svc._mini.played, "play_move was not the play entry")
        played = svc._mini.played[0]
        self.assertIsInstance(played, _TimeScaledMove)
        needed = self._SweepMoves.delta_deg / max_speed
        self.assertGreaterEqual(float(played.duration), needed - 0.05)
        inner = played._inner
        self.assertTrue(inner.evaluated_at, "speed gate never sampled the move")
        self.assertLess(
            max(inner.evaluated_at),
            self._SweepMoves.duration,
            "sampler evaluated t >= duration (RecordedMove would raise)",
        )
        self.assertEqual(svc._mini.sleeps, 0)

    def test_recorded_play_passthrough_without_a_speed_bound(self):
        svc = self._svc()
        svc._moves = self._SweepMoves()
        svc.dispatch(P.SERVO_CMD_PLAY, P.SERVO_CURIOUS)
        svc._play_thread.join(timeout=2.0)
        self.assertAlmostEqual(float(svc._mini.played[0].duration), self._SweepMoves.duration, places=3)


class SlowMoves(FakeMoves):
    """First get() is slow — reproduces a play thread stalled in the HF load."""

    def __init__(self, delay=0.15):
        self._delay = delay

    def get(self, name):
        time.sleep(self._delay)
        return super().get(name)


class TestServoOwnership(unittest.TestCase):
    """One writer at a time: a pose command and a move must not both stream."""

    def setUp(self):
        self.svc = ReachyMotionService()
        self.mini = FakeMini()
        self.svc._mini = self.mini
        self.svc._moves = FakeMoves()

    def tearDown(self):
        self.svc._music_playing = False
        self.svc._music_recording = None
        self.svc._suppressed = True
        self.mini.cancel_move()
        if self.svc._play_thread:
            self.svc._play_thread.join(timeout=2.0)

    def test_move_to_cancels_a_running_animation(self):
        self.svc.dispatch(P.SERVO_CMD_PLAY, P.SERVO_CURIOUS)
        self.assertTrue(_wait_until(lambda: self.mini.play_count("curious1") >= 1))
        before = self.mini.cancels

        self.svc.move_to({"head_yaw.pos": 10.0}, duration=0.05)
        self.assertGreater(self.mini.cancels, before, "move_to streamed against a move")
        self.assertTrue(self.mini.gotos, "move_to did not reach the daemon")

    def test_aim_hands_the_servo_back_to_the_groove(self):
        self.svc.dispatch(P.SERVO_CMD_MUSIC_START, P.SERVO_MUSIC_GROOVE)
        self.assertTrue(_wait_until(lambda: self.mini.play_count("dance1") >= 1))

        self.svc.aim(P.AIM_LEFT, 0.05, self.svc.get_positions(), None)
        played = self.mini.play_count("dance1")
        self.assertTrue(
            _wait_until(lambda: self.mini.play_count("dance1") > played),
            "groove never resumed after aim",
        )

    def test_two_plays_never_overlap(self):
        for name in (P.SERVO_CURIOUS, P.SERVO_HAPPY_WIGGLE, P.SERVO_THINKING_DEEP,
                     P.SERVO_CURIOUS, P.SERVO_HAPPY_WIGGLE):
            self.svc.dispatch(P.SERVO_CMD_PLAY, name)
        if self.svc._play_thread:
            self.svc._play_thread.join(timeout=3.0)
        self.assertLessEqual(
            self.mini.max_concurrent_plays, 1,
            f"two move streams overlapped: {self.mini.played}",
        )

    def test_play_stalled_in_the_library_never_starts_late(self):
        """The cancel misses a thread still inside moves.get() — it must skip."""
        self.svc._moves = SlowMoves()
        self.svc.dispatch(P.SERVO_CMD_PLAY, P.SERVO_CURIOUS)
        self.svc.dispatch(P.SERVO_CMD_PLAY, P.SERVO_HAPPY_WIGGLE)
        if self.svc._play_thread:
            self.svc._play_thread.join(timeout=3.0)
        time.sleep(0.25)   # let the superseded thread finish its slow get()
        self.assertEqual(
            self.mini.play_count("curious1"), 0,
            f"superseded play started anyway: {self.mini.played}",
        )
        self.assertEqual(self.mini.play_count("cheerful1"), 1)

    def test_pose_command_clears_the_reported_recording(self):
        """GET /servo must not keep naming the animation the aim just cancelled."""
        self.svc.dispatch(P.SERVO_CMD_PLAY, P.SERVO_CURIOUS)
        self.assertTrue(_wait_until(lambda: self.svc._current_recording == P.SERVO_CURIOUS))

        self.svc.aim(P.AIM_LEFT, 0.05, self.svc.get_positions(), None)
        self.assertIsNone(self.svc._current_recording)

    def test_superseded_play_never_publishes_its_name(self):
        """GET /servo must name the winner, not a thread that lost the servo."""
        self.svc._moves = SlowMoves(delay=0.1)
        self.svc.dispatch(P.SERVO_CMD_PLAY, P.SERVO_SAD)
        self.svc.dispatch(P.SERVO_CMD_PLAY, P.SERVO_GREETING)
        winner = self.svc._play_thread
        if winner:
            winner.join(timeout=3.0)
        time.sleep(0.3)   # let the superseded thread finish its slow get()
        self.assertNotEqual(self.svc._current_recording, P.SERVO_SAD)

    def test_aim_without_music_does_not_start_an_animation(self):
        self.svc.aim(P.AIM_RIGHT, 0.05, self.svc.get_positions(), None)
        time.sleep(0.15)
        self.assertEqual(self.mini.play_count(), 0)


class TestFreezeAndHold(unittest.TestCase):
    """Camera freeze and /servo/hold must not shred animations or gag emotions."""

    def setUp(self):
        self.svc = ReachyMotionService()
        self.mini = FakeMini()
        self.svc._mini = self.mini
        self.svc._moves = FakeMoves()

    def tearDown(self):
        self.svc._music_playing = False
        self.svc._music_recording = None
        self.svc._released = True
        self.mini.cancel_move()
        if self.svc._play_thread:
            self.svc._play_thread.join(timeout=2.0)

    def test_freeze_does_not_cancel_the_move_in_flight(self):
        """A vision snapshot every few seconds must not chop the animation."""
        self.svc.dispatch(P.SERVO_CMD_PLAY, P.SERVO_CURIOUS)
        self.assertTrue(_wait_until(lambda: self.mini.play_count("curious1") >= 1))
        before = self.mini.cancels

        self.svc.freeze()
        self.assertEqual(self.mini.cancels, before, "freeze cancelled a running move")
        self.assertTrue(self.svc.is_frozen)

    def test_freeze_stops_the_groove_at_the_next_pass(self):
        self.svc.dispatch(P.SERVO_CMD_MUSIC_START, P.SERVO_MUSIC_GROOVE)
        self.assertTrue(_wait_until(lambda: self.mini.play_count("dance1") >= 1))

        self.svc.freeze()
        if self.svc._play_thread:
            self.svc._play_thread.join(timeout=2.0)
        settled = self.mini.play_count()
        time.sleep(_MOVE_DURATION_S * 5)
        self.assertEqual(self.mini.play_count(), settled, "groove kept going while frozen")

        self.svc.unfreeze()
        self.assertTrue(
            _wait_until(lambda: self.mini.play_count() > settled),
            "groove did not resume after unfreeze",
        )

    def test_emotion_still_plays_during_a_hold(self):
        """The emotion route decides what a hold blocks — the driver must not
        drop what the route let through."""
        self.svc.hold()
        self.svc.dispatch(P.SERVO_CMD_PLAY, P.SERVO_GREETING)
        self.assertTrue(
            _wait_until(lambda: self.mini.play_count("welcoming1") >= 1),
            "hold swallowed an emotion the route allowed",
        )

    def test_hold_exposes_the_flags_the_routes_read(self):
        self.svc.hold(explicit=True)
        self.assertTrue(self.svc._hold_mode)
        self.assertTrue(self.svc._hold_explicit)
        self.assertTrue(self.svc.is_suppressed)      # /servo/play refuses

        self.svc.resume()
        self.assertFalse(self.svc._hold_mode)
        self.assertFalse(self.svc._hold_explicit)
        self.assertFalse(self.svc.is_suppressed)

    def test_hold_stops_ambient_motion(self):
        self.svc.dispatch(P.SERVO_CMD_MUSIC_START, P.SERVO_MUSIC_GROOVE)
        self.assertTrue(_wait_until(lambda: self.mini.play_count("dance1") >= 1))

        self.svc.hold()
        if self.svc._play_thread:
            self.svc._play_thread.join(timeout=2.0)
        settled = self.mini.play_count()
        time.sleep(_MOVE_DURATION_S * 5)
        self.assertEqual(self.mini.play_count(), settled, "groove survived hold()")

    def test_released_robot_refuses_every_play(self):
        self.svc.release()
        self.svc.dispatch(P.SERVO_CMD_PLAY, P.SERVO_CURIOUS)
        time.sleep(0.15)
        self.assertEqual(self.mini.play_count(), 0)
        self.assertTrue(self.svc.is_suppressed)


class TestAvailableRecordings(unittest.TestCase):
    """GET /servo must list the same vocabulary its `current` field reports."""

    def setUp(self):
        self.svc = ReachyMotionService()
        self.svc._moves = FakeMoves()

    def test_mapped_moves_are_listed_under_their_hal_name(self):
        listed = self.svc.get_available_recordings()
        self.assertIn(P.SERVO_MUSIC_GROOVE, listed)       # not 'dance1'
        self.assertIn(P.SERVO_THINKING_DEEP, listed)      # not 'thoughtful1'
        self.assertNotIn("dance1", listed)
        self.assertNotIn("thoughtful1", listed)

    def test_unmapped_hf_moves_stay_listed_verbatim(self):
        self.assertIn("amazed1", self.svc.get_available_recordings())

    def test_current_recording_is_always_in_the_list(self):
        """What the web highlights (`current`) must exist among the items."""
        listed = set(self.svc.get_available_recordings())
        for name in (P.SERVO_MUSIC_GROOVE, P.SERVO_CURIOUS, P.SERVO_HAPPY_WIGGLE):
            self.assertIn(name, listed)

    def test_no_move_library_returns_empty(self):
        self.svc._moves = False
        self.assertEqual(self.svc.get_available_recordings(), [])


def _antenna_gotos(gotos):
    return [g for g in gotos if g.get("antennas") is not None]


class TestReachySleepFoldsAntennas(unittest.TestCase):
    """Sleep must fold the ears down and must not call SDK goto_sleep()."""

    def setUp(self):
        self.svc = ReachyMotionService()
        self.mini = FakeMini()
        self.svc._mini = self.mini
        self.svc._moves = FakeMoves()

    def tearDown(self):
        self.svc._music_playing = False
        thread = self.svc._play_thread
        if thread:
            thread.join(timeout=2.0)

    def test_release_parks_antennas_and_skips_sdk_goto_sleep(self):
        self.svc.release()
        self.assertEqual(self.mini.sleeps, 0)
        folded = _antenna_gotos(self.mini.gotos)
        self.assertTrue(folded, "release() did not command the antennas")
        last = folded[-1]["antennas"]
        self.assertAlmostEqual(float(last[0]), -3.05, places=2)
        self.assertAlmostEqual(float(last[1]), 3.05, places=2)

    def test_sleepy_play_folds_antennas_instead_of_tired1(self):
        self.svc.dispatch(P.SERVO_CMD_PLAY, P.SERVO_SLEEPY)
        thread = self.svc._play_thread
        self.assertIsNotNone(thread)
        thread.join(timeout=2.0)
        self.assertEqual(self.mini.play_count(), 0)
        self.assertEqual(self.mini.sleeps, 0)
        folded = _antenna_gotos(self.mini.gotos)
        self.assertTrue(folded, "sleepy play did not command the antennas")
        last = folded[-1]["antennas"]
        self.assertAlmostEqual(float(last[0]), -3.05, places=2)
        self.assertAlmostEqual(float(last[1]), 3.05, places=2)

    def test_stop_parks_without_sdk_goto_sleep(self):
        self.svc.stop(timeout=0.1)
        self.assertEqual(self.mini.sleeps, 0)
        self.assertTrue(_antenna_gotos(self.mini.gotos))


def _sleep_to_init_antenna_delta_deg():
    """Largest INIT-bound antenna travel, from the shipped sleep/INIT constants."""
    return abs(
        math.degrees(reachy_service._SLEEP_ANTENNAS_RAD[0])
        - math.degrees(reachy_service._INIT_ANTENNAS_RAD[0])
    )


class TestReachyWakeSpeedGate(unittest.TestCase):
    """resume/start must stretch INIT from a folded pose; SDK wake_up is a snap."""

    def setUp(self):
        from hal.safety.policy import MotionBounds, SafetyPolicy

        self.max_speed = 30
        self.delta = _sleep_to_init_antenna_delta_deg()
        self.needed = self.delta / self.max_speed
        self.policy = SafetyPolicy(
            schema="autonomous.safety.v1",
            motion=MotionBounds(max_speed=self.max_speed),
        )

    def _sleep_pose_mini(self):
        import numpy as np

        mini = FakeMini()
        mini.head_pose = np.eye(4)
        mini.head_joints = [0.0] * 7
        mini.antennas = list(reachy_service._SLEEP_ANTENNAS_RAD)
        return mini

    def _assert_stretched_init(self, mini):
        self.assertEqual(mini.wakes, 0, "SDK wake_up() is the ungated snap")
        self.assertGreaterEqual(mini.enables, 1, "wake did not enable motors")
        gotos = _antenna_gotos(mini.gotos)
        self.assertTrue(gotos, "wake never commanded a goto")
        last = gotos[-1]
        self.assertGreaterEqual(float(last["duration"]), self.needed - 0.05)
        ants = last["antennas"]
        self.assertAlmostEqual(float(ants[0]), reachy_service._INIT_ANTENNAS_RAD[0], places=2)
        self.assertAlmostEqual(float(ants[1]), reachy_service._INIT_ANTENNAS_RAD[1], places=2)

    def test_resume_from_sleep_fold_is_stretched_by_the_speed_gate(self):
        svc = ReachyMotionService(safety_policy=self.policy)
        mini = self._sleep_pose_mini()
        svc._mini = mini
        svc._released = True
        svc.resume()
        self._assert_stretched_init(mini)
        self.assertFalse(svc._released)

    def test_resume_without_live_pose_does_not_snap(self):
        """Unreadable pose must not fall back to the vendor 2s INIT snap."""
        svc = ReachyMotionService(safety_policy=self.policy)
        mini = FakeMini()  # pose reads raise
        svc._mini = mini
        svc._released = True
        svc.resume()
        self._assert_stretched_init(mini)

    def test_start_from_sleep_fold_is_stretched_by_the_speed_gate(self):
        mini = self._sleep_pose_mini()
        orig = reachy_service.ReachyMini
        reachy_service.ReachyMini = lambda **kwargs: mini
        try:
            svc = ReachyMotionService(safety_policy=self.policy)
            svc.start()
        finally:
            reachy_service.ReachyMini = orig
        self._assert_stretched_init(mini)

    def test_resume_passthrough_without_a_speed_bound(self):
        svc = ReachyMotionService()
        mini = self._sleep_pose_mini()
        svc._mini = mini
        svc.resume()
        self.assertEqual(mini.wakes, 0)
        last = _antenna_gotos(mini.gotos)[-1]
        self.assertAlmostEqual(
            float(last["duration"]), reachy_service._WAKE_GOTO_S, places=3
        )


if __name__ == "__main__":
    unittest.main()
