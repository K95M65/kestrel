# Servo Calibration

The arm uses 5x Feetech STS3215 servos controlled via the `lerobot` library. Calibration
maps raw encoder positions to joint positions. **It lives in two places, and this split is
the single most important thing to understand here.**

## Where calibration actually lives

A calibration JSON holds five values per servo, but they do **not** share a fate at runtime:

| Field | Where it takes effect | Applied by |
|-------|----------------------|------------|
| `homing_offset` | **The servo's own EEPROM** (STS3215 control-table address 31) | The **servo, in hardware**: `Present_Position = Actual_Position − Homing_Offset` |
| `range_min` / `range_max` | Read from the **JSON file** on every start | lerobot **software** normalization (`_normalize` / `_unnormalize`) |
| `id`, `drive_mode` | JSON | software |

`homing_offset` is the per-unit "tare" that compensates how the servo horn sits on its
spline. **The runtime never writes it to the motors** — the services call
`connect(calibrate=False)`, and `write_calibration()` is only reachable from
`hal.calibrate` / `hal.apply_calibration`. So the number in the JSON has **no effect on
the servo's zero**; only the copy inside the servo's EEPROM does.

Consequence, and the reason this page exists:

> A freshly assembled unit ships with factory `Homing_Offset = 0`. It will move to the
> wrong poses **no matter which JSON it is pointed at**, until the calibration is pushed
> into the motors once. Copying JSON files around cannot fix it.

EEPROM is non-volatile, so the push survives power-off, reflash and OTA. It only has to
happen once per physical unit (or after replacing a servo / remounting a horn).

## Provisioning a unit — the normal path

Push the shared reference calibration into the motors. **No jogging the arm by hand, no
per-device JSON, no `.env` edit.**

```bash
sudo systemctl stop hal          # the running service holds /dev/ttyACM0
cd /opt/hal
sudo ./.venv/bin/python3 -m hal.apply_calibration --port /dev/ttyACM0
sudo systemctl start hal
```

`hal/apply_calibration.py` is non-interactive. It prints the servos' current values next
to the file's, disables torque first (which clears the `Lock` register — the servo rejects
EEPROM writes while locked), writes, then **reads back and verifies**. Success looks like:

```
=== AFTER (read back from the servos) ===
motor                         homing             range_min             range_max
base_yaw                        1928                   828                  3255
...
5/5 motors already match the file
```

Anything other than `5/5` means the write did not land — do not ship the unit.

Useful flags:

- `--dry-run` — report what would change, write nothing.
- `--port` — default `/dev/ttyACM0`; check `ls /dev/ttyACM*` if it differs.
- `--id <device>` — resolve the file for a per-device id instead of the shared one.
- `--file <path>` — write an explicit JSON (used to restore a backup).

To back up a unit's current EEPROM before overwriting it, run with `--dry-run` first and
keep the printed values, or dump them via `bus.read_calibration()`.

## Do not hand-calibrate to "fix" a unit

`hal.calibrate` with the `c` option re-records `homing_offset` **and** `range_min/max`
from scratch. That produces a *valid but different* frame for that one unit — and the
shared animation library (`hal/recordings/*.csv`) stores **normalized** values that were
authored in the reference frame. Playback maps them back through whatever `range_min/max`
is in force, so a unit with its own ranges plays every canned animation at the wrong pose.

**For a shared animation library, every unit must share one frame.** That means: the same
`range_min/max` (the repo `hal.json`) and a `homing_offset` that anchors each unit to the
same physical zero. `apply_calibration` gives exactly that.

Only reach for a real re-calibration when the hardware itself changed — a replaced servo,
or a horn remounted a tooth off (~14° per spline tooth, visible by eye).

> **Device-verified 2026-07-27 (lamp-ac82).** Before the push, the unit's EEPROM held
> `base_pitch homing = −556` against the file's `−381`, i.e. the arm sat **15° off** the
> reference frame, and its EEPROM travel limits silently clipped `base_yaw` by ~13° at
> each end. After `apply_calibration`, read-back showed 5/5 matching and the arm visibly
> tilted forward by the predicted ~15°. Three units had been running in three slightly
> different frames — "working" only because lamp motion tolerates a few degrees.

## Which JSON the runtime reads

lerobot loads `calibration_dir / f"{id}.json"`, where `id` is `HAL_DEVICE_ID`
(`hal/config.py`, default `"hal"`; `hal/.env.example` sets `HAL_DEVICE_ID=hal`).
`config_hal_follower.py.__post_init__` resolves the directory:

- **`id = "hal"`** (default / unset) → the version-controlled repo file. This is the
  **shared reference frame** the animation library is authored in, and the normal choice
  for the whole fleet:
  - `hal/follower/config_hal_follower.py` → `hal/calibration/robots/hal_follower`
  - `hal/leader/config_hal_leader.py` → `hal/calibration/teleoperators/hal_leader`
- **Per-device `id`** (e.g. `lamp-abcd`) → `/var/lib/hal/calibration/robots/hal_follower/<id>.json`
  (override the base with `HAL_CALIBRATION_DIR`). Outside the OTA tree (`/opt/hal` is
  overwritten every update), so it survives updates. **If that file is missing, the config
  logs a warning and falls back to the repo `hal.json`** — the arm never starts with no
  calibration registered.

The per-device path exists for the exception case (a unit that genuinely needs its own
numbers). It is **not** the normal provisioning route — see the warning above.

```
hal/calibration/
├── robots/hal_follower/hal.json          # shared reference frame
└── teleoperators/hal_leader/hal.json
```

On a device the repo copy ships under `/opt/hal/calibration/...` via OTA.

> **Why override lerobot's default at all:** lerobot's per-user path
> (`~/.cache/huggingface/lerobot/calibration`) breaks when the service user differs from
> the user that ran calibration. `hal.service` runs as **root**, so it would look under
> `/root/.cache/...` and miss a calibration saved elsewhere — surfacing as
> `FeetechMotorsBus(...) has no calibration registered`. Both dirs above are independent
> of the service user.

Startup log line (greppable) confirms what was loaded:

```bash
journalctl -u hal -b | grep -i calib
# calibration: loading id=hal from /opt/hal/calibration/robots/hal_follower/hal.json (exists=True)
```

Note this only tells you which **file** was read — it says nothing about what is in the
servos' EEPROM. Use `apply_calibration --dry-run` to compare the two.

## Servos

| ID | Name | Function |
|----|------|----------|
| 1 | `base_yaw` | Left/right rotation |
| 2 | `base_pitch` | Forward/backward tilt |
| 3 | `elbow_pitch` | Elbow bend |
| 4 | `wrist_roll` | Wrist rotation |
| 5 | `wrist_pitch` | Wrist tilt |

## Re-recording a calibration (hardware changed)

Use this only when the mechanics changed. It rewrites both the frame and the file, so
afterwards the unit no longer matches the shared animation frame unless you re-record the
reference for the whole fleet.

```bash
sudo systemctl stop hal
cd /opt/hal
sudo ./.venv/bin/python3 -m hal.calibrate --id hal --port /dev/ttyACM0 --follower-only
sudo systemctl start hal
```

`--id` names the output file (the CLI reads this argument, **not** `.env`): `--id hal`
writes the repo file; `--id <device>` writes straight to the persistent per-device dir.

Interactive steps:

1. Press `c` to run a fresh calibration (ENTER instead **writes the existing file to the
   motors** — the same thing `apply_calibration` does, but unverified and without
   unlocking the EEPROM first).
2. Torque is disabled — the servos go limp so you can move them by hand.
3. Move all joints to the middle of their range, press ENTER. This sets `homing_offset`.
4. Move each joint through its **full** range, min to max, press ENTER. An incomplete
   sweep records too-narrow ranges, which compresses every animation.
5. The result is saved to `<calibration_dir>/<id>.json` and written to the motors.

Afterwards: commit the updated `hal/calibration/...` file if you re-recorded the shared
reference (`--id hal`); a per-device file stays on the device and is not committed.
Restart HAL: `sudo systemctl restart hal`.
