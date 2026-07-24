#!/usr/bin/env python

# Copyright 2025 The HuggingFace Inc. team. All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import os
from dataclasses import dataclass, field
from pathlib import Path

from lerobot.cameras import CameraConfig

from lerobot.robots import RobotConfig

# Repo-local, version-controlled calibration dir (hal/calibration/robots/hal_follower)
# instead of lerobot's per-user default (~/.cache/huggingface/lerobot/calibration), which
# breaks when the service user differs (e.g. hal.service runs as root, not orangepi).
# lerobot loads `calibration_dir / f"{id}.json"` (id = HAL_DEVICE_ID, default "hal").
CALIBRATION_DIR = Path(__file__).resolve().parent.parent / "calibration" / "robots" / "hal_follower"

# Per-device calibration dir for the fleet. homing_offset / range_min / range_max are
# unique to each physically-assembled unit (servo-horn spline mounting varies per build),
# so the repo file only fits the one unit it was recorded on. Each device carries its own
# <HAL_DEVICE_ID>.json here, provisioned by the hardware team. Lives OUTSIDE the OTA tree
# (/opt/hal is overwritten every update) so calibration survives updates. Override the base
# with HAL_CALIBRATION_DIR.
PERSISTENT_CALIBRATION_DIR = Path(
    os.environ.get("HAL_CALIBRATION_DIR", "/var/lib/hal/calibration/robots/hal_follower")
)


@RobotConfig.register_subclass("hal_follower")
@dataclass
class LeLampFollowerConfig(RobotConfig):
    # Port to connect to the arm
    port: str

    disable_torque_on_disconnect: bool = True

    # `max_relative_target` limits the magnitude of the relative positional target vector for safety purposes.
    # Set this to a positive scalar to have the same value for all motors, or a list that is the same length as
    # the number of motors in your follower arms.
    max_relative_target: int | None = None

    # cameras
    cameras: dict[str, CameraConfig] = field(default_factory=dict)

    # Set to `True` for backward compatibility with previous policies/dataset
    use_degrees: bool = False

    def __post_init__(self):
        super().__post_init__()
        # Resolve the calibration dir unless a caller pinned one explicitly.
        # Default id "hal" (HAL_DEVICE_ID unset) keeps reading the version-controlled
        # repo file. A per-device id (e.g. "lamp-abcd") reads its own file from the
        # persistent fleet dir, where the hardware team drops <id>.json per unit.
        if self.calibration_dir is None:
            if self.id and self.id != "hal":
                self.calibration_dir = PERSISTENT_CALIBRATION_DIR
            else:
                self.calibration_dir = CALIBRATION_DIR
