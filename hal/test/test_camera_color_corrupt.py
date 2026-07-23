"""Tests for the camera ISP color-corruption detector.

The detector flags the wedged-ISP failure mode where frames keep changing
but chroma is garbage: posterized green+magenta, red+magenta, or a broad
magenta/deep-magenta palette.
Thresholds were calibrated against live corrupt specimens and synthetic
fixtures mirror their measured hue distributions.
"""
import numpy as np
import pytest

cv2 = pytest.importorskip("cv2")

from hal.drivers.camera.video_capture_device import LocalVideoCaptureDevice


def _bgr_from_hsv(h: int, s: int, v: int) -> tuple[int, int, int]:
    px = np.array([[[h, s, v]]], dtype=np.uint8)
    b, g, r = cv2.cvtColor(px, cv2.COLOR_HSV2BGR)[0, 0]
    return int(b), int(g), int(r)


def _frame(fill_bgr: tuple[int, int, int]) -> np.ndarray:
    frame = np.zeros((23, 40, 3), dtype=np.uint8)
    frame[:] = fill_bgr
    return frame


def test_corrupt_green_plus_magenta_detected():
    # Mirror the live specimen: ~19% saturated acid green, ~1.5% magenta,
    # rest a desaturated office-like gray.
    frame = _frame((120, 128, 125))
    frame[0:5, :] = _bgr_from_hsv(60, 220, 180)  # acid green rows (~22%)
    frame[6, 0:1] = _bgr_from_hsv(150, 140, 180)  # magenta speck
    frame[7, 0:1] = _bgr_from_hsv(155, 140, 170)
    frame[8, 0:12] = _bgr_from_hsv(150, 140, 180)  # magenta patch (~1.5%)
    assert LocalVideoCaptureDevice._looks_color_corrupt(frame)


def test_corrupt_red_plus_magenta_detected():
    # Live red-corrupt frames measured red=0.23-0.42 and magenta=0.10-0.35.
    frame = _frame((120, 128, 125))
    frame[0:7, :] = _bgr_from_hsv(2, 220, 180)  # red ~30%
    frame[8:11, :] = _bgr_from_hsv(150, 180, 180)  # magenta ~13%
    assert LocalVideoCaptureDevice._looks_color_corrupt(frame)


def test_corrupt_red_with_sparse_magenta_detected():
    # The first live failure under continuous streaming had red=0.258 and
    # magenta=0.016. It must not be missed while waiting for the later,
    # stronger magenta palette to form.
    frame = _frame((120, 128, 125))
    frame[0:6, :] = _bgr_from_hsv(2, 220, 180)  # red ~26%
    frame[7, 0:15] = _bgr_from_hsv(150, 180, 180)  # magenta ~1.6%
    assert LocalVideoCaptureDevice._looks_color_corrupt(frame)


def test_corrupt_magenta_deep_magenta_palette_detected():
    # The live blue-LED specimen was magenta=0.24 / deep-magenta=0.16, with
    # almost no canonical red or green. Mirror that two-part palette.
    frame = _frame((120, 128, 125))
    frame[0:4, :] = _bgr_from_hsv(165, 220, 180)  # deep-magenta ~17%
    frame[5:7, :] = _bgr_from_hsv(145, 180, 180)  # magenta companion ~9%
    assert LocalVideoCaptureDevice._looks_color_corrupt(frame)


def test_clean_desaturated_scene_not_detected():
    assert not LocalVideoCaptureDevice._looks_color_corrupt(_frame((120, 128, 125)))


def test_single_hue_green_wall_not_detected():
    # A green wall / plant / LED spill saturates ONE hue family only — the
    # complementary-magenta requirement must keep this from triggering.
    assert not LocalVideoCaptureDevice._looks_color_corrupt(
        _frame(_bgr_from_hsv(60, 220, 180))
    )


def test_single_hue_magenta_flood_not_detected():
    assert not LocalVideoCaptureDevice._looks_color_corrupt(
        _frame(_bgr_from_hsv(150, 220, 180))
    )


def test_single_hue_red_led_spill_not_detected():
    # A real red LED can dominate the camera view; without magenta it must
    # not trigger an ISP recovery.
    assert not LocalVideoCaptureDevice._looks_color_corrupt(
        _frame(_bgr_from_hsv(2, 220, 180))
    )


def test_dark_frame_not_detected():
    # Near-black pixels are excluded by the value floor regardless of hue.
    assert not LocalVideoCaptureDevice._looks_color_corrupt(
        _frame(_bgr_from_hsv(60, 255, 20))
    )
