"""TEN-VAD front-end (log-mel + pitch) reimplemented with numpy.

This is a line-by-line port of the feature path in the upstream C sources
(``ten-vad/src/aed.cc``, ``stft.cc``, ``coeff.h``), so the ONNX model can be run
without the prebuilt ``libten_vad`` shared library:

    int16-scale samples
      -> pre-emphasis (0.97, state carried across hops)
      -> 768-sample FIFO * Hann-768 window, zero-padded to 1024
      -> rFFT -> power spectrum (513 bins)
      -> 40 triangular mel filters (HTK mel scale, 0..8000 Hz)
      -> log(E / 32768^2 + 1e-20)
      -> concat pitch (Hz) as feature 40
      -> (x - mean) / (std + 1e-20)

Reference points in upstream:
  * framing/window/FFT      ``stft.cc:AUP_Analyzer_proc``
  * pre-emphasis            ``aed.cc:AUP_Aed_proc``
  * mel bank construction   ``aed.cc:AUP_Aed_resetVariables``
  * mel/log/normalisation   ``aed.cc:AUP_Aed_aivad_proc``
"""

from __future__ import annotations

import pathlib

import numpy as np
import numpy.typing as npt

# --- Constants, mirroring ten-vad/src/aed_st.h + coeff.h ---
SAMPLE_RATE: int = 16000
HOP_SIZE: int = 256  # AUP_AED_ASSUMED_HOPSZ — 16 ms
WINDOW_SIZE: int = 768  # AUP_AED_ASSUMED_WINDOWSZ
FFT_SIZE: int = 1024  # AUP_AED_ASSUMED_FFTSZ
N_BINS: int = FFT_SIZE // 2 + 1  # 513
N_MEL: int = 40  # AUP_AED_MEL_FILTER_BANK_NUM
FEATURE_DIM: int = N_MEL + 1  # AUP_AED_FEA_LEN — 40 mel + 1 pitch
CONTEXT_FRAMES: int = 3  # AUP_AED_CONTEXT_WINDOW_LEN

_EPS: float = 1e-20  # AUP_AED_EPS
_PREEMPHASIS: float = 0.97
_POWER_NORMALIZER: float = 32768.0 * 32768.0

_COEFF_PATH = pathlib.Path(__file__).with_name("assets") / "coeffs.npz"

# Frames are transformed in blocks rather than one batch per clip. numpy's FFT
# upcasts float32 to complex128 internally, so a whole-clip batch costs ~2.5
# MB/s of audio in transients; blocking caps that at a few MB no matter how long
# the clip is. Results are unchanged — FFT rows are independent.
_FFT_BLOCK_FRAMES: int = 256  # ~4 s of audio per block


def _load_coeffs() -> tuple[npt.NDArray[np.float32], npt.NDArray[np.float32], npt.NDArray[np.float32]]:
    with np.load(_COEFF_PATH) as data:
        return (
            data["window"].astype(np.float32),
            data["feat_mean"].astype(np.float32),
            data["feat_std"].astype(np.float32),
        )


def build_mel_filterbank() -> npt.NDArray[np.float32]:
    """Return the (N_BINS, N_MEL) triangular mel matrix used by TEN-VAD.

    Port of the filter-bank generation in ``AUP_Aed_resetVariables``. Uses the
    HTK mel scale (2595*log10(1+f/700)) with bin edges truncated to int, and
    unnormalised triangles — i.e. *not* the same as librosa/slaney banks.
    All arithmetic is done in float32 so the int() truncation of the bin edges
    lands on exactly the same bins as the C code.
    """
    f32 = np.float32
    low_mel = f32(2595.0) * np.log10(f32(1.0) + f32(0.0) / f32(700.0))
    high_mel = f32(2595.0) * np.log10(f32(1.0) + f32(8000.0) / f32(700.0))

    edges: list[int] = []
    for i in range(N_MEL + 2):
        mel_point = f32(i) * (high_mel - low_mel) / (f32(N_MEL) + f32(1.0)) + low_mel
        hz_point = f32(700.0) * (np.power(f32(10.0), mel_point / f32(2595.0)) - f32(1.0))
        edge = int((f32(FFT_SIZE) + f32(1.0)) * hz_point / f32(SAMPLE_RATE))
        if edges and edge == edges[-1]:
            raise ValueError("degenerate mel filter bank (duplicate bin edge)")
        edges.append(edge)

    bank = np.zeros((N_MEL, N_BINS), dtype=np.float32)
    for j in range(N_MEL):
        lo, mid, hi = edges[j], edges[j + 1], edges[j + 2]
        for i in range(lo, mid):
            bank[j, i] = f32(i - lo) / f32(mid - lo)
        for i in range(mid, hi):
            bank[j, i] = f32(hi - i) / f32(hi - mid)
    return bank.T.copy()  # (N_BINS, N_MEL) for a single matmul


class FeatureExtractor:
    """Streaming TEN-VAD feature extractor.

    Call :meth:`process` with any number of whole hops (multiples of 256
    samples); it returns one 41-dim feature vector per hop. State (pre-emphasis
    memory + the 768-sample analysis FIFO) is carried between calls, so
    streaming and one-shot use produce identical features.

    Input samples are expected on the **int16 scale** (-32768..32767) as
    floats — the same convention as upstream's ``ten_vad_process``.

    The 40 log-mel features are bit-faithful to the C library. Feature 40 is
    pitch in Hz, which upstream derives from a 1200-line LPCNet-style tracker
    that is not ported here; see ``pitch_hz`` below.

    Args:
        pitch_hz: constant substituted for the pitch feature. ``None`` (the
            default) uses the training mean, i.e. a normalised value of 0 —
            the least biased stand-in, and measurably closer to the C library
            than the 0 Hz that sherpa-onnx substitutes. Pass 0.0 for
            sherpa-compatible behaviour, or feed real per-frame pitch to
            :meth:`process` if you have an estimator.
    """

    def __init__(self, pitch_hz: float | None = None) -> None:
        self._window, self._mean, self._std = _load_coeffs()
        self._default_pitch: np.float32 = (
            self._mean[N_MEL] if pitch_hz is None else np.float32(pitch_hz)
        )
        self._mel = build_mel_filterbank()
        self._inv_std: npt.NDArray[np.float32] = (
            np.float32(1.0) / (self._std + np.float32(_EPS))
        ).astype(np.float32)
        # FIFO holds the WINDOW_SIZE - HOP_SIZE pre-emphasised samples that
        # precede the next hop, plus the raw (non-emphasised) tail the pitch
        # estimator needs.
        self._emph_tail: npt.NDArray[np.float32] = np.zeros(
            WINDOW_SIZE - HOP_SIZE, dtype=np.float32
        )
        self._last_sample: np.float32 = np.float32(0.0)

    def reset(self) -> None:
        self._emph_tail[:] = 0.0
        self._last_sample = np.float32(0.0)

    def _preemphasis(self, samples: npt.NDArray[np.float32]) -> npt.NDArray[np.float32]:
        """y[n] = x[n] - 0.97*x[n-1], with x[-1] carried from the previous call."""
        out = np.empty_like(samples)
        out[0] = samples[0] - np.float32(_PREEMPHASIS) * self._last_sample
        out[1:] = samples[1:] - np.float32(_PREEMPHASIS) * samples[:-1]
        self._last_sample = np.float32(samples[-1])
        return out

    def process(
        self, samples: npt.NDArray[np.float32], pitch_hz: npt.NDArray[np.float32] | None = None
    ) -> npt.NDArray[np.float32]:
        """Compute normalised features for ``samples``.

        Args:
            samples: int16-scale float samples, length a multiple of HOP_SIZE.
            pitch_hz: optional per-frame pitch in Hz (one value per hop).
                Defaults to the constant chosen in ``__init__``.

        Returns:
            (n_frames, 41) float32 normalised features.
        """
        samples = np.asarray(samples, dtype=np.float32)
        if samples.size % HOP_SIZE != 0:
            raise ValueError(f"expected a multiple of {HOP_SIZE} samples, got {samples.size}")
        n_frames = samples.size // HOP_SIZE
        if n_frames == 0:
            return np.zeros((0, FEATURE_DIM), dtype=np.float32)

        emph = self._preemphasis(samples)
        padded = np.concatenate([self._emph_tail, emph])
        self._emph_tail = padded[-(WINDOW_SIZE - HOP_SIZE) :].copy()

        # Frame k spans padded[k*HOP : k*HOP + WINDOW]; zero-pad to FFT_SIZE.
        frames = np.lib.stride_tricks.sliding_window_view(padded, WINDOW_SIZE)[
            :: HOP_SIZE
        ][:n_frames]
        feats = np.empty((n_frames, FEATURE_DIM), dtype=np.float32)
        # Reused across blocks; the tail past WINDOW_SIZE stays zero throughout,
        # which is the zero-padding to FFT_SIZE.
        buf = np.zeros((min(n_frames, _FFT_BLOCK_FRAMES), FFT_SIZE), dtype=np.float32)
        for lo in range(0, n_frames, _FFT_BLOCK_FRAMES):
            hi = min(lo + _FFT_BLOCK_FRAMES, n_frames)
            block = buf[: hi - lo]
            block[:, :WINDOW_SIZE] = frames[lo:hi] * self._window

            spectrum = np.fft.rfft(block, n=FFT_SIZE, axis=1)
            power = (spectrum.real**2 + spectrum.imag**2).astype(np.float32)
            mel_energy = (power @ self._mel) / np.float32(_POWER_NORMALIZER)
            feats[lo:hi, :N_MEL] = np.log(mel_energy + np.float32(_EPS))
        feats[:, N_MEL] = (
            self._default_pitch if pitch_hz is None else np.asarray(pitch_hz, dtype=np.float32)
        )

        feats -= self._mean
        feats *= self._inv_std
        return feats
