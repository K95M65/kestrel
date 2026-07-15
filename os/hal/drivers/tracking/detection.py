"""Object detection for tracking: local YOLOv8n (COCO), YuNet face detector,
and the remote open-vocab YOLOWorld fallback.

All detectors run on the downscaled frame (frame_utils) and return bboxes in
ORIGINAL camera coordinates as (x, y, w, h), or None.
"""

import base64
import json
import logging
import os
import threading
import time
from typing import Callable, Optional, Tuple

import cv2
import numpy as np
import numpy.typing as npt
import requests

import hal.config as config
from hal.config import (
    TRACKING_DETECT_LOCAL_ENABLED as _DETECT_LOCAL_ENABLED,
    TRACKING_FACE_DETECTOR_ENABLED as _FACE_DETECTOR_ENABLED,
    YUNET_CONFIDENCE_THRESHOLD as _YUNET_CONF,
)
from hal.drivers.sensing.crypto import CryptoSession, resolve_public_key
from hal.drivers.tracking import constants as C
from hal.drivers.tracking.frame_utils import downscale, scale_bbox

logger = logging.getLogger(__name__)

# Local YOLOv8n (COCO) — ~300ms/frame on Allwinner A523 CPU. Used by default
# when target maps to a COCO class. Falls back to remote API for open-vocab.
# Weights are checked into the repo next to this file so deploy is one rsync
# and the Pi never needs internet at boot to start tracking. Source:
# https://github.com/ultralytics/assets/releases/download/v8.3.0/yolov8n.pt
_LOCAL_MODEL_PATH = os.path.join(os.path.dirname(__file__), "models", "yolov8n.pt")
# Inference size for local YOLO. Kept at 320: on the Allwinner A523 CPU, 640
# pushed inference to 1.3–2.9s/call, so YOLO could not confirm within the
# trust window (yolo_age climbed past 20s) and the tracker sat in BLOAT-HOLD
# pointing the wrong way. 320 keeps detection ~260ms — fast enough to correct
# ViT drift every redetect. (Small/far objects that 320 misses are better
# handled by the remote open-vocab detector than by a slower local imgsz.)
_LOCAL_IMGSZ = 320

# YuNet face detector (OpenCV built-in). Lighter than InsightFace, ~30ms/frame on
# Pi, no extra dependency. Used for target='face' so we don't fall back to the
# remote YOLOWorld (~1.3s) for what's a very common tracking target.
_YUNET_MODEL_PATH = os.path.join(os.path.dirname(__file__), "models",
                                 "face_detection_yunet_2023mar.onnx")
# Aliases that route to the face detector instead of YOLO.
_FACE_TARGET_ALIASES = {"face", "human face", "khuôn mặt", "mặt"}

# Target label → COCO class index. Add aliases for natural Vietnamese/English usage.
_COCO_CLASSES = {
    # NOTE: "hand" / "face" intentionally NOT mapped — COCO has no hand/face class.
    # Mapping them to "person" caused bbox to lock onto whole body (25-78% frame),
    # triggering "object too close" stop. Let them fall through to YOLOWorld remote.
    "person": 0, "people": 0, "human": 0,
    "bicycle": 1, "car": 2, "motorcycle": 3, "airplane": 4, "bus": 5,
    "train": 6, "truck": 7, "boat": 8, "traffic light": 9, "fire hydrant": 10,
    "stop sign": 11, "parking meter": 12, "bench": 13,
    "bird": 14, "cat": 15, "dog": 16, "horse": 17, "sheep": 18, "cow": 19,
    "elephant": 20, "bear": 21, "zebra": 22, "giraffe": 23,
    "backpack": 24, "umbrella": 25, "handbag": 26, "tie": 27, "suitcase": 28,
    "frisbee": 29, "skis": 30, "snowboard": 31, "sports ball": 32, "ball": 32,
    "kite": 33, "baseball bat": 34, "baseball glove": 35, "skateboard": 36,
    "surfboard": 37, "tennis racket": 38, "bottle": 39, "wine glass": 40,
    "cup": 41, "fork": 42, "knife": 43, "spoon": 44, "bowl": 45,
    "banana": 46, "apple": 47, "sandwich": 48, "orange": 49, "broccoli": 50,
    "carrot": 51, "hot dog": 52, "pizza": 53, "donut": 54, "cake": 55,
    "chair": 56, "couch": 57, "potted plant": 58, "bed": 59, "dining table": 60,
    "toilet": 61, "tv": 62, "laptop": 63, "mouse": 64, "remote": 65,
    "keyboard": 66, "cell phone": 67, "phone": 67, "microwave": 68, "oven": 69,
    "toaster": 70, "sink": 71, "refrigerator": 72, "book": 73, "clock": 74,
    "vase": 75, "scissors": 76, "teddy bear": 77, "hair drier": 78, "toothbrush": 79,
}
# Remote API fallback (open vocabulary).
_DETECT_MODEL = "yoloworld"
_YOLO_ENDPOINT = f"/detect/{_DETECT_MODEL}"
_YOLO_TIMEOUT = 10.0

# Remote-fallback throttle. Local YOLOv8n@320 misses small/far objects (e.g. a
# cup across the room) that the remote open-vocab YOLOWorld can still find. On a
# local miss we fall back to remote — but remote is ~1.3s + network, so a target
# local genuinely can't see would fire remote on every redetect. Rate-limit it
# to at most one remote attempt per this interval (seconds). The very first
# detect (e.g. session start) is never throttled (timestamp starts at 0).
REMOTE_FALLBACK_MIN_INTERVAL = 2.0

# Singleton local YOLO model — loaded lazily on first detection.
_local_yolo = None
_local_yolo_lock = threading.Lock()

# Singleton YuNet face detector — same lazy pattern.
_yunet = None
_yunet_lock = threading.Lock()


def _get_local_yolo():
    """Lazy-load YOLOv8n model from the repo path. Thread-safe singleton."""
    global _local_yolo
    if _local_yolo is not None:
        return _local_yolo
    with _local_yolo_lock:
        if _local_yolo is not None:
            return _local_yolo
        if not os.path.exists(_LOCAL_MODEL_PATH):
            logger.error(
                "YOLO weights missing at %s — re-deploy from repo (file is checked in). "
                "Falling back to remote YOLOWorld until present.",
                _LOCAL_MODEL_PATH,
            )
            return None
        try:
            from ultralytics import YOLO
            logger.info("Loading local YOLO model from %s", _LOCAL_MODEL_PATH)
            t0 = time.perf_counter()
            _local_yolo = YOLO(_LOCAL_MODEL_PATH)
            # Warm-up inference to trigger model compile/cache.
            import numpy as _np
            _local_yolo(_np.zeros((480, 640, 3), dtype=_np.uint8),
                        verbose=False, imgsz=_LOCAL_IMGSZ)
            logger.info("Local YOLO loaded + warmed up in %.0fms", (time.perf_counter() - t0) * 1000)
        except Exception as e:
            logger.error("Local YOLO load failed: %s", e)
            _local_yolo = None
    return _local_yolo


def _get_yunet():
    """Lazy-load YuNet face detector. Thread-safe singleton.

    Input size is set per-call via setInputSize before detect(), so we can keep
    one shared detector across frames of different sizes.
    """
    global _yunet
    if _yunet is not None:
        return _yunet
    with _yunet_lock:
        if _yunet is not None:
            return _yunet
        if not os.path.exists(_YUNET_MODEL_PATH):
            logger.error("YuNet weights missing at %s — face detection disabled",
                         _YUNET_MODEL_PATH)
            return None
        try:
            t0 = time.perf_counter()
            _yunet = cv2.FaceDetectorYN.create(
                _YUNET_MODEL_PATH,
                "",
                (320, 320),
                score_threshold=_YUNET_CONF,
                nms_threshold=0.3,
                top_k=50,
            )
            logger.info("YuNet face detector loaded in %.0fms (conf>=%.2f)",
                        (time.perf_counter() - t0) * 1000, _YUNET_CONF)
        except Exception as e:
            logger.error("YuNet load failed: %s", e)
            _yunet = None
    return _yunet


def _detect_face_yunet(frame: npt.NDArray[np.uint8]) -> Optional[Tuple[int, int, int, int]]:
    """Run YuNet on the frame, return the largest face bbox (x,y,w,h) or None.

    Largest-face policy: most prominent / closest face — predictable for a single
    tracking session. If multiple people, the closest one wins.
    """
    detector = _get_yunet()
    if detector is None:
        return None
    h, w = frame.shape[:2]
    try:
        detector.setInputSize((w, h))
        t0 = time.perf_counter()
        _, faces = detector.detect(frame)
        latency_ms = (time.perf_counter() - t0) * 1000
    except Exception as e:
        logger.warning("YuNet detect failed: %s", e)
        return None
    if faces is None or len(faces) == 0:
        logger.info("[tracking_yunet] not found latency=%.0fms", latency_ms)
        return None
    # faces rows: [x, y, w, h, lm_x1..lm_y5, score]. Pick the largest by area.
    best = max(faces, key=lambda f: float(f[2]) * float(f[3]))
    x, y, fw, fh = int(best[0]), int(best[1]), int(best[2]), int(best[3])
    score = float(best[-1])
    # Clamp to frame in case the detector returns slightly negative coords.
    x = max(0, x); y = max(0, y)
    fw = max(1, min(fw, w - x)); fh = max(1, min(fh, h - y))
    logger.info("[tracking_yunet] face bbox=(%d,%d,%d,%d) score=%.3f count=%d latency=%.0fms",
                x, y, fw, fh, score, len(faces), latency_ms)
    return (x, y, fw, fh)


class ObjectDetector:
    """Detect an object by name: YuNet for faces, local YOLOv8n for COCO
    classes, remote YOLOWorld for open vocabulary.

    Owns the remote-call encryption session and the remote-fallback throttle.
    on_confidence (optional) is called with the detection confidence when the
    remote path finds the target — mirrors the tracker status field.
    """

    def __init__(self, on_confidence: Optional[Callable[[float], None]] = None):
        self._on_confidence = on_confidence
        # perf_counter of the last remote-YOLOWorld fallback attempt (throttle).
        self._last_remote_attempt_t: float = 0.0
        self._crypto: CryptoSession | None = None
        if config.DL_ENCRYPTION_ENABLED:
            public_key = resolve_public_key(config.DL_PUBLIC_KEY_URL, config.DL_API_KEY, config.DL_PUBLIC_KEY_FILE)
            if public_key is not None:
                self._crypto = CryptoSession(public_key)
                logger.info("Tracker: encryption enabled for remote YOLOWorld")
            elif config.DL_ENCRYPTION_REQUIRED:
                logger.error("Tracker: encryption required but no public key available")

    def detect(self, frame: npt.NDArray[np.uint8], target: str) -> Optional[Tuple[int, int, int, int]]:
        """Detect an object by name. Tries local YOLOv8n first (fast, COCO classes),
        falls back to remote YOLOWorld API for open-vocab targets.

        Returns (x, y, w, h) top-left bbox in ORIGINAL camera coords, or None.
        """
        target_key = (target or "").lower().strip()

        # Run every detector on the downscaled frame for speed; map any bbox back
        # to original coords before returning so callers/servo math are unaware.
        frame, _scale = downscale(frame)
        _up = 1.0 / _scale if _scale else 1.0

        # --- Path 0: YuNet face detector (target = face) ---
        # COCO has no face class; this avoids the ~1.3s remote round-trip for what
        # is a common tracking target.
        if _FACE_DETECTOR_ENABLED and target_key in _FACE_TARGET_ALIASES:
            face_bbox = _detect_face_yunet(frame)
            if face_bbox is not None:
                return scale_bbox(face_bbox, _up)
            # YuNet missed — fall through to remote YOLOWorld below.

        # --- Path 1: local YOLOv8n (if target maps to COCO class) ---
        coco_idx = _COCO_CLASSES.get(target_key)
        if _DETECT_LOCAL_ENABLED and coco_idx is not None:
            model = _get_local_yolo()
            if model is not None:
                t_req = time.perf_counter()
                try:
                    results = model(frame, verbose=False, imgsz=_LOCAL_IMGSZ,
                                    classes=[coco_idx], conf=C.DETECT_MIN_CONFIDENCE)
                    t_ms = (time.perf_counter() - t_req) * 1000
                    h_fr, w_fr = frame.shape[:2]
                    frame_area = float(h_fr * w_fr)
                    best = None
                    for r in results:
                        if r.boxes is None or len(r.boxes) == 0:
                            continue
                        for b in r.boxes:
                            x1, y1, x2, y2 = b.xyxy[0].tolist()
                            conf = float(b.conf[0])
                            bw = int(x2 - x1)
                            bh = int(y2 - y1)
                            area_ratio = (bw * bh) / frame_area if frame_area > 0 else 0.0
                            if not (C.DETECT_MIN_AREA_RATIO <= area_ratio <= C.DETECT_MAX_AREA_RATIO):
                                continue
                            if best is None or conf > best[1]:
                                best = ((int(x1), int(y1), bw, bh), conf, area_ratio)
                    if best is not None:
                        bbox, conf, area_ratio = best
                        logger.info("[tracking_yolo_local] target='%s' bbox=%s conf=%.3f area=%.1f%% latency=%.0fms",
                                    target, bbox, conf, area_ratio * 100, t_ms)
                        return scale_bbox(bbox, _up)
                    logger.info("[tracking_yolo_local] target='%s' not found latency=%.0fms", target, t_ms)
                    # Local missed. Fall back to remote open-vocab YOLOWorld for
                    # small/far objects local can't see — but throttle it so a
                    # truly-unseeable target doesn't hit remote on every redetect.
                    now_fb = time.perf_counter()
                    if now_fb - self._last_remote_attempt_t < REMOTE_FALLBACK_MIN_INTERVAL:
                        return None
                    self._last_remote_attempt_t = now_fb
                    logger.info("[tracking_yolo_local] miss → remote YOLOWorld fallback target='%s'", target)
                    # fall through to Path 2 (remote)
                except Exception as e:
                    logger.warning("Local YOLO inference failed: %s — falling back to remote", e)
        elif coco_idx is None:
            logger.info("[tracking_yolo] target='%s' not in COCO — using remote", target)

        return self._detect_remote(frame, target, _up)

    def _detect_remote(self, frame: npt.NDArray[np.uint8], target: str,
                       up_factor: float) -> Optional[Tuple[int, int, int, int]]:
        """Path 2: remote YOLOWorld API (open-vocab fallback). `frame` is already
        downscaled; up_factor maps the result back to original coords."""
        from hal.config import DL_BACKEND_URL, DL_API_KEY
        if not DL_BACKEND_URL:
            logger.error("YOLOWorld: DL_BACKEND_URL not configured")
            return None

        url = DL_BACKEND_URL.rstrip("/") + "/" + _YOLO_ENDPOINT.strip("/")
        logger.info("[tracking_yolo_request] target='%s' url=%s", target, url)
        t_req = time.perf_counter()
        try:
            _, buf = cv2.imencode(".jpg", frame, [cv2.IMWRITE_JPEG_QUALITY, 85])
            img_b64 = base64.b64encode(buf.tobytes()).decode()

            payload = {"image_b64": img_b64, "classes": [target]}
            headers: dict[str, str] = {"Content-Type": "application/json"}
            if DL_API_KEY:
                headers["X-API-Key"] = DL_API_KEY
            if self._crypto is not None:
                resp = requests.post(
                    url,
                    data=self._crypto.wrap_http_request(json.dumps(payload).encode()),
                    headers=headers,
                    timeout=_YOLO_TIMEOUT,
                )
            else:
                resp = requests.post(
                    url,
                    json=payload,
                    headers=headers,
                    timeout=_YOLO_TIMEOUT,
                )
            if resp.status_code != 200:
                logger.warning("YOLOWorld HTTP %d: %s", resp.status_code, resp.text[:200])
                return None

            if self._crypto is not None:
                detections = json.loads(self._crypto.unwrap_http_response(resp.content))
            else:
                detections = resp.json()
            if not detections:
                logger.info("YOLOWorld: '%s' not found in frame", target)
                return None

            frame_area = float(frame.shape[0] * frame.shape[1])
            valid = []
            for d in detections:
                cx, cy, w, h = d["xywh"]
                conf = d.get("confidence", 0)
                area_ratio = (w * h) / frame_area if frame_area > 0 else 0.0
                cname = d.get("class_name", "?")
                if conf < C.DETECT_MIN_CONFIDENCE:
                    reason = "REJECTED (conf)"
                elif not (C.DETECT_MIN_AREA_RATIO <= area_ratio <= C.DETECT_MAX_AREA_RATIO):
                    reason = "REJECTED (size)"
                else:
                    reason = "ACCEPTED"
                logger.info(
                    "  YOLO candidate: class='%s' conf=%.3f bbox=(%d,%d,%d,%d) area=%.1f%% %s",
                    cname, conf, int(cx - w / 2), int(cy - h / 2), int(w), int(h),
                    area_ratio * 100, reason,
                )
                if reason == "ACCEPTED":
                    valid.append(d)

            if not valid:
                logger.warning(
                    "YOLOWorld: '%s' — %d detection(s) but none passed filters "
                    "(conf >= %.2f, area %.1f%%–%.1f%%)",
                    target, len(detections), C.DETECT_MIN_CONFIDENCE,
                    C.DETECT_MIN_AREA_RATIO * 100, C.DETECT_MAX_AREA_RATIO * 100,
                )
                return None

            best = max(valid, key=lambda d: d.get("confidence", 0))
            cx, cy, w, h = best["xywh"]
            x = int(cx - w / 2)
            y = int(cy - h / 2)
            bbox = (x, y, int(w), int(h))
            latency_ms = (time.perf_counter() - t_req) * 1000
            if self._on_confidence is not None:
                self._on_confidence(round(best["confidence"], 3))
            logger.info("YOLOWorld: '%s' found at bbox=%s conf=%.3f", target, bbox, best["confidence"])
            logger.info("[tracking_yolo_response] target='%s' found=True bbox=%s conf=%.3f latency=%.0fms",
                        target, bbox, best["confidence"], latency_ms)
            return scale_bbox(bbox, up_factor)
        except Exception as e:
            logger.error("YOLOWorld detect failed: %s", e)
            return None
