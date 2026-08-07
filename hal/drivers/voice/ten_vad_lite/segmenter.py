"""Turn per-frame speech probabilities into speech segments.

The hysteresis + min-duration + padding logic mirrors silero-vad's
``get_speech_timestamps`` (onset at ``threshold``, offset at
``threshold - 0.15``, min speech/silence durations, symmetric padding with
neighbour-aware clamping) so this is a behavioural drop-in for code currently
built around silero. Only the probability source differs.
"""

from __future__ import annotations

from typing import Sequence, TypedDict

import numpy as np
import numpy.typing as npt

from .features import HOP_SIZE


class SpeechTimestamp(TypedDict):
    start: int
    end: int


OFFSET_MARGIN: float = 0.15  # silero's fixed onset/offset hysteresis gap


def probs_to_segments(
    probs: Sequence[float] | npt.NDArray[np.float32],
    num_samples: int,
    threshold: float = 0.5,
    min_speech_duration_ms: int = 250,
    min_silence_duration_ms: int = 100,
    speech_pad_ms: int = 30,
    sample_rate: int = 16000,
    hop_size: int = HOP_SIZE,
) -> list[SpeechTimestamp]:
    """Convert frame probabilities to ``[{"start": int, "end": int}]`` in samples.

    Args:
        probs: one probability per ``hop_size`` samples.
        num_samples: length of the source waveform, used to clamp the last segment.
        threshold: speech onset probability. Offset uses ``threshold - 0.15``.
        min_speech_duration_ms: segments shorter than this are dropped.
        min_silence_duration_ms: gaps shorter than this do not close a segment.
        speech_pad_ms: padding added to each side of every segment.
    """
    probs = np.asarray(probs, dtype=np.float32)
    if probs.size == 0:
        return []

    neg_threshold = max(threshold - OFFSET_MARGIN, 0.01)
    min_speech_samples = int(sample_rate * min_speech_duration_ms / 1000)
    min_silence_samples = int(sample_rate * min_silence_duration_ms / 1000)
    speech_pad_samples = int(sample_rate * speech_pad_ms / 1000)

    segments: list[SpeechTimestamp] = []
    triggered = False
    start = 0
    temp_end = 0

    for i, prob in enumerate(probs):
        pos = hop_size * i

        if prob >= threshold and temp_end:
            temp_end = 0

        if prob >= threshold and not triggered:
            triggered = True
            start = pos
            continue

        if prob < neg_threshold and triggered:
            if not temp_end:
                temp_end = pos
            if pos - temp_end < min_silence_samples:
                continue
            if temp_end - start > min_speech_samples:
                segments.append({"start": start, "end": temp_end})
            temp_end = 0
            triggered = False

    if triggered:
        end = num_samples
        if end - start > min_speech_samples:
            segments.append({"start": start, "end": end})

    # Pad each segment, splitting the gap when neighbours would overlap.
    for i, seg in enumerate(segments):
        if i == 0:
            seg["start"] = int(max(0, seg["start"] - speech_pad_samples))
        if i != len(segments) - 1:
            silence = segments[i + 1]["start"] - seg["end"]
            if silence < 2 * speech_pad_samples:
                seg["end"] += silence // 2
                segments[i + 1]["start"] = int(max(0, segments[i + 1]["start"] - silence // 2))
            else:
                seg["end"] = int(min(num_samples, seg["end"] + speech_pad_samples))
                segments[i + 1]["start"] = int(
                    max(0, segments[i + 1]["start"] - speech_pad_samples)
                )
        else:
            seg["end"] = int(min(num_samples, seg["end"] + speech_pad_samples))

    return segments
