# Hello World Plugin

A minimal Autonomous OS plugin that pulses LEDs and speaks a greeting.

## Quick Start

1. Fork this repo
2. Edit `main.py` to do something cool
3. Push to HuggingFace / GitHub / GitLab
4. Install on your device: **Settings > Plugins > paste URL > Install**

## Plugin Format

```
your-plugin/
  plugin.json         # name, version, description, entry point
  main.py             # your Python code
  requirements.txt    # pip dependencies
  README.md           # description + demo video
```

## HAL API

Plugins control hardware via HAL's HTTP API (injected as `HAL_URL` env var, default `http://localhost:5001`):

```python
import os, requests

HAL = os.environ.get("HAL_URL", "http://localhost:5001")

# LED
requests.post(f"{HAL}/led/effect", json={"effect": "rainbow"})
requests.post(f"{HAL}/led/effect", json={"effect": "pulse", "color": "blue"})
requests.post(f"{HAL}/led/off")

# Voice
requests.post(f"{HAL}/voice/speak", json={"text": "Hello!"})

# Servo (devices with motion capability)
requests.post(f"{HAL}/servo/move", json={"pan": 45, "tilt": 10})

# Camera (devices with vision capability)
resp = requests.get(f"{HAL}/camera/snapshot")
```

## Tips

- Keep `requirements.txt` minimal — installs run on constrained devices
- Use `HAL_URL` env var, don't hardcode `localhost:5001`
- Add a demo video/GIF to your README for discoverability
- Tag your HuggingFace Space with `autonomous-os-plugin` for discovery
