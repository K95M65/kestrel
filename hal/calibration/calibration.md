# Servo Calibration

The arm uses 5x Feetech STS3215 servos controlled via the `lerobot` library. Calibration
maps raw encoder positions to joint positions. **It lives in two places, and that split
explains everything else on this page.**

## Where calibration actually lives

A calibration JSON holds five values per servo, but they do **not** share a fate at runtime:

| Field | Where it takes effect | Applied by | Scope |
|-------|----------------------|------------|-------|
| `homing_offset` | **The servo's own EEPROM** (STS3215 control-table address 31) | The **servo, in hardware**: `Present_Position = Actual_Position − Homing_Offset` | **Per unit** |
| `range_min` / `range_max` | Read from the **JSON file** on every start | lerobot **software** normalization (`_normalize` / `_unnormalize`) | Shared |
| `id`, `drive_mode` | JSON | software | Shared |

`homing_offset` is the "tare" that compensates how the servo horn happens to sit on its
spline — the horn mounts on discrete teeth (~14° per tooth), so it is **specific to each
physically-assembled arm**. One arm's value does not fit another.

**The runtime never writes it to the motors.** The services call `connect(calibrate=False)`,
and `write_calibration()` is only reachable from `hal.calibrate` / `hal.apply_calibration`.
So the number in a JSON file has **no effect on the servo's zero** — only the copy inside
the servo's EEPROM does, and a JSON on the SD card cannot change it.

EEPROM is non-volatile: it survives power-off, reboot, SD reflash, OTA and a device factory
reset. Calibration therefore has to be done **once per physical unit**, and only needs
redoing if a servo is replaced or a horn remounted.

## Provisioning a unit — hand calibration

Each new arm is calibrated by hand. There is no shortcut: `homing_offset` is per-unit, so
nothing can be copied in from another arm.

```bash
sudo systemctl stop hal          # the running service holds /dev/ttyACM0
cd /opt/hal
sudo ./.venv/bin/python3 -m hal.calibrate --id lamp-abcd --port /dev/ttyACM0 --follower-only
#   press 'c', then follow the prompts (below)
sudo systemctl start hal
```

Replace `lamp-abcd` with the unit's id — matching the hostname keeps it easy to trace.

Interactive steps:

1. Press `c` to run a fresh calibration (ENTER instead just writes the existing file to the
   motors — see `apply_calibration` below, which does the same thing but verified).
2. Torque is disabled — the servos go limp so you can move them by hand.
3. Move all joints to the middle of their range, press ENTER. **This sets
   `homing_offset`** and writes it into the servos' EEPROM immediately.
4. Move each joint through its **full** range, min to max, press ENTER. An incomplete
   sweep records too-narrow ranges.
5. The result is saved to `/var/lib/hal/calibration/robots/hal_follower/lamp-abcd.json`
   and written to the motors.

That is the whole process. **Do not set `HAL_DEVICE_ID`** — see the next section for why.

### Why this works without any `.env` change

After the steps above the unit runs on:

- **its own `homing_offset`**, living in the servos' EEPROM — this is the part that must be
  per-unit, and hand calibration has already put it there;
- **the shared `range_min/max`**, because with `HAL_DEVICE_ID` unset the runtime loads the
  repo `hal.json` (see below), not the `lamp-abcd.json` that was just written.

Shared ranges are what you want: `hal/recordings/*.csv` stores **normalized** values, so
every unit playing the same animation library has to normalize against the same range. The
per-device file that calibration produced simply sits unused — treat it as a backup of that
unit's numbers.

## `apply_calibration` — restoring a unit, not provisioning one

`hal/apply_calibration.py` writes a calibration file into the motors' EEPROM
non-interactively and verifies the write by reading it back:

```bash
sudo systemctl stop hal
cd /opt/hal
sudo ./.venv/bin/python3 -m hal.apply_calibration --file /var/lib/hal/calibration/robots/hal_follower/lamp-abcd.json
sudo systemctl start hal
```

Use it to **put a unit's own numbers back** — after a servo swap, or if someone recalibrated
and the result was worse. To inspect the servos without writing anything:

```bash
sudo ./.venv/bin/python3 -m hal.apply_calibration --dry-run --id hal
```

`--file` or `--id` is **required**; there is no default source. Pushing the shared
`hal.json` onto an arm other than the one it was recorded from replaces that arm's zero
with the reference unit's, and the servos move to visibly wrong poses — so the tool refuses
to guess. Check the `source :` line it prints before letting it write.
>
> The tool does not back up what it overwrites — the `BEFORE` table it prints is the only
> record of the previous values. Keep it.

## Which JSON the runtime reads

lerobot loads `calibration_dir / f"{id}.json"`, where `id` is `HAL_DEVICE_ID`
(`hal/config.py`, default `"hal"`; `hal/.env.example` sets `HAL_DEVICE_ID=hal`).
`config_hal_follower.py.__post_init__` resolves the directory:

- **`id = "hal"`** (default / unset — **the normal case**) → the version-controlled repo
  file, i.e. the shared ranges the animation library is authored against:
  - `hal/follower/config_hal_follower.py` → `hal/calibration/robots/hal_follower`
  - `hal/leader/config_hal_leader.py` → `hal/calibration/teleoperators/hal_leader`
- **Per-device `id`** (e.g. `lamp-abcd`) → `/var/lib/hal/calibration/robots/hal_follower/<id>.json`
  (override the base with `HAL_CALIBRATION_DIR`). Outside the OTA tree (`/opt/hal` is
  overwritten every update), so it survives updates. If that file is missing, the config
  logs a warning and falls back to the repo `hal.json`.

Setting `HAL_DEVICE_ID` makes the unit normalize against **its own** recorded ranges instead
of the shared ones. That is a deliberate choice, not the default — only reach for it if a
unit's travel really differs enough to matter, and accept that its animations will land
slightly differently from the rest of the fleet.

```
hal/calibration/
├── robots/hal_follower/hal.json          # shared ranges + the reference unit's homing
└── teleoperators/hal_leader/hal.json
```

> **Why override lerobot's default at all:** lerobot's per-user path
> (`~/.cache/huggingface/lerobot/calibration`) breaks when the service user differs from the
> user that ran calibration. `hal.service` runs as **root**, so it would look under
> `/root/.cache/...` and miss a calibration saved elsewhere — surfacing as
> `FeetechMotorsBus(...) has no calibration registered`. Both dirs above are independent of
> the service user.

Startup log line (greppable) shows which **file** was loaded — it says nothing about what is
in the servos' EEPROM:

```bash
journalctl -u hal -b | grep -i calib
# calibration: loading id=hal from /opt/hal/calibration/robots/hal_follower/hal.json (exists=True)
```

To see the EEPROM side, run `apply_calibration --dry-run --id hal` and read its `BEFORE`
table.

## Servos

| ID | Name | Function |
|----|------|----------|
| 1 | `base_yaw` | Left/right rotation |
| 2 | `base_pitch` | Forward/backward tilt |
| 3 | `elbow_pitch` | Elbow bend |
| 4 | `wrist_roll` | Wrist rotation |
| 5 | `wrist_pitch` | Wrist tilt |

## What can and cannot erase a calibration

**Cannot** (verified): power off, reboot, reflashing the SD card, OTA updates, or the device
factory reset — the factory-reset path only calls `release_servos()` (torque off), it never
touches the calibration registers.

**Can**: `hal.calibrate` with `c` (it calls `reset_calibration()` internally before
recording new values — the only reset path in the codebase), `hal.calibrate` with ENTER,
`hal.apply_calibration`, or physically replacing a servo
(a new servo arrives with factory `Homing_Offset = 0`).

## Field notes (2026-07-27, lamp-ac82)

The reference unit's EEPROM held `base_pitch homing = −556` while the committed `hal.json`
said `−381` — a 15.4° gap nobody had noticed, because nothing reads that field back from the
file. `hal.json` had been recorded on 2026-06-10 (commit `ede2e350`, new servos fitted), the
unit was hand-calibrated again afterwards without committing the result, and the animation
library was re-recorded on 06-29 / 07-06 — i.e. against the EEPROM state, not the file. The
file's `homing_offset` was simply stale; it was corrected to the unit's live values (ranges
untouched).

Two lessons:

- `homing_offset` in a file is effectively **write-only** from the runtime's point of view.
  It can drift from the hardware indefinitely with no symptom. Never assume the file
  reflects the servos — check with `apply_calibration --dry-run --id hal`.
- Pushing that same file to a *different* arm was tried and produced visibly wrong poses,
  which is what confirmed `homing_offset` cannot be shared between units.
