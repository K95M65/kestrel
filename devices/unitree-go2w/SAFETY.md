---
schema: autonomous.safety.v1
motion:
  stop_always: true
---

# SAFETY.md — Unitree Go2-W

This is a declaration-only safety reference, not evidence of a deployed safety
implementation. A supported Unitree port must implement and validate the controls
below before claiming the device is runnable.

## motion

Motion here is **locomotion** (the Unitree SDK drives wheels/legs, not Feetech servos).

- **E-stop is immediate** — it halts all drive instantly and does not queue behind the
  runtime, the network, or any in-flight skill.
- Speed and acceleration are **bounded**, and slower near detected people.
- **Obstacle stop**: the 3D depth camera halts motion before contact; the runtime cannot
  override it.
- No autonomous motion toward a person without explicit intent; no motion during a
  localization or sensor fault.
- Motion is **announced** before it happens ("moving to the kitchen").

## fail-safe states

| Condition | Behavior |
|-----------|----------|
| Network loss | Stop; hold position; local reflexes only |
| Depth / sensor fault | Stop motion; report health |
| Runtime unreachable | System Managers run; no agent-driven motion |
| Low battery / thermal | Return-to-dock or safe stop |
| E-stop pressed | All motion off until reset |
