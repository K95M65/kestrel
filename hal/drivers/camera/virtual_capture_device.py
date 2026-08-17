"""Synthetic camera backend for the laptop simulation mode."""

from __future__ import annotations

import threading
import time

import cv2
import numpy as np

from .models import VideoCaptureDeviceInfo, VideoCaptureDeviceResponse
from .video_capture_device import VideoCaptureDeviceBase


class VirtualVideoCaptureDevice(VideoCaptureDeviceBase):
    """A deterministic calibration scene with the production camera surface."""

    runable = True
    requires_v4l2_index = False

    def __init__(self, device_info: VideoCaptureDeviceInfo, name: str | None = None):
        super().__init__(device_info, name)
        self.actual_width = device_info.max_width or 640
        self.actual_height = device_info.max_height or 480
        self.actual_fps = float(device_info.fps or 15)
        self.zoom = 1.0
        self._lock = threading.Lock()
        self._last_response: VideoCaptureDeviceResponse | None = None
        self._last_frame_ts = 0.0

    def _frame(self) -> np.ndarray:
        height, width = self.actual_height, self.actual_width
        frame = np.full((height, width, 3), (24, 18, 12), dtype=np.uint8)
        center = (width // 2, height // 2)
        cv2.circle(frame, center, min(width, height) // 5, (70, 150, 255), -1)
        cv2.circle(frame, center, min(width, height) // 8, (32, 85, 180), -1)
        for x in range(0, width, 64):
            cv2.line(frame, (x, 0), (x, height), (60, 60, 45), 1)
        for y in range(0, height, 64):
            cv2.line(frame, (0, y), (width, y), (60, 60, 45), 1)
        cv2.putText(frame, "Autonomous Lamp simulation", (20, 36),
                    cv2.FONT_HERSHEY_SIMPLEX, 0.65, (230, 230, 230), 2)
        cv2.putText(frame, "virtual camera", (20, height - 22),
                    cv2.FONT_HERSHEY_SIMPLEX, 0.52, (220, 190, 80), 1)
        return frame

    def start(self) -> None:
        with self._lock:
            self.running = True
            self._last_response = VideoCaptureDeviceResponse(
                frame=self._frame(),
                frame_description="Synthetic Lamp simulator calibration scene",
            )
            self._last_frame_ts = time.monotonic()

    def stop(self) -> None:
        with self._lock:
            self.running = False

    @property
    def last_frame(self):
        with self._lock:
            if not self._last_response or self._last_response.frame is None:
                return None
            return self._last_response.frame.copy()

    @property
    def last_frame_ts(self) -> float:
        return self._last_frame_ts

    @property
    def last_response(self):
        with self._lock:
            return self._last_response.model_copy(deep=True) if self._last_response else None

    def capture(self, need_description: bool = False):
        if not self.running:
            raise RuntimeError("VirtualVideoCaptureDevice has not started")
        return self.last_response

    def acquire_consumer(self) -> None:
        pass

    def release_consumer(self) -> None:
        pass
