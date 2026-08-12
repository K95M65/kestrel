"""ten_vad_lite — TEN-VAD with nothing but numpy + onnxruntime.

VENDORED third-party package. Upstream source lives outside this repo (the
``ten_vad_lite`` project); it is copied in verbatim rather than pip-installed
because the speaker-recognition VAD must ship with HAL over OTA, and the only
runtime deps (numpy, onnxruntime) are already HAL dependencies. Re-vendor by
copying the upstream ``ten_vad_lite/`` package over this directory — do not
hand-edit the modules here.

Runs the upstream 300 KB ``ten-vad.onnx`` model directly: the C feature
front-end (pre-emphasis, 768-pt Hann STFT, 40 mel bands, normalisation) is
reimplemented in numpy, so none of libten_vad, sherpa-onnx or torch is needed.

``assets/ten-vad.onnx`` is the ORIGINAL FP32 model copied verbatim from
TEN-framework/ten-vad (Apache-2.0). No quantized/INT8 variant is shipped or
supported here.

Typical use — a silero-compatible speech-timestamp API::

    from hal.drivers.voice.ten_vad_lite import TenVad

    vad = TenVad()
    segments = vad.get_speech_timestamps(waveform, sample_rate=16000)
    # -> [{"start": 4096, "end": 30720}, ...]  sample indices

Streaming use::

    vad = TenVad()
    for hop in chunks_of_256_samples:
        prob = vad.model.process_hop(hop * 32768.0)
"""

from .features import FEATURE_DIM, HOP_SIZE, SAMPLE_RATE, FeatureExtractor
from .model import TenVadModel
from .gates import pitch_and_voicing, relative_level_mask, speaker_band_mask
from .segmenter import SpeechTimestamp, probs_to_segments
from .vad import TenVad

__all__ = [
    "FEATURE_DIM",
    "HOP_SIZE",
    "SAMPLE_RATE",
    "FeatureExtractor",
    "SpeechTimestamp",
    "TenVad",
    "TenVadModel",
    "pitch_and_voicing",
    "relative_level_mask",
    "speaker_band_mask",
    "probs_to_segments",
]
