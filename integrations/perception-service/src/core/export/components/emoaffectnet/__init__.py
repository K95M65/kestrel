"""Emo-AffectNet static ResNet-50 architecture (export-only).

Vendored from ElenaRyumina/face_emotion_recognition (MIT) so the .pt
checkpoint can be loaded and exported to ONNX. Only the static image
classifier (ResNet-50) is included — the temporal LSTM head is not used
by the single-crop server pipeline.
"""

from core.export.components.emoaffectnet.resnet import ResNet50

__all__ = ["ResNet50"]
