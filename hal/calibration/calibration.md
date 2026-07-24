# Servo Calibration

The arm uses 5x Feetech STS3215 servos controlled via the `lerobot` library. Each servo requires calibration to map raw encoder positions to degrees. Calibration data is stored as JSON and loaded at startup.

## Calibration files (repo default + per-device fleet override)

`homing_offset` / `range_min` / `range_max` are **unique to each physically-assembled
unit** — the servo horn mounts on a discrete spline, so the mechanical zero shifts a
few degrees per build. A calibration recorded on one arm only fits that arm; sharing
it makes another unit start with the wrong zero and travel limits. The follower/leader
configs therefore resolve `calibration_dir` in `__post_init__` two ways, keyed on
`HAL_DEVICE_ID`:

- **Default `id = "hal"`** (unset / dev) → the **version-controlled repo file**, so a
  fresh checkout or dev machine works out of the box:
  - `hal/follower/config_hal_follower.py` → `hal/calibration/robots/hal_follower`
  - `hal/leader/config_hal_leader.py` → `hal/calibration/teleoperators/hal_leader`
- **Per-device `id` (e.g. `lamp-abcd`)** → the **persistent fleet dir**
  `/var/lib/hal/calibration/robots/hal_follower/<id>.json` (override the base with
  `HAL_CALIBRATION_DIR`). The hardware team drops each unit's `<id>.json` there at
  provisioning; it lives **outside the OTA tree** (`/opt/hal` is overwritten every
  update) so calibration survives updates.
  - **Fallback:** if `HAL_DEVICE_ID` is set but that `<id>.json` is **missing**, the
    config logs a warning and falls back to the repo `hal.json` (it drops the effective
    id to `"hal"`), so the arm never starts uncalibrated — it runs on the shared
    calibration until the per-device file is in place.

> **Why override lerobot's default at all:** lerobot's per-user path
> (`~/.cache/huggingface/lerobot/calibration`) breaks when the service user differs
> from the user that ran calibration. `hal.service` runs as **root**, so it would look
> under `/root/.cache/...` and miss a calibration saved elsewhere — surfacing as
> `FeetechMotorsBus(...) has no calibration registered`. Both dirs above are
> independent of the service user.

lerobot loads `calibration_dir / f"{id}.json"`, where `id` is `HAL_DEVICE_ID`
(`hal/config.py`, default `"hal"`; `hal/.env.example` sets `HAL_DEVICE_ID=hal`).
With the default id the repo files are:

```
hal/calibration/
├── robots/hal_follower/hal.json
└── teleoperators/hal_leader/hal.json
```

On a deployed device the repo copy ships under `/opt/hal/calibration/...`; a per-device
unit reads `/var/lib/hal/calibration/robots/hal_follower/<id>.json` instead.

> **Provisioning order:** ideally place `<id>.json` in the persistent dir first, then
> set `HAL_DEVICE_ID` and restart HAL. If the order is reversed (env set, file not yet
> there), the arm does **not** start uncalibrated — it falls back to the repo `hal.json`
> and logs a warning; once the per-device file is dropped in and HAL restarts, it picks
> up the unit's own calibration.
>
The runtime fallback only affects **reads**. `hal.calibrate` pins the write target
explicitly (see below), so calibrating on-device with a per-device id always writes to
the persistent dir, even on the first calibration — no clobbering the shared repo file.

Each JSON contains per-servo values:

| Field | Description |
|-------|-------------|
| `id` | Servo ID (1–5) |
| `drive_mode` | Direction (0 = normal) |
| `homing_offset` | Encoder offset for center position |
| `range_min` | Minimum encoder value (physical limit) |
| `range_max` | Maximum encoder value (physical limit) |

## Servos

| ID | Name | Function |
|----|------|----------|
| 1 | `base_yaw` | Left/right rotation |
| 2 | `base_pitch` | Forward/backward tilt |
| 3 | `elbow_pitch` | Elbow bend |
| 4 | `wrist_roll` | Wrist rotation |
| 5 | `wrist_pitch` | Wrist tilt |

## Deploy to a device

No copy-to-`~/.cache` step is needed: the calibration ships with the repo checkout
under `/opt/hal/calibration/...` and is read directly. After deploying new code or a
new calibration, restart the HAL service:

```bash
sudo systemctl restart hal
```

## Run fresh calibration on a Pi

Use this when the arm hardware changes (e.g. after replacing servos), and as the
per-unit provisioning step for a new device. Pass the `id` the device runs with:

- `--id hal` → writes the repo file `hal/calibration/robots/hal_follower/hal.json`.
- `--id <device>` (e.g. `lamp-abcd`) → writes **straight to**
  `/var/lib/hal/calibration/robots/hal_follower/<id>.json`.

For a per-device id the file lands in its final persistent location directly — **no
separate copy step**, and it survives OTA. Provisioning a new unit is therefore: set
`HAL_DEVICE_ID=<id>`, run calibrate on the lamp, restart HAL.

```bash
# Follower only
sudo /opt/hal/.venv/bin/python3 -m hal.calibrate \
  --id hal --port /dev/ttyACM0 --follower-only

# Leader only
sudo /opt/hal/.venv/bin/python3 -m hal.calibrate \
  --id hal --port /dev/ttyACM0 --leader-only

# Both follower and leader
sudo /opt/hal/.venv/bin/python3 -m hal.calibrate \
  --id hal --port /dev/ttyACM0
```

### Calibration steps (interactive)

1. **Torque is disabled** — servos go limp so you can move them by hand.
2. **Move all joints to the middle** of their range of motion, then press ENTER. This sets the homing offset.
3. **Move each joint through its full range** (min to max). The script records encoder positions. Press ENTER when done.
4. Calibration is saved to `hal/calibration/robots/hal_follower/hal.json`.

### After calibration

- For the default `hal` id: commit the updated file(s) under `hal/calibration/` so a
  fresh checkout picks them up. For a per-device id the file stays on the device under
  `/var/lib/hal/...` — do **not** commit it (each unit's calibration is its own).
- Restart the HAL service: `sudo systemctl restart hal` (or reboot).
