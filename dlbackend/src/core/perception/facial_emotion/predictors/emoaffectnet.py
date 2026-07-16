"""Emo-AffectNet emotion predictor (7-class, AffectNet).

Static ResNet-50 facial expression recognizer from
ElenaRyumina/face_emotion_recognition. Pure emotion classification from a
face crop — no face detection.

Input: 224x224 RGB face crop scaled to [0, 1] (base predictor's ``/255``).
The model's native preprocessing (RGB->BGR, ``*255``, VGGFace2 mean
subtraction) and the final softmax are **baked into the ONNX export
wrapper** (see ``core/export/entries/export_emoaffectnet.py``), so at
runtime this predictor is trivial: identity MEAN/STD and the default
single-output postprocess.
"""

from pathlib import Path

import numpy as np
import numpy.typing as npt

from core.enums.files import ModelEnum
from core.perception.facial_emotion.constants import RESOURCES_DIR
from core.perception.facial_emotion.predictors.base import EmotionRecognizer
from core.utils.files import get_default_cdn_url, get_default_model_path


class EmoAffectNetRecognizer(EmotionRecognizer):
    """Emo-AffectNet (static ResNet-50) ONNX emotion predictor."""

    DEFAULT_MODEL_PATH: Path | None = get_default_model_path(ModelEnum.EMOAFFECTNET_ONNX)
    DEFAULT_REMOTE_URL: str | None = get_default_cdn_url(ModelEnum.EMOAFFECTNET_ONNX)
    DEFAULT_CLASSES_PATH: Path = RESOURCES_DIR / "emoaffectnet_classes.txt"
    DEFAULT_INPUT_SIZE: tuple[int, int] = (224, 224)

    # Identity — RGB->BGR flip, *255, and VGGFace2 mean subtraction are all
    # baked into the ONNX export wrapper, so the base predictor's
    # ``(x/255 - MEAN)/STD`` must be a no-op beyond the ``/255`` it already does.
    MEAN: npt.NDArray[np.float32] = np.array([0, 0, 0], dtype=np.float32)
    STD: npt.NDArray[np.float32] = np.array([1, 1, 1], dtype=np.float32)
