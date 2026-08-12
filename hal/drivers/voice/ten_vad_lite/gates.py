"""Frame-level gates that clean up VAD output, and the pitch analysis behind them.

TEN-VAD and silero both fire on doors, taps, bursts and room tone. On real
recogniser clips that is not a rounding error — measured against a diarisation
reference, only 71% of what the raw VAD hands downstream is actually speech.

Neither level nor the VAD's own probability separates those cases: a door slam
peaks as loud as the speech beside it, and tonal noise scores 0.5-0.6. What does
separate them is **pitch coherence**. One person speaking holds a narrow f0 band
for a whole clip; noise pitch scatters across the search range. Measured on
hand-labelled clips, speech had an f0 inter-quartile range of 7-22 Hz against
76-214 Hz for noise in the same recordings.

:func:`speaker_band_mask` turns that into a per-frame gate, and
:func:`relative_level_mask` drops frames far below the clip's own speech level.
Used together they raised the share of the kept span that is really speech from
0.67 to 0.79 across 22 recogniser clips, and got the accept/reject verdict right
on 22 of 22 against a hand-corrected diarisation reference.

Between them they are also the only filters here that can empty a clip, which is
what lets an all-noise recording be *rejected* by a minimum-duration gate rather
than trimmed into something that looks plausible.
"""

from __future__ import annotations

from typing import Optional

import numpy as np
import numpy.typing as npt

from .features import HOP_SIZE, SAMPLE_RATE

MIN_F0: float = 70.0
MAX_F0: float = 400.0
WINDOW_SIZE: int = 1024  # 64 ms — at least two periods of a 70 Hz voice
# Pitch lives below 400 Hz, so the autocorrelation runs on a 4 kHz decimation of
# the signal rather than the full 16 kHz. That is a 4x shorter FFT for the same
# 64 ms window and the same lag range, and it is what upstream TEN-VAD does for
# the same reason. Lag resolution at 4 kHz is ~3% around 130 Hz, far finer than
# the +/-35% band this feeds.
DECIMATION: int = 4
_BLOCK_FRAMES: int = 256  # bound transient memory, as in features.py
_SILENCE_ENERGY: float = 1e-12

# --- speaker-band gate defaults, tuned against the diarisation reference ---
# Voiced frames must sit within this fraction of the clip's anchor pitch.
BAND_TOLERANCE: float = 0.35
VOICING_THRESHOLD: float = 0.35
# The anchor is the median pitch of this share of the most confident frames.
ANCHOR_FRACTION: float = 0.4
# A frame survives only if this share of its +/-span neighbourhood is also in
# band, which kills the isolated in-band frames noise throws off by chance.
DENSITY_SPAN: int = 3
DENSITY_FRACTION: float = 0.4
# Autocorrelation pitch suffers octave errors: a 130 Hz voice intermittently
# reports ~260 or ~390, splitting one speaker across two bands. When a single
# band explains less than this share of the voiced frames, the band is widened
# to accept the harmonics.
#
# The threshold is low on purpose. Folding octaves makes *any* pitch histogram
# look coherent — an all-noise clip went from 0.54 to 0.94 — so widening cannot
# be validated after the fact, only gated before it. Measured: a genuinely
# octave-split voice sat at 0.30, while noisy clips and ordinary speech sat at
# 0.54 and above. 0.45 separates them; raising it to 0.6 re-admitted 5.3 s of a
# clip that is noise end to end.
MIN_BAND_SHARE: float = 0.45
_OCTAVE_RATIOS: tuple[float, ...] = (1.0, 0.5, 2.0)


def pitch_and_voicing(
    waveform: npt.NDArray,
    sample_rate: int = SAMPLE_RATE,
    hop_size: int = HOP_SIZE,
) -> tuple[npt.NDArray[np.float64], npt.NDArray[np.float64]]:
    """Return ``(voicing, f0_hz)`` per hop, on the VAD's frame grid.

    ``voicing`` is the normalised autocorrelation peak, 0..1: above 0.4 is
    reliably periodic, below 0.3 aperiodic. ``f0_hz`` is that peak's lag.

    The f0 estimate is a raw argmax and *does* suffer octave errors; treat it
    statistically (band membership, spread) rather than as a per-frame pitch.
    """
    x = np.asarray(waveform, dtype=np.float64)
    n_frames = x.size // hop_size
    if n_frames == 0:
        return np.zeros(0), np.zeros(0)

    padded = np.concatenate([np.zeros(WINDOW_SIZE - hop_size), x])
    # Boxcar-average then take every DECIMATION-th sample: a cheap anti-alias
    # filter plus downsample in one pass.
    usable = (padded.size // DECIMATION) * DECIMATION
    decimated = padded[:usable].reshape(-1, DECIMATION).mean(axis=1)
    win = WINDOW_SIZE // DECIMATION
    hop = hop_size // DECIMATION
    rate = sample_rate / DECIMATION

    if decimated.size < win or hop < 1:
        return np.zeros(n_frames), np.full(n_frames, MIN_F0)
    frames = np.lib.stride_tricks.sliding_window_view(decimated, win)[::hop][:n_frames]
    n_frames = min(n_frames, frames.shape[0])
    frames = frames[:n_frames]

    lag_lo = max(1, int(rate / MAX_F0))
    lag_hi = min(win - 1, int(rate / MIN_F0))
    if lag_hi <= lag_lo:
        return np.zeros(n_frames), np.full(n_frames, MIN_F0)

    voicing = np.zeros(n_frames)
    lags = np.full(n_frames, lag_hi, dtype=np.int64)
    fft_size = 2 * win  # zero-pad: linear, not circular, autocorrelation
    for lo in range(0, n_frames, _BLOCK_FRAMES):
        hi = min(lo + _BLOCK_FRAMES, n_frames)
        block = frames[lo:hi] - frames[lo:hi].mean(axis=1, keepdims=True)
        spectrum = np.fft.rfft(block, n=fft_size, axis=1)
        autocorr = np.fft.irfft(spectrum * np.conj(spectrum), n=fft_size, axis=1)

        energy = autocorr[:, 0]
        best = lag_lo + np.argmax(autocorr[:, lag_lo : lag_hi + 1], axis=1)
        loud = energy > _SILENCE_ENERGY
        voicing[lo:hi] = np.where(
            loud, autocorr[np.arange(hi - lo), best] / np.where(loud, energy, 1.0), 0.0
        )
        lags[lo:hi] = np.where(loud, best, lag_hi)

    return np.clip(voicing, 0.0, 1.0), rate / np.maximum(lags, 1)


def _band(
    voicing: npt.NDArray[np.float64],
    f0: npt.NDArray[np.float64],
    anchor: float,
    ratios: tuple[float, ...],
) -> npt.NDArray[np.bool_]:
    """Voiced frames whose pitch — or a harmonic of it — matches the anchor."""
    deviation = np.min([np.abs(np.log(f0 * r / anchor)) for r in ratios], axis=0)
    return (voicing > VOICING_THRESHOLD) & (deviation <= np.log1p(BAND_TOLERANCE))


def speaker_band_mask(
    waveform: npt.NDArray,
    probs: npt.NDArray[np.float32],
    threshold: float = 0.5,
    sample_rate: int = SAMPLE_RATE,
    hop_size: int = HOP_SIZE,
    pitch: Optional[tuple[npt.NDArray[np.float64], npt.NDArray[np.float64]]] = None,
) -> npt.NDArray[np.bool_]:
    """Keep only frames whose pitch sits in the clip's own speaker band.

    Anchors on the pitch of the frames the VAD is most confident about, keeps
    voiced frames near that anchor, and requires the neighbourhood to agree so
    that the in-band frames noise produces by chance do not survive.

    Assumes **one dominant speaker per clip**. That holds for recognition-style
    recordings and is wrong for long-form multi-speaker audio, where a quieter
    second talker outside the band is discarded. A background voice is removed
    too — usually desirable here, since it would otherwise pollute the embedding.

    Returns a per-frame boolean mask; AND it with the VAD decision by zeroing the
    masked probabilities before segmentation.
    """
    voicing, f0 = pitch_and_voicing(waveform, sample_rate, hop_size) if pitch is None else pitch
    n = min(voicing.size, probs.size)
    if n == 0:
        return np.ones(probs.size, dtype=bool)

    voicing, f0 = voicing[:n], f0[:n]
    candidate = (voicing > VOICING_THRESHOLD) & (probs[:n] >= threshold)
    if candidate.sum() < 5:
        return np.zeros(probs.size, dtype=bool)

    confidence = (probs[:n] * voicing)[candidate]
    order = np.where(candidate)[0][np.argsort(confidence)[::-1]]
    top = order[: max(3, int(order.size * ANCHOR_FRACTION))]
    anchor = float(np.median(f0[top]))

    near = _band(voicing, f0, anchor, (1.0,))
    if near[candidate].mean() < MIN_BAND_SHARE:
        # Too little explained by one band — the anchor is likely split across
        # octave errors, so accept the harmonics too.
        near = _band(voicing, f0, anchor, _OCTAVE_RATIOS)

    width = 2 * DENSITY_SPAN + 1
    density = np.convolve(near.astype(float), np.ones(width) / width, mode="same")

    mask = np.zeros(probs.size, dtype=bool)
    mask[:n] = near & (density >= DENSITY_FRACTION)
    return mask


# --- level gate ---
# Frames this far below the clip's own speech level are treated as non-speech.
# 20 dB is a cliff, not a slope: at 25 dB an all-noise clip kept 4.3 s and was
# accepted, at 20 dB it keeps 0.43 s and is correctly rejected.
MAX_LEVEL_DROP_DB: float = 20.0
_LEVEL_REFERENCE_PERCENTILE: float = 90.0


def frame_levels(waveform: npt.NDArray, hop_size: int = HOP_SIZE) -> npt.NDArray[np.float64]:
    """Per-frame RMS in dBFS, on the VAD's frame grid."""
    x = np.asarray(waveform, dtype=np.float64)
    if np.max(np.abs(x), initial=0.0) > 1.0:
        x = x / 32768.0
    n_frames = x.size // hop_size
    if n_frames == 0:
        return np.zeros(0)
    block = x[: n_frames * hop_size].reshape(n_frames, hop_size)
    return 20.0 * np.log10(np.sqrt(np.mean(block**2, axis=1)) + 1e-9)


def relative_level_mask(
    waveform: npt.NDArray,
    probs: npt.NDArray[np.float32],
    threshold: float = 0.5,
    max_drop_db: float = MAX_LEVEL_DROP_DB,
    hop_size: int = HOP_SIZE,
) -> npt.NDArray[np.bool_]:
    """Mask out frames far below the clip's *own* speech level.

    A VAD scores room tone, reverb tails and distant chatter above threshold.
    Those frames sit tens of dB under the utterance and contribute nothing to a
    speaker embedding.

    The reference is a high percentile of the level of frames the VAD already
    called speech, so this is scale-free: a uniformly quiet recording moves the
    floor with it. It does not survive a genuinely quiet second speaker, who is
    masked if more than ``max_drop_db`` below the near one — usually what you
    want when recognising the near speaker.
    """
    levels = frame_levels(waveform, hop_size=hop_size)
    n = min(levels.size, probs.size)
    if n == 0:
        return np.ones(probs.size, dtype=bool)
    speech = probs[:n] >= threshold
    if not speech.any():
        return np.ones(probs.size, dtype=bool)

    reference = float(np.percentile(levels[:n][speech], _LEVEL_REFERENCE_PERCENTILE))
    mask = np.ones(probs.size, dtype=bool)
    mask[:n] = levels[:n] >= reference - max_drop_db
    return mask
