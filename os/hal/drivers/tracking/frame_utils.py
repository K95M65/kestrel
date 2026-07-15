"""Frame downscale + bbox coordinate mapping.

The whole vision pipeline (ViT tracker + every detector) runs on a frame
downscaled to VISION_MAX_WIDTH; every bbox is mapped back to ORIGINAL camera
coordinates before any servo/PID math, so pixel-tuned constants never need
re-tuning when the downscale factor changes.
"""

from typing import Tuple

import cv2
import numpy as np
import numpy.typing as npt

from hal.drivers.tracking.constants import VISION_MAX_WIDTH


def downscale(frame: npt.NDArray[np.uint8]) -> Tuple[npt.NDArray[np.uint8], float]:
    """Return (small_frame, scale) with scale = small_w / orig_w (≤ 1.0).

    No-op (returns the frame and scale 1.0) when downscale is disabled or the
    frame is already within VISION_MAX_WIDTH. INTER_AREA is the correct
    interpolation for shrinking (avoids aliasing that would jitter the bbox).
    """
    if not VISION_MAX_WIDTH:
        return frame, 1.0
    h, w = frame.shape[:2]
    if w <= VISION_MAX_WIDTH:
        return frame, 1.0
    scale = VISION_MAX_WIDTH / float(w)
    small = cv2.resize(frame, (VISION_MAX_WIDTH, max(1, int(round(h * scale)))),
                       interpolation=cv2.INTER_AREA)
    return small, scale


def scale_bbox(bbox: Tuple[int, int, int, int], factor: float) -> Tuple[int, int, int, int]:
    """Scale a bbox by `factor` (use scale to go orig→small, 1/scale for small→orig)."""
    if factor == 1.0:
        return tuple(int(v) for v in bbox)
    return (int(round(bbox[0] * factor)), int(round(bbox[1] * factor)),
            max(1, int(round(bbox[2] * factor))), max(1, int(round(bbox[3] * factor))))
