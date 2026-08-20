"""Recorded/emotion play is speed-gated; recovery actions are not.

Drives the shipped lamp play entry (`_handle_play` → `_continue_playback`)
and asserts peak commanded deg/s cannot exceed a tight motion.max_speed.
halt/release/zero/hold/stop source must not stretch through that gate.
"""
from __future__ import annotations

import ast
import unittest
from pathlib import Path

from hal.safety.policy import SafetyPolicy, max_joint_delta, parse_safety

_HAL = Path(__file__).resolve().parents[1]
_LAMP_SRC = _HAL / "drivers" / "motors" / "animation_service.py"
_REACHY_SRC = _HAL / "drivers" / "motors" / "reachy_service.py"

_FM_SLOW = (
    "---\n"
    "schema: autonomous.safety.v1\n"
    "motion:\n"
    "  max_speed: 30\n"
    "  stop_always: true\n"
    "---\n"
)


class _FakeRobot:
    def __init__(self):
        self.sent = []

    def send_action(self, action):
        self.sent.append(dict(action))


def _try_animation_service():
    try:
        from hal.drivers.motors.animation_service import AnimationService
        return AnimationService
    except Exception:
        return None


AnimationService = _try_animation_service()


@unittest.skipIf(AnimationService is None, "feetech AnimationService not importable")
class TestLampRecordedPlayIsSpeedGated(unittest.TestCase):
    def setUp(self):
        self.max_speed = 30
        self.delta = 90.0
        self.fps = 10
        self.policy = parse_safety(_FM_SLOW)
        self.svc = AnimationService(
            port="none",
            lamp_id="gate-test",
            fps=self.fps,
            duration=0.1,
            idle_recording="spike",
            safety_policy=self.policy,
        )
        self.svc.robot = _FakeRobot()
        self.svc._current_state = {"base_yaw.pos": 0.0}
        self.svc._recording_cache["spike"] = [
            {"base_yaw.pos": 0.0},
            {"base_yaw.pos": self.delta},
        ]
        self.svc._no_idle_recordings = {"spike"}

    def test_handle_play_damps_so_peak_dps_cannot_exceed_bound(self):
        self.svc._handle_play("spike")
        # Bound the loop: interpolation + damped frames, plus a little slack.
        for _ in range(5000):
            if (
                self.svc._interpolation_frames <= 0
                and self.svc._current_frame_index >= len(self.svc._current_actions)
            ):
                break
            self.svc._continue_playback()
        sent = self.svc.robot.sent
        self.assertGreaterEqual(len(sent), 2, "play never commanded a pose")
        max_step = self.max_speed / float(self.fps)
        peak = 0.0
        for a, b in zip(sent, sent[1:]):
            peak = max(peak, max_joint_delta(a, b))
        self.assertLessEqual(peak, max_step + 1e-6)
        play_s = (len(sent) - 1) / float(self.fps)
        self.assertGreaterEqual(play_s, self.delta / self.max_speed - 0.15)

    def test_passthrough_without_a_speed_bound(self):
        self.svc._safety_policy = SafetyPolicy(schema="autonomous.safety.v1")
        self.svc._handle_play("spike")
        self.assertEqual(len(self.svc._current_actions), 2)


def _method_source(path: Path, class_name: str, method: str) -> str:
    text = path.read_text(encoding="utf-8")
    tree = ast.parse(text)
    for node in tree.body:
        if isinstance(node, ast.ClassDef) and node.name == class_name:
            for item in node.body:
                if isinstance(item, (ast.FunctionDef, ast.AsyncFunctionDef)) and item.name == method:
                    src = ast.get_source_segment(text, item)
                    if src is None:
                        raise AssertionError(f"no source segment for {class_name}.{method}")
                    return src
    raise AssertionError(f"{class_name}.{method} not found in {path}")


class TestRecoveryActionsStayUngated(unittest.TestCase):
    """stop/release/zero/hold must not stretch through the playback damper."""

    _GATES = (
        "damp_recorded_actions",
        "playback_time_scale",
        "_stretch_move",
        "_goto_awake_pose",
        "_wake_duration",
    )

    def _assert_ungated(self, path: Path, class_name: str, names):
        for name in names:
            src = _method_source(path, class_name, name)
            for gate in self._GATES:
                self.assertNotIn(
                    gate,
                    src,
                    f"{class_name}.{name} must not {gate} a recovery action",
                )

    def test_lamp_recovery_methods_do_not_call_the_play_gate(self):
        self._assert_ungated(
            _LAMP_SRC,
            "AnimationService",
            ("halt", "release", "zero_pose", "hold", "stop"),
        )

    def test_reachy_recovery_methods_do_not_call_the_play_gate(self):
        self._assert_ungated(
            _REACHY_SRC,
            "ReachyMotionService",
            ("halt", "release", "zero_pose", "hold", "stop"),
        )

    def test_reachy_start_and_resume_use_the_wake_gate(self):
        for name in ("start", "resume"):
            src = _method_source(_REACHY_SRC, "ReachyMotionService", name)
            self.assertIn(
                "_goto_awake_pose",
                src,
                f"{name} must command INIT through the speed-gated helper",
            )
            self.assertNotIn(
                "wake_up()",
                src,
                f"{name} must not call SDK wake_up() (fixed 2s snap + sound)",
            )


if __name__ == "__main__":
    unittest.main()
