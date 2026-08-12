"""High-level TEN-VAD API: waveform in, speech probabilities or segments out."""

from __future__ import annotations

import pathlib

import numpy as np
import numpy.typing as npt

from .features import HOP_SIZE, SAMPLE_RATE
from .model import DEFAULT_MODEL_PATH, DEFAULT_STATE_RESET_FRAMES, TenVadModel
from .gates import relative_level_mask, speaker_band_mask
from .segmenter import SpeechTimestamp, probs_to_segments


def to_int16_scale(waveform: npt.NDArray) -> npt.NDArray[np.float32]:
    """Normalise input to the int16 float scale the model front-end expects.

    Accepts int16 arrays, or float arrays in [-1, 1] (the usual librosa /
    soundfile convention), which are scaled by 32768.
    """
    arr = np.asarray(waveform)
    if arr.ndim != 1:
        raise ValueError(f"expected mono 1-D audio, got shape {arr.shape}")
    if arr.dtype == np.int16:
        return arr.astype(np.float32)
    arr = arr.astype(np.float32)
    if arr.size and np.max(np.abs(arr)) <= 1.0:
        arr = arr * np.float32(32768.0)
    return arr


class TenVad:
    """TEN-VAD wrapper with a silero-compatible speech-timestamp API.

    One instance holds one onnxruntime session and is **not** thread-safe;
    guard it with a lock or use one instance per thread.
    """

    def __init__(
        self,
        model_path: str | pathlib.Path = DEFAULT_MODEL_PATH,
        num_threads: int = 1,
        state_reset_frames: int = DEFAULT_STATE_RESET_FRAMES,
        pitch_hz: float | None = None,
    ) -> None:
        self.model = TenVadModel(
            model_path=model_path,
            num_threads=num_threads,
            state_reset_frames=state_reset_frames,
            pitch_hz=pitch_hz,
        )

    def reset(self) -> None:
        """Clear streaming state — call between unrelated utterances."""
        self.model.reset()

    def probabilities(
        self, waveform: npt.NDArray, sample_rate: int = SAMPLE_RATE, reset: bool = True
    ) -> npt.NDArray[np.float32]:
        """Return one speech probability per 16 ms hop.

        A trailing partial hop is zero-padded so short clips still yield frames.
        """
        if sample_rate != SAMPLE_RATE:
            raise ValueError(
                f"ten-vad only supports {SAMPLE_RATE} Hz audio; got {sample_rate} Hz. "
                "Resample before calling."
            )
        if reset:
            self.reset()

        samples = to_int16_scale(waveform)
        remainder = samples.size % HOP_SIZE
        if remainder:
            samples = np.concatenate(
                [samples, np.zeros(HOP_SIZE - remainder, dtype=np.float32)]
            )
        return self.model.process(samples)

    def get_speech_timestamps(
        self,
        waveform: npt.NDArray,
        sample_rate: int = SAMPLE_RATE,
        threshold: float = 0.5,
        min_speech_duration_ms: int = 250,
        min_silence_duration_ms: int = 100,
        speech_pad_ms: int = 30,
        return_seconds: bool = False,
        speaker_band: bool = False,
        max_level_drop_db: float | None = None,
    ) -> list[SpeechTimestamp]:
        """Detect speech segments. Signature mirrors silero's helper of the same name.

        Returns a list of ``{"start": ..., "end": ...}`` in sample indices, or in
        seconds when ``return_seconds`` is set.

        ``speaker_band`` and ``max_level_drop_db`` are off by default, so the
        result is plain TEN-VAD. Enable both to suppress VAD false positives on
        doors, taps and room tone; they are designed to work together and were
        measured that way. ``speaker_band`` assumes one dominant speaker per
        clip — see :mod:`ten_vad_lite.gates`.
        """
        arr = np.asarray(waveform)
        probs = self.probabilities(arr, sample_rate=sample_rate)
        if speaker_band:
            probs = np.where(
                speaker_band_mask(arr, probs, threshold=threshold, sample_rate=sample_rate),
                probs, np.float32(0.0),
            )
        if max_level_drop_db is not None:
            probs = np.where(
                relative_level_mask(arr, probs, threshold=threshold,
                                    max_drop_db=max_level_drop_db),
                probs, np.float32(0.0),
            )
        segments = probs_to_segments(
            probs,
            num_samples=int(arr.shape[0]),
            threshold=threshold,
            min_speech_duration_ms=min_speech_duration_ms,
            min_silence_duration_ms=min_silence_duration_ms,
            speech_pad_ms=speech_pad_ms,
            sample_rate=sample_rate,
        )
        if return_seconds:
            return [
                {  # type: ignore[misc]
                    "start": round(seg["start"] / sample_rate, 3),
                    "end": round(seg["end"] / sample_rate, 3),
                }
                for seg in segments
            ]
        return segments
