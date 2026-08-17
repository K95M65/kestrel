---
schema: autonomous.device.v1
id: so101
name: LeRobot SO-101 (policy interface)
type: robot_arm
boards: [raspberry_pi_4, raspberry_pi_5, orangepi_sun60]
manufacturer: TheRobotStudio / Hugging Face LeRobot ecosystem
gateway:
  default: openclaw
  protocol: websocket
capabilities:
  vision: { routes: [camera], driver: opencv, required: true }
  policy: { routes: [policy], required: true, safety: SAFETY.md#motion }
  system: { routes: [system], required: true }
safety_ref: SAFETY.md
---

# LeRobot SO-101

This declaration exposes only the learned-policy **interface**. `POST
/policy/run` accepts a policy identifier and task, logs them as a dry run, and
does not load LeRobot or send a motor command. The body declares a camera
because a live visuomotor policy needs observations; its normal HAL lifecycle
is separate from a policy request.

The SO-101 motion driver, calibrated joint map, observation pipeline, and
safety-gated target executor are intentionally not declared yet. Once they
exist, this profile will add the `motion` capability and `/servo/stop` will
cancel the live policy worker before holding the arm. Until then,
`POST /policy/stop` clears a dry-run request without touching hardware.
