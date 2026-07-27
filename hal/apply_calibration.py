#!/usr/bin/env python
"""Push a calibration file into the servos' EEPROM — non-interactive.

Why this tool exists
--------------------
`homing_offset` does NOT live in the JSON file at runtime. It lives in each servo's
EEPROM (STS3215 control table address 31) and the servo applies it in hardware:

    Present_Position = Actual_Position - Homing_Offset

The runtime never writes it: the services call `connect(calibrate=False)`, so
`write_calibration()` is only reachable from the interactive `hal.calibrate` flow. A JSON
file sitting on the SD card therefore has NO effect on the servo's zero — only
`range_min`/`range_max` are read from it (software normalization).

Consequence: a freshly assembled unit ships with factory `Homing_Offset = 0` and moves to
the wrong poses no matter which JSON it is pointed at. It must be pushed once.

What this does
--------------
Writes `homing_offset` + `range_min`/`range_max` from a calibration file into the motors,
then reads them back to prove the write landed. Compared to `hal.calibrate`:

  * no interactive prompt and no jogging the arm by hand,
  * torque is disabled first — that also clears the `Lock` register, which the servo
    requires before it will accept EEPROM writes (`hal.calibrate`'s ENTER branch skips
    this, so its write can be silently rejected),
  * the result is verified instead of assumed.

Only servo configuration registers are touched. This never commands a movement.

Usage (on the device, HAL stopped so the serial port is free):

    sudo systemctl stop hal
    cd /opt/hal
    sudo ./.venv/bin/python3 -m hal.apply_calibration --port /dev/ttyACM0
    sudo systemctl start hal

Add `--dry-run` to only report what would change.
"""

import argparse
import sys
import traceback

import draccus
from lerobot.motors import MotorCalibration

from .follower import LeLampFollower, LeLampFollowerConfig

_ATTRS = ("homing_offset", "range_min", "range_max")


def _load_calibration_file(path: str) -> dict[str, MotorCalibration]:
    """Load a calibration JSON the same way lerobot's Robot._load_calibration does."""
    with open(path) as f, draccus.config_type("json"):
        return draccus.load(dict[str, MotorCalibration], f)


def _row_matches(current: MotorCalibration, target: MotorCalibration) -> bool:
    return all(getattr(current, a) == getattr(target, a) for a in _ATTRS)


def _print_comparison(current: dict[str, MotorCalibration], target: dict[str, MotorCalibration]) -> int:
    """Print servo-vs-file for every motor. Returns the number of motors already matching."""
    header = f"{'motor':<14}{'homing':>22}{'range_min':>22}{'range_max':>22}"
    print(header)
    print("-" * len(header))

    matching = 0
    for name, want in target.items():
        have = current.get(name)
        if have is None:
            print(f"{name:<14}{'!! motor not found on the bus':>66}")
            continue

        cells = []
        for attr in _ATTRS:
            servo_val, file_val = getattr(have, attr), getattr(want, attr)
            cells.append(f"{servo_val}" if servo_val == file_val else f"{servo_val} -> {file_val}")
        print(f"{name:<14}" + "".join(f"{c:>22}" for c in cells))

        if _row_matches(have, want):
            matching += 1

    print()
    print(f"{matching}/{len(target)} motors already match the file "
          f"(a bare number = already correct, 'a -> b' = would change)")
    return matching


def apply_calibration(port: str, calib_id: str, file_path: str | None, dry_run: bool) -> int:
    config = LeLampFollowerConfig(port=port, id=calib_id)
    robot = LeLampFollower(config)

    # --file wins; otherwise use whatever the config resolved for this id.
    if file_path:
        target = _load_calibration_file(file_path)
        source = file_path
    else:
        target = robot.calibration
        source = str(robot.calibration_fpath)

    if not target:
        print(f"ERROR: no calibration loaded from {source}", file=sys.stderr)
        print("Pass --file <path>, or use an --id whose <id>.json exists.", file=sys.stderr)
        return 1

    print(f"port   : {port}")
    print(f"source : {source}")
    print()

    bus = robot.bus
    try:
        bus.connect()
    except Exception as exc:
        print(f"ERROR: cannot open {port}: {exc}", file=sys.stderr)
        print("\nIs HAL still running and holding the port? Stop it first:", file=sys.stderr)
        print("    sudo systemctl stop hal", file=sys.stderr)
        return 1

    try:
        print("=== BEFORE (servo EEPROM vs file) ===")
        already_matching = _print_comparison(bus.read_calibration(), target)

        if dry_run:
            print("\n--dry-run: nothing was written.")
            return 0

        if already_matching == len(target):
            print("\nNothing to do — the servos already hold these values.")
            return 0

        # disable_torque() also writes Lock=0, which unlocks the EEPROM area. Without it the
        # servo can silently reject the writes below. It leaves the arm limp, which is the
        # same state HAL leaves it in when stopped.
        print("\nDisabling torque (unlocks the servo EEPROM; the arm goes limp)...")
        bus.disable_torque()

        print("Writing calibration to the motors...")
        bus.write_calibration(target)

        print()
        print("=== AFTER (read back from the servos) ===")
        verified = _print_comparison(bus.read_calibration(), target)

        if verified != len(target):
            print("\nFAILED: the servos did not accept every value.", file=sys.stderr)
            print("Power-cycle the arm and retry; if it persists, the EEPROM may be "
                  "write-locked or a motor is off the bus.", file=sys.stderr)
            return 1

        print("\nOK: all motors now hold the calibration. It survives power-off.")
        print("Start HAL again:  sudo systemctl start hal")
        return 0
    finally:
        if bus.is_connected:
            # Leave torque as-is; HAL configures it on the next start.
            bus.disconnect(disable_torque=False)


def main() -> int:
    parser = argparse.ArgumentParser(
        description="Write a calibration file into the servos' EEPROM (non-interactive).",
    )
    parser.add_argument(
        "--port", type=str, default="/dev/ttyACM0",
        help="Serial port of the servo bus (default: /dev/ttyACM0)",
    )
    parser.add_argument(
        "--id", type=str, default="hal", dest="calib_id",
        help="Calibration id to resolve the file from (default: 'hal' = the shared repo "
             "reference file). Use the device id to pick up its per-device file instead.",
    )
    parser.add_argument(
        "--file", type=str, default=None,
        help="Explicit calibration JSON to write, bypassing --id resolution.",
    )
    parser.add_argument(
        "--dry-run", action="store_true",
        help="Only report what would change; write nothing.",
    )
    args = parser.parse_args()

    try:
        return apply_calibration(args.port, args.calib_id, args.file, args.dry_run)
    except Exception:
        traceback.print_exc()
        return 1


if __name__ == "__main__":
    sys.exit(main())
