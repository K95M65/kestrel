---
schema: autonomous.device.v1
id: kestrel-host
name: Kestrel Host
type: desk_agent
boards: [sim]
gateway:
  default: openclaw
  protocol: websocket
capabilities:
  audio:      { routes: [audio, speaker, voice], required: true }
  vision:     { routes: [camera], driver: opencv, required: true }
  sensing:    { routes: [sensing], required: false }
  media:      { routes: [music], required: false }
  companion:  { routes: [buddy], required: false }
  system:     { routes: [system], required: true }
soul_ref:   SOUL.md
safety_ref: SAFETY.md
memory:     { backend: local }
---

# Kestrel Host

A computer that runs the stack: camera, mic, speaker, no motors. Same OS as
Reachy Mini. The chassis is a Linux VM, a Mac Mini, or any box with a webcam.

HAL boots with `HAL_BOARD=sim` and `HAL_SIMULATE=1` until you pass through a
real camera/mic (`HAL_SIM_MEDIA=host`). There is no servo driver and no GPIO.

This is not a mock test fixture (`robots/sim` is motion-only). It is a shippable
shape: audio + vision, no `motion`, so `make cts` does not demand an e-stop.
