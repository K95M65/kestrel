---
schema: autonomous.safety.v1
motion:
  stop_always: true
---

## motion

Any future SO-101 policy executor must route every generated joint target
through the HAL motion safety gate. `POST /servo/stop` remains unconditional
and must cancel the executor before holding the arm.
