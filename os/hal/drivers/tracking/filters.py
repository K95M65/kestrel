"""Math helpers for the tracking pipeline: SmoothDamp servo follower step,
soft dead zone, alpha-beta centroid filter, and a time-aware PID."""

import time
from typing import Optional, Tuple

from hal.drivers.tracking import constants as C


def smooth_damp(current: float, target: float, velocity: float,
                smooth_time: float, dt: float, max_speed: float) -> Tuple[float, float]:
    """Critically-damped follower. Returns (new_position, new_velocity).

    Eases in and out toward `target`, carrying `velocity` across calls so
    retargeting mid-move stays smooth. Overshoot-clamped so it settles cleanly.
    (Game Programming Gems 4 / Unity's Mathf.SmoothDamp.)
    """
    smooth_time = max(1e-4, smooth_time)
    omega = 2.0 / smooth_time
    x = omega * dt
    exp = 1.0 / (1.0 + x + 0.48 * x * x + 0.235 * x * x * x)
    change = current - target
    # Clamp the max distance closed per unit smooth_time → caps peak speed.
    max_change = max_speed * smooth_time
    change = max(-max_change, min(max_change, change))
    new_target = current - change
    temp = (velocity + omega * change) * dt
    velocity = (velocity - omega * temp) * exp
    new = new_target + (change + temp) * exp
    # Prevent overshoot past the (clamped) target.
    if (target - current > 0.0) == (new > target):
        new = target
        velocity = (new - target) / dt if dt > 0 else 0.0
    return new, velocity


def soft_deadband(error: float, dz: float) -> float:
    """Continuous dead zone: 0 inside ±dz, then ramps from 0 at the edge.

    The old hard dead zone fed the controller the RAW error the instant the
    target crossed the boundary — output jumped from 0 to a full dz-worth of
    error, the "kick out of center" jerk. This shifts the error so it starts at
    0 at the edge and grows from there (no value step), giving a smooth handoff
    between holding and chasing. Sign-preserving.
    """
    if error > dz:
        return error - dz
    if error < -dz:
        return error + dz
    return 0.0


class AlphaBetaFilter2D:
    """Constant-velocity alpha-beta filter on a 2D point (the target centroid).

    Steady-state Kalman for a constant-velocity model (SORT/ByteTrack use a full
    Kalman; alpha-beta is the lightweight fixed-gain equivalent — no matrices,
    cheap on the A523). Smooths detector/tracker jitter, coasts through dropped
    or garbage frames via prediction, and exposes velocity so the servo can lead
    a moving target. `update()` gates a measurement whose residual from the
    prediction exceeds gate_px (ViT-bloat teleport / false detection): it then
    coasts on the prediction instead of snapping to the bad point.
    """

    def __init__(self, alpha: float, beta: float, gate_px: float,
                 gate_decay: float = C.AB_GATE_DECAY,
                 max_gated_streak: int = C.AB_MAX_GATED_STREAK):
        self.alpha, self.beta = alpha, beta
        self.gate_px, self.gate_decay = gate_px, gate_decay
        self.max_gated_streak = max_gated_streak
        self.x: Optional[float] = None
        self.y: Optional[float] = None
        self.vx: float = 0.0
        self.vy: float = 0.0
        self._gated_streak: int = 0

    def reset(self) -> None:
        self.x = self.y = None
        self.vx = self.vy = 0.0
        self._gated_streak = 0

    def update(self, mx: float, my: float, dt: float):
        """Feed a raw centroid measurement; return (fx, fy, vx, vy, gated)."""
        if self.x is None or self.y is None or dt <= 0:
            self.x, self.y = mx, my
            self.vx = self.vy = 0.0
            self._gated_streak = 0
            return self.x, self.y, self.vx, self.vy, False
        # Predict (constant velocity).
        px = self.x + self.vx * dt
        py = self.y + self.vy * dt
        rx, ry = mx - px, my - py
        gated = (rx * rx + ry * ry) ** 0.5 > self.gate_px
        # Hysteresis: a real fast move produces a large residual too, and would be
        # gated forever (velocity never updates → filter never catches up). After
        # a short streak of rejects, force-accept and re-seed to the measurement —
        # transient teleports (ViT bloat) last 1–2 frames, sustained motion does
        # not.
        if gated and self._gated_streak >= self.max_gated_streak:
            self.x, self.y = mx, my
            self.vx = self.vy = 0.0
            self._gated_streak = 0
            return self.x, self.y, self.vx, self.vy, False
        if gated:
            # Reject the measurement — coast on the prediction, bleed velocity so
            # a stuck/garbage lock decelerates instead of running off-frame.
            self._gated_streak += 1
            self.x, self.y = px, py
            self.vx *= self.gate_decay
            self.vy *= self.gate_decay
        else:
            self._gated_streak = 0
            self.x = px + self.alpha * rx
            self.y = py + self.alpha * ry
            self.vx += (self.beta / dt) * rx
            self.vy += (self.beta / dt) * ry
        return self.x, self.y, self.vx, self.vy, gated


class PID:
    """Time-aware PID with anti-windup. Industry-standard servo control."""

    def __init__(self, kp: float, ki: float, kd: float,
                 output_min: float = -C.PID_OUTPUT_MAX_DEG,
                 output_max: float = C.PID_OUTPUT_MAX_DEG,
                 integral_min: float = -C.PID_INTEGRAL_MAX,
                 integral_max: float = C.PID_INTEGRAL_MAX):
        self.kp, self.ki, self.kd = kp, ki, kd
        self.output_min, self.output_max = output_min, output_max
        self.integral_min, self.integral_max = integral_min, integral_max
        self._prev_error: float = 0.0
        self._integral: float = 0.0
        self._last_t: Optional[float] = None

    def reset(self) -> None:
        self._prev_error = 0.0
        self._integral = 0.0
        self._last_t = None

    def update(self, error: float) -> float:
        now = time.perf_counter()
        dt = 0.0 if self._last_t is None else (now - self._last_t)
        self._last_t = now
        p = self.kp * error
        if dt > 0:
            self._integral += error * dt
            self._integral = max(self.integral_min, min(self.integral_max, self._integral))
        i = self.ki * self._integral
        d = self.kd * (error - self._prev_error) / dt if dt > 0 else 0.0
        self._prev_error = error
        return max(self.output_min, min(self.output_max, p + i + d))
