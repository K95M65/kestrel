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
separate copy step**, and it survives OTA.

> **Stop HAL first.** The running `hal` service holds the servo serial port; two
> processes can't own `/dev/ttyACM0` at once. Stop it before calibrating, start it after.

### Provision a new fleet device (per-unit)

Run on the lamp itself (same image flashed to every unit). In the commands below,
replace **`lamp-abcd`** with this unit's id everywhere — match the hostname so it's easy
to track.

> **Two different things use the id — don't confuse them:**
> - **`--id lamp-abcd` on the calibrate command** = what *names the output file*. The
>   CLI reads this argument directly, **not** `.env`. So calibrate writes
>   `lamp-abcd.json` no matter what `HAL_DEVICE_ID` currently says.
> - **`HAL_DEVICE_ID=lamp-abcd` in `.env`** = what the *runtime reads at boot*. This is
>   only needed **after** the file exists.
>
> Therefore the order is: **calibrate first** (creates the file via `--id`), **then set
> `.env` and restart** (points the runtime at it). Do **not** set `.env` and restart
> before the file exists — the runtime would just fall back to the repo `hal.json`.

```bash
# 1) Release the servo serial port (the running service holds it)
sudo systemctl stop hal

# 2) Calibrate this unit's follower arm. --id names the output file, so this writes
#    /var/lib/hal/calibration/robots/hal_follower/lamp-abcd.json (persistent, survives OTA).
#    No .env change is needed for this step.
cd /opt/hal
sudo ./.venv/bin/python3 -m hal.calibrate --id lamp-abcd --port /dev/ttyACM0 --follower-only

# 3) NOW that the file exists, point the runtime at it and bring HAL back up.
#    Append the line, or edit it in place if HAL_DEVICE_ID already exists in .env.
echo 'HAL_DEVICE_ID=lamp-abcd' | sudo tee -a /opt/hal/.env
sudo systemctl start hal
```

- **Serial port:** if it isn't `ACM0`, check `ls /dev/ttyACM*` and adjust `--port`.
- **Only the follower** (5 servos) is on a shipped lamp; the leader is the teleop arm.
- **Custom store:** prepend `HAL_CALIBRATION_DIR=/other/path` to step 2 to change the dir.

Verify:

```bash
cat /var/lib/hal/calibration/robots/hal_follower/lamp-abcd.json   # 5 servos present
journalctl -u hal -n 30 | grep -i calib                          # NOT "falling back to repo hal.json"
```

A "falling back to repo hal.json" line means the `HAL_DEVICE_ID` in `.env` doesn't match
a file that exists in the persistent dir — recheck that step 2 wrote `lamp-abcd.json` and
that step 3 used the same id.

### Default `hal` calibration (dev / shared repo file)

Use `--id hal` to (re)record the version-controlled repo file — e.g. on a dev bench, or
to refresh the shared fallback:

```bash
sudo systemctl stop hal
cd /opt/hal
sudo ./.venv/bin/python3 -m hal.calibrate --id hal --port /dev/ttyACM0 --follower-only  # add --leader-only / omit for both
sudo systemctl start hal
```

### Calibration steps (interactive)

Both flows prompt the same way:

1. Press `c` at the prompt to run calibration (ENTER reuses the existing file).
2. **Torque is disabled** — servos go limp so you can move them by hand.
3. **Move all joints to the middle** of their range of motion, then press ENTER. This sets the homing offset.
4. **Move each joint through its full range** (min to max). The script records encoder positions. Press ENTER when done.
5. Calibration is saved to `<calibration_dir>/<id>.json` — the persistent dir for a
   per-device id, or the repo file for `--id hal`.

### After calibration

- **Per-device id:** the file stays on the device under `/var/lib/hal/...` — do **not**
  commit it (each unit's calibration is its own).
- **`hal` id:** commit the updated file under `hal/calibration/` so a fresh checkout /
  the shared fallback picks it up.
- Restart the HAL service if you haven't: `sudo systemctl restart hal` (or reboot).
