"""Speaker voice recognition service.

Stores per-user voice embeddings under ``/root/local/users/<name>/voice/`` and
recognizes speakers via cosine similarity. Embeddings are computed by a
configurable external API (see ``SPEAKER_EMBEDDING_API_URL``).

Audio preprocessing (Mono → Resample → [HighPass] → [NoiseReduce] → VAD →
[STOI gate] → RMS) runs ON THIS DEVICE (see ``audio_processors/``), next to the
mic. Only audio that passes the VAD + STOI intelligibility gate is uploaded; the
server is told to skip its own preprocessing (``preprocess=false``) and just
compute the embedding.

External API contract:
    POST {SPEAKER_EMBEDDING_API_URL}
    Headers: X-API-Key: {SPEAKER_EMBEDDING_API_KEY} (optional)
    Body:    {"audios_b64": ["<base64 WAV>", ...], "preprocess": false}
    Response: {"embedding": [float, float, ...]}  (single 1-D vector
              aggregated from all inputs, any dimension)

Storage layout per user::

    /root/local/users/<norm>/
        metadata.json           — SHARED identity (telegram_username, telegram_id,
                                   display_name). Same file face-enroll writes —
                                   merged on write, never overwritten blindly.
        voice/
            embedding.npy       — single L2-normalized aggregated vector [D].
                                   Mirrors perception-service's per-speaker storage so
                                   recognize uses the same per-chunk voting
                                   logic as perception-service's /recognize endpoint.
            metadata.json       — voice-specific (enrolled_at, updated_at,
                                   num_samples, sample_files, embedding_dim)
            sample_<ts>_<uuid>.wav  — source WAV files (16kHz mono)

Label normalization matches :class:`FaceRecognizer.normalize_label` so face /
voice / mood / wellbeing all share the same per-user folder for a person.

Registry of users with registered voices::

    /root/local/users/.voice_registry.json
"""

from __future__ import annotations

import base64
import io
import json
import logging
import os
import re
import shutil
import threading
import time
import uuid
import wave
from math import gcd
from pathlib import Path
from typing import Any, Iterable, Optional

import numpy as np
import requests

from hal import config
from hal.drivers.sensing.crypto import CryptoSession, resolve_public_key

logger = logging.getLogger("hal.voice.speaker")

# --- Storage layout (paths come from hal.config) ---
_USERS_DIR = Path(config.USERS_DIR)
_VOICE_SUBDIR = "voice"
_EMBEDDING_FILE = "embedding.npy"
_METADATA_FILE = "metadata.json"
_REGISTRY_FILE = _USERS_DIR / ".voice_registry.json"
_UNKNOWN_AUDIO_DIR = Path(config.SPEAKER_UNKNOWN_AUDIO_DIR)

# --- External embedding API (centralized in hal.config) ---
_API_URL = config.SPEAKER_EMBEDDING_API_URL
_API_KEY = config.SPEAKER_EMBEDDING_API_KEY
_API_TIMEOUT_S = config.SPEAKER_EMBEDDING_API_TIMEOUT_S
_MATCH_THRESHOLD = config.SPEAKER_MATCH_THRESHOLD
_ENROLL_CONSISTENCY_THRESHOLD = config.SPEAKER_ENROLL_CONSISTENCY_THRESHOLD

# --- Voice stranger clustering ---
# Assigns a stable "voiceprint_hash" (voice_<N> label) to every unknown voice
# so callers can track "same unknown speaker seen multiple times" without
# needing voiceprint_hash support from the embedding backend. Mirrors the
# face stranger tracker in faceid/perception.py.
_VOICE_STRANGERS_DIR = Path(
    os.environ.get("HAL_VOICE_STRANGERS_DIR", "/root/local/voice_strangers")
)
# All thresholds in this file use SCALED cosine in [0, 1] (`(raw + 1) / 2`).
# That includes MATCH / CONSISTENCY plus CLUSTER_MERGE — call sites convert the
# raw `embeds @ query` dot-product to scaled before comparing. Ordering:
# CLUSTER_MERGE < MATCH = CONSISTENCY, i.e. the merge gate (used at enroll time
# to pull fragmented clusters back together) is the loosest, and the
# recognize/match decision is strictest.
#
# Stranger re-appearance matching uses the SAME threshold as enrolled-user
# matching (SPEAKER_MATCH_THRESHOLD, i.e. self._match_threshold) — a returning
# unknown voice must clear the same bar as a known one, so there is NO separate
# stranger threshold. The trade-off is deliberate: the stricter bar means one
# speaker may fragment across several voice_<N> rather than two speakers sharing
# a cluster, and enroll's looser CLUSTER_MERGE gate pulls the fragments back
# together. A cluster is claimed WHOLE at enroll time, so a merged-in second
# voice is the more expensive error.
# Cap cluster count so disk doesn't grow unbounded. Oldest evicted first.
_MAX_VOICE_STRANGERS = int(
    os.environ.get("HAL_MAX_VOICE_STRANGERS", "50")
)
_VOICE_STRANGER_PREFIX = "voice_"
_VOICE_STRANGER_DIR_RE = re.compile(r"^voice_\d+$")

# Target sample rate for stored/enrolled audio (matches STT pipeline).
_TARGET_SR = 16000


class SpeakerRecognizerError(Exception):
    """Raised on invalid input or external API failure."""


class EmbeddingAPIUnavailableError(SpeakerRecognizerError):
    """Raised when the embedding API is unreachable / 5xx / protocol-broken.

    Distinct from audio-level rejections: the audio itself may be perfectly
    fine — the caller should retry rather than ask the user to re-record.
    Callers that batch over multiple samples MUST abort on this error
    instead of skipping the sample, to avoid misattributing an outage to
    bad audio and to avoid destructive cleanup of valid on-disk samples.
    """


class AudioGateRejectedError(SpeakerRecognizerError):
    """Raised when the on-device preprocessing gate (VAD/quality) rejects audio.

    Still a ``SpeakerRecognizerError`` so recognize() degrades to "unknown" and
    the enroll new-sample loop skips the clip — same as when perception used to
    return HTTP 400. Kept DISTINCT from a plain decode/corrupt failure so the
    enroll path does NOT delete a previously-accepted on-disk sample just
    because the gate got stricter (the gate is a moving target; a stored WAV is
    not "corrupt" merely because today's VAD trims it below threshold).
    """


# ---------------------------------------------------------------------------
# On-device audio preprocessing (moved from perception-service).
#
# The filter/VAD/normalize chain (Mono -> Resample -> [HighPass] ->
# [NoiseReduce] -> VAD -> [STOI gate] -> RMS) now runs HERE, next to the mic.
# Only audio that passes the VAD + STOI intelligibility gate is uploaded; the
# embedding server is then told to skip its own preprocessing (preprocess=false)
# and just compute the embedding. The composite processor is a lazily-built
# singleton because the silero VAD + STOI models load once and are reused across
# every enroll/recognize call.
# ---------------------------------------------------------------------------
_audio_processor: Optional[Any] = None
_audio_processor_lock = threading.Lock()


def _get_audio_processor() -> Any:
    """Lazily build + start the composite preprocessor (silero VAD loads once)."""
    global _audio_processor
    if _audio_processor is not None:
        return _audio_processor
    with _audio_processor_lock:
        if _audio_processor is None:
            # Heavy imports (torch / silero-vad) are deferred to first use so
            # module import stays light and lint doesn't require the VAD deps.
            from hal.drivers.voice.speaker_recognizer.audio_processors.factory import (
                AudioProcessorFactory,
            )

            factory = AudioProcessorFactory(
                target_sample_rate=config.SPEAKER_PROC_TARGET_SR,
                enable_mono=config.SPEAKER_PROC_ENABLE_MONO,
                enable_resample=config.SPEAKER_PROC_ENABLE_RESAMPLE,
                enable_high_pass=config.SPEAKER_PROC_ENABLE_HIGH_PASS,
                high_pass_cutoff_hz=config.SPEAKER_PROC_HIGH_PASS_CUTOFF_HZ,
                enable_noise_reduce=config.SPEAKER_PROC_ENABLE_NOISE_REDUCE,
                noise_reduce_stationary=config.SPEAKER_PROC_NOISE_STATIONARY,
                enable_vad=config.SPEAKER_PROC_ENABLE_VAD,
                vad_min_duration_sec=config.SPEAKER_PROC_VAD_MIN_DURATION_SEC,
                vad_min_voice_ratio=config.SPEAKER_PROC_VAD_MIN_VOICE_RATIO,
                vad_speech_prob_threshold=config.SPEAKER_PROC_VAD_SPEECH_PROB_THRESHOLD,
                enable_rms_normalize=config.SPEAKER_PROC_ENABLE_RMS_NORMALIZE,
                rms_target=config.SPEAKER_PROC_RMS_TARGET,
                enable_stoi=config.SPEAKER_PROC_ENABLE_STOI,
                stoi_model_path=config.SPEAKER_PROC_STOI_MODEL_PATH,
                stoi_threshold=config.SPEAKER_PROC_STOI_THRESHOLD,
                stoi_chunk_sec=config.SPEAKER_PROC_STOI_CHUNK_SEC,
            )
            try:
                proc = factory.create()
                proc.start()  # loads the silero VAD model
            except Exception as e:
                # Missing dep / model-load failure is systemic, not audio-level.
                # Raise EmbeddingAPIUnavailableError so enroll aborts cleanly
                # (never deletes on-disk samples) and recognize degrades to
                # "unknown" instead of crashing every turn with a 500.
                logger.error(
                    "Failed to init on-device audio preprocessor: %s", e)
                raise EmbeddingAPIUnavailableError(
                    f"audio preprocessor unavailable: {e}"
                ) from e
            _audio_processor = proc
            logger.info(
                "On-device audio preprocessor ready "
                "(mono=%s resample=%s highpass=%s noise=%s vad=%s stoi=%s rms=%s)",
                config.SPEAKER_PROC_ENABLE_MONO, config.SPEAKER_PROC_ENABLE_RESAMPLE,
                config.SPEAKER_PROC_ENABLE_HIGH_PASS, config.SPEAKER_PROC_ENABLE_NOISE_REDUCE,
                config.SPEAKER_PROC_ENABLE_VAD, config.SPEAKER_PROC_ENABLE_STOI,
                config.SPEAKER_PROC_ENABLE_RMS_NORMALIZE,
            )
    return _audio_processor


def _normalize_label(name: str) -> str:
    """Folder-safe lowercase label — matches FaceRecognizer.normalize_label.

    Keeping this rule identical to the face recognizer ensures that a person
    enrolled via face and via voice lands in the SAME per-user folder, and
    that mood/wellbeing/music-suggestion logs all refer to the same identity.
    """
    s = (name or "").strip().lower()
    s = re.sub(r"[^a-z0-9_-]+", "_", s)
    s = s.strip("_")
    return s[:64] if s else "person"


def _cosine_similarity(e1: np.ndarray, e2: np.ndarray) -> float:
    """Compute raw cosine similarity in range [-1, 1].

    Tolerates non-normalized inputs (unlike plain ``np.dot`` which requires
    pre-normalized vectors). The ``+ 1e-12`` guards against zero-norm inputs.
    Returns the confidence in range [0, 1].
    """
    raw_cos = float(np.dot(e1, e2) / (np.linalg.norm(e1) * np.linalg.norm(e2) + 1e-12))
    return (raw_cos + 1.0) / 2.0


def _l2(vec: np.ndarray) -> np.ndarray:
    """L2-normalize; return unchanged if norm is ~0."""
    arr = np.asarray(vec, dtype=np.float32)
    n = float(np.linalg.norm(arr))
    if n < 1e-12:
        return arr
    return (arr / n).astype(np.float32)


def _weighted_aggregate(
    embeddings: list[np.ndarray], *, power: float = 4.0
) -> np.ndarray:
    """Self-consistency weighted mean + L2-normalize.

    Mirrors ``perception-service.audio_preprocess.weighted_aggregate`` so pooling
    per-sample embeddings client-side produces a vector comparable with one
    the server would produce from the same samples — without the server-side
    artifact of concatenating multiple WAVs into a single waveform before
    VAD / chunking (which the /embed endpoint currently does).

    Each embedding is weighted by its cosine sim to the L2-normalized
    median centroid, mapped to [0, 1] and raised to ``power`` to sharpen —
    outlier samples (a rejected-but-still-on-disk recording, for example)
    contribute near zero to the final vector.
    """
    if not embeddings:
        raise SpeakerRecognizerError("no embeddings to aggregate")
    stack = np.stack([_l2(e) for e in embeddings], axis=0).astype(np.float32)
    centroid = _l2(np.median(stack, axis=0))
    sims = stack @ centroid
    sims01 = np.clip((sims + 1.0) / 2.0, 0.0, 1.0)
    weights = sims01.astype(np.float32) ** float(power)
    total = float(weights.sum())
    if total < 1e-9:
        weights = np.full(stack.shape[0], 1.0 / stack.shape[0], dtype=np.float32)
    else:
        weights = (weights / total).astype(np.float32)
    agg = (stack * weights[:, None]).sum(axis=0)
    return _l2(agg)

def _sample_origin(filename: str) -> str:
    """Parse the origin tag encoded in ``sample_<origin>_<ts>_<uuid>.wav``.

    Legacy files ``sample_<ts>_<uuid>.wav`` (no origin) → ``"unknown"``.
    """
    parts = filename.split("_", 2)
    if len(parts) >= 2 and parts[0] == "sample":
        candidate = parts[1]
        if candidate in ("mic", "telegram", "other"):
            return candidate
    return "unknown"


def _merge_shared_metadata(
    user_dir: Path,
    *,
    display_name: str | None = None,
    telegram_username: str | None = None,
    telegram_id: str | None = None,
) -> dict[str, Any]:
    """Merge identity fields into ``/root/local/users/<norm>/metadata.json``.

    This is the SAME file that :class:`FaceRecognizer` writes — we read,
    update only the provided fields, and write back. Empty/``None`` values
    never overwrite existing entries.
    """
    path = user_dir / "metadata.json"
    data: dict[str, Any] = {}
    if path.is_file():
        try:
            data = json.loads(path.read_text()) or {}
        except (json.JSONDecodeError, OSError):
            data = {}
    if display_name:
        data.setdefault("display_name", display_name)
    if telegram_username:
        data["telegram_username"] = telegram_username
    if telegram_id:
        data["telegram_id"] = telegram_id
    try:
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(json.dumps(data))
    except OSError as e:
        logger.warning("failed to write shared metadata %s: %s", path, e)
    return data


def _read_bytes(path: str) -> bytes:
    with open(path, "rb") as f:
        return f.read()


def _wav_bytes_to_float32_16k_mono(raw: bytes) -> np.ndarray:
    """Decode WAV bytes into float32 mono waveform at 16kHz."""
    try:
        import soundfile as sf  # type: ignore
    except ImportError as e:
        raise SpeakerRecognizerError(
            "soundfile is required for WAV processing"
        ) from e

    try:
        data, sr = sf.read(io.BytesIO(raw), dtype="float32")
    except Exception as e:
        raise SpeakerRecognizerError(f"cannot decode WAV: {e}") from e

    arr = np.asarray(data, dtype=np.float32)
    if arr.ndim == 2:
        arr = arr.mean(axis=1)
    elif arr.ndim != 1:
        raise SpeakerRecognizerError(f"unsupported WAV shape {arr.shape}")

    if sr != _TARGET_SR:
        try:
            from scipy.signal import resample_poly  # type: ignore
        except ImportError as e:
            raise SpeakerRecognizerError(
                "scipy is required for resampling"
            ) from e
        g = gcd(_TARGET_SR, int(sr))
        arr = resample_poly(arr, _TARGET_SR // g, int(sr) // g).astype(np.float32)
    return arr


def _float32_waveform_to_wav_bytes(waveform: np.ndarray) -> bytes:
    """Encode a float32 mono waveform into 16kHz PCM_16 WAV bytes."""
    try:
        import soundfile as sf  # type: ignore
    except ImportError as e:
        raise SpeakerRecognizerError(
            "soundfile is required for WAV processing"
        ) from e
    buf = io.BytesIO()
    sf.write(buf, np.asarray(waveform, dtype=np.float32), _TARGET_SR, format="WAV", subtype="PCM_16")
    return buf.getvalue()


def _ensure_wav_16k_mono(raw: bytes) -> bytes:
    """Normalize WAV bytes to 16kHz mono PCM_16 WAV bytes."""
    return _float32_waveform_to_wav_bytes(_wav_bytes_to_float32_16k_mono(raw))


def pcm16_bytes_to_wav(pcm_bytes: bytes, sample_rate: int = _TARGET_SR) -> bytes:
    """Wrap raw int16 mono PCM bytes in a WAV header."""
    buf = io.BytesIO()
    with wave.open(buf, "wb") as w:
        w.setnchannels(1)
        w.setsampwidth(2)
        w.setframerate(sample_rate)
        w.writeframes(pcm_bytes)
    return buf.getvalue()


# ==========================================================================
# ==  SPEAKER-DEBUG  —  TEMPORARY DIAGNOSTIC TRACER  —  REMOVE BEFORE DEPLOY ==
# ==========================================================================
# Everything tagged `SPEAKER-DEBUG` in this file is a throwaway diagnostic aid
# for tuning speaker recognition. It traces every recognize()/enroll() call to
# disk (input audio, query/enroll embeddings, full cosine-similarity breakdown,
# reject reason, or server error), mirroring the facial-emotion debug logs.
#
# TO REMOVE FOR PRODUCTION: delete this block and every line/call marked
# `SPEAKER-DEBUG` (grep -n "SPEAKER-DEBUG" this file — the __init__ line and the
# self._debug.* calls in recognize()/enroll()). It is self-contained: no other
# module or config file is touched. It is OFF by default (production-safe); set
# HAL_SPEAKER_DEBUG=true to enable it during development.
#
# Env knobs (all optional):
#   HAL_SPEAKER_DEBUG            "true" to enable (OFF by default)
#   HAL_SPEAKER_DEBUG_DIR        output root (default: ./speaker_logs next to this file)
#   HAL_SPEAKER_DEBUG_MAX_ENTRIES  per-kind dir cap, oldest pruned (default 1000; 0=unbounded)
#
# Layout (per call, named like the facial-emotion logs); <root> defaults to the
# `speaker_logs/` folder beside this file for fast inspection:
#   <root>/recognize/<ts>_<class>_<confidence>/       (class = enrolled name | stranger-<N> | unknown)
#   <root>/recognize/<ts>_FAIL-<reason>/              (too-short | too-silent | server-error | ...)
#   <root>/enroll/<ts>_<norm>_<cohesion>/             (cohesion = mean sim of kept samples to centroid)
#   <root>/enroll/<ts>_FAIL-<reason>/
# each dir holds: input.wav (raw) + preprocessed.wav (post VAD/STOI/RMS, what was
# uploaded) / sample_NN.wav, *.npy embeddings, result.json + profile.json.
# result.json carries the recognition decision plus the preprocessing metrics
# (incl. STOI score) and, on a gate reject, the reason; profile.json carries ONLY
# the per-stage latency + memory numbers, kept separate so neither file buries
# the other.
# NOTE: speaker_logs/ lands inside the source tree — don't commit it (the whole
# SPEAKER-DEBUG block is meant to be removed before deploy anyway).
def _debug_audio_stats(wav_bytes: bytes) -> tuple[Optional[float], Optional[float]]:
    """Best-effort (duration_s, rms) from WAV bytes for the trace. Never raises."""
    try:
        wf = _wav_bytes_to_float32_16k_mono(wav_bytes)
        dur = round(float(wf.shape[0]) / _TARGET_SR, 3)
        rms = round(float(np.sqrt(np.mean(wf.astype(np.float64) ** 2))), 6)
        return dur, rms
    except Exception:
        return None, None


def _debug_stranger_label(vp_hash: Optional[str]) -> str:
    """SPEAKER-DEBUG: 'voice_3' -> 'stranger-3' for readable trace-dir names.

    The internal id stays 'voice_<N>' everywhere (cluster folders, result.json
    voiceprint_hash); only the human-facing debug dir token is rewritten.
    """
    if not vp_hash:
        return "unknown"
    return vp_hash.replace(_VOICE_STRANGER_PREFIX, "stranger-", 1)


# ---------------------------------------------------------------------------
# SPEAKER-DEBUG: per-stage latency / CPU / memory profiler.
#
# Rides on the SAME trace the tracer already writes — same per-call dir, same
# HAL_SPEAKER_DEBUG switch, no extra env knob — but lands in its own
# `profile.json` rather than in result.json, which is already dense with the
# recognition decision.
#
# Stages form a TREE: a stage opened inside another becomes its child, so the
# containment is structural instead of a naming convention the reader has to
# parse. `preprocess` owns `silero_vad` / `stoi_gate`; summing the top level
# gives the call total without double-counting:
#
#   decode_input                 base64/file read + WAV normalize to 16k mono
#   preprocess                   the whole on-device chain
#     +- decode_wav                WAV bytes -> float32 waveform
#     +- processor_init            lazy build/start of the composite (models load)
#     +- mono / resample / high_pass / noise_reduce
#     +- silero_vad              << the silero-vad stage, profiled explicitly
#     +- stoi_gate               << the STOI gate, profiled explicitly
#     +- rms_normalize
#     +- encode_wav                cleaned waveform -> WAV bytes -> base64
#   embed_api                    the embedding call
#     +- request                 << the HTTP request itself
#     +- decode                    response parse + L2 normalize
#   load_enrolled / match_vote / stranger_cluster / save_input_wav
#
# Each node also carries `self_ms` (its own time minus its children's), so the
# cost of a parent's own glue is visible rather than hidden in the total.
#
# MEMORY. RSS is SAMPLED on a background thread (~20 ms) for the life of the
# call, and each stage reports the PEAK seen inside its own window. Endpoint-only
# sampling — read RSS at entry, read it again at exit — was the original approach
# and it is wrong for this pipeline: RSS moves only when the allocator asks the
# OS for pages or returns them, so a stage that allocates and frees within its
# own window reports 0.0, and a stage that happens to run when an earlier
# allocation is released reports a NEGATIVE cost. Real traces showed exactly
# that — `stoi_gate` at 1.7 s of ONNX inference reporting 0.00 MB on one call
# and -28.62 MB on the next. So three numbers are kept, each answering a
# different question:
#
#   rss_peak_delta_mb   peak-during-stage minus RSS at entry — "what did this
#                       stage cost at its worst". THIS is the memory number to
#                       read; it survives allocate-then-free.
#   rss_end_delta_mb    exit minus entry — "what did it keep". Legitimately
#                       negative when the allocator hands pages back.
#   rss_peak_mb         absolute high-water RSS during the stage.
#
# Caveat that no RSS-based method escapes: RSS is PROCESS-wide, so a concurrent
# HAL thread allocating during a stage lands in that stage's numbers. Sampling
# widens that exposure (it sees every spike in the window, not just the
# endpoints) in exchange for catching transient peaks at all. Read a single
# stage's memory as an upper bound, and prefer the shape across several calls.
#
# CPU. `cpu_ms` is process CPU time (`time.process_time()`) across the whole
# process, which is what catches the ONNX/torch intra-op thread pools — the
# work that makes `stoi_gate` expensive. `cpu_pct` is cpu_ms/wall_ms*100, so
# >100% means it used more than one core and ~0% means it was blocked, not
# working (`embed_api.request` should sit near zero — it is waiting on the
# network). `thread_cpu_ms` is the calling thread alone, so the gap between it
# and `cpu_ms` is roughly what the pools did. Both are process-wide in the same
# way RSS is: another busy HAL thread inflates `cpu_ms`.
#
# The RSS source is recorded as `rss_source`, because it changes what a delta means:
#
#   "psutil" / "statm"  CURRENT RSS — the accurate case (on-device Linux, and
#                       any box with psutil). A delta can be NEGATIVE when the
#                       allocator hands pages back.
#   "rusage"            HIGH-WATER RSS (ru_maxrss) — the macOS-without-psutil
#                       fallback. Monotonic, so deltas are growth-only: a stage
#                       reports > 0 only when it pushed the process past its
#                       previous peak. Good enough to spot which stage owns the
#                       footprint, useless for measuring memory being released.
#
# Repeated stages (enroll embeds many samples) are summed, with `calls` and
# `ms_max` alongside.
from contextlib import contextmanager, nullcontext  # SPEAKER-DEBUG

# Resolved once on first sample: "psutil" | "statm" | "rusage" | "none".
_debug_rss_mode: Optional[str] = None
_debug_rss_proc: Any = None


def _debug_rss_bytes() -> Optional[int]:
    """SPEAKER-DEBUG: process RSS in bytes; None if unmeasurable.

    See the block comment above for what each backend actually measures —
    ``rusage`` is a high-water mark, not current RSS.
    """
    global _debug_rss_mode, _debug_rss_proc
    if _debug_rss_mode is None:
        try:
            import psutil  # optional; present on-device via the HAL deps

            _debug_rss_proc = psutil.Process()
            _debug_rss_mode = "psutil"
        except Exception:
            if os.path.exists("/proc/self/statm"):
                _debug_rss_mode = "statm"
            else:
                _debug_rss_mode = "rusage"
    try:
        if _debug_rss_mode == "psutil":
            return int(_debug_rss_proc.memory_info().rss)
        if _debug_rss_mode == "statm":
            # field 1 = resident pages
            with open("/proc/self/statm", "r") as fh:
                return int(fh.read().split()[1]) * os.sysconf("SC_PAGE_SIZE")
        if _debug_rss_mode == "rusage":
            import resource
            import sys

            maxrss = resource.getrusage(resource.RUSAGE_SELF).ru_maxrss
            # ru_maxrss is bytes on Darwin/BSD, kilobytes on Linux.
            return int(maxrss) if sys.platform == "darwin" else int(maxrss) * 1024
    except Exception:
        return None
    return None


def _debug_peak_rss_bytes() -> Optional[int]:
    """SPEAKER-DEBUG: process high-water RSS (VmHWM). Linux only, else None."""
    try:
        with open("/proc/self/status", "r") as fh:
            for line in fh:
                if line.startswith("VmHWM:"):
                    return int(line.split()[1]) * 1024
    except Exception:
        pass
    return None


def _debug_mb(n: Optional[float]) -> Optional[float]:
    """SPEAKER-DEBUG: bytes -> MB, rounded. Passes None through."""
    return None if n is None else round(float(n) / (1024.0 * 1024.0), 2)


# SPEAKER-DEBUG: processor class -> stage name. The two gates the profile is
# really about are named after the model (silero_vad / stoi_gate), not the class.
_DEBUG_STAGE_NAMES: dict[str, str] = {
    "MonoConverter": "mono",
    "Resampler": "resample",
    "HighPassFilter": "high_pass",
    "NoiseReducer": "noise_reduce",
    "VoiceActivityFilter": "silero_vad",
    "SpeechIntelligibilityFilter": "stoi_gate",
    "RMSNormalizer": "rms_normalize",
}


class _StageNode:
    """SPEAKER-DEBUG: one node in the stage tree. Children nest under parents."""

    __slots__ = (
        "name", "calls", "ms", "ms_max", "cpu_ms", "thread_cpu_ms",
        "rss_peak_b", "rss_peak_delta_b", "rss_end_delta_b", "rss_after_b",
        "children", "_order",
    )

    def __init__(self, name: str) -> None:
        self.name = name
        self.calls = 0
        self.ms = 0.0
        self.ms_max = 0.0
        self.cpu_ms = 0.0
        self.thread_cpu_ms = 0.0
        self.rss_peak_b: Optional[int] = None
        self.rss_peak_delta_b: Optional[int] = None
        self.rss_end_delta_b: Optional[int] = None
        self.rss_after_b: Optional[int] = None
        self.children: dict[str, "_StageNode"] = {}
        self._order: list[str] = []

    def child(self, name: str) -> "_StageNode":
        """Get-or-create a child, preserving first-seen (pipeline) order."""
        node = self.children.get(name)
        if node is None:
            node = _StageNode(name)
            self.children[name] = node
            self._order.append(name)
        return node

    def kids(self) -> list["_StageNode"]:
        return [self.children[n] for n in self._order]

    def accumulate(
        self,
        ms: float,
        cpu_ms: float,
        thread_cpu_ms: float,
        rss_before: Optional[int],
        rss_after: Optional[int],
        rss_peak: Optional[int],
    ) -> None:
        self.calls += 1
        self.ms += ms
        self.ms_max = max(self.ms_max, ms)
        self.cpu_ms += cpu_ms
        self.thread_cpu_ms += thread_cpu_ms
        if rss_after is not None:
            self.rss_after_b = rss_after
            if rss_before is not None:
                self.rss_end_delta_b = (
                    (self.rss_end_delta_b or 0) + (rss_after - rss_before)
                )
        if rss_peak is not None:
            self.rss_peak_b = (
                rss_peak if self.rss_peak_b is None else max(self.rss_peak_b, rss_peak)
            )
            if rss_before is not None:
                # MAX, not sum: for a repeated stage the useful number is the
                # worst single occurrence ("how big did one call of this get"),
                # not a total that grows with the sample count.
                growth = max(0, rss_peak - rss_before)
                self.rss_peak_delta_b = (
                    growth if self.rss_peak_delta_b is None
                    else max(self.rss_peak_delta_b, growth)
                )

    def to_dict(self) -> dict[str, Any]:
        kids = self.kids()
        d: dict[str, Any] = {
            "stage": self.name,
            "ms": round(self.ms, 2),
            # This node's OWN time — the parent's glue, not its children's work.
            "self_ms": round(max(0.0, self.ms - sum(k.ms for k in kids)), 2),
            "cpu_ms": round(self.cpu_ms, 2),
            # >100% = used more than one core; ~0% = blocked, not working.
            "cpu_pct": (round(self.cpu_ms / self.ms * 100.0, 1) if self.ms > 0 else None),
            "thread_cpu_ms": round(self.thread_cpu_ms, 2),
            # THE memory number: peak inside this stage minus RSS at entry.
            "rss_peak_delta_mb": _debug_mb(self.rss_peak_delta_b),
            "rss_peak_mb": _debug_mb(self.rss_peak_b),
            # What it KEPT — legitimately negative when pages go back to the OS.
            "rss_end_delta_mb": _debug_mb(self.rss_end_delta_b),
            "rss_after_mb": _debug_mb(self.rss_after_b),
        }
        if self.calls != 1:
            d["calls"] = self.calls
            d["ms_max"] = round(self.ms_max, 2)
        if kids:
            d["children"] = [k.to_dict() for k in kids]
        return d


class _StageProfiler:
    """SPEAKER-DEBUG: nested per-stage latency / CPU / memory. Never raises."""

    # RSS is sampled on a background thread because endpoint-only sampling
    # misses any allocation freed before the stage exits — see the block
    # comment above for the traces that exposed it.
    _SAMPLE_INTERVAL_S = 0.02
    _MAX_SAMPLES = 20000        # ~400 s of sampling; backstop, not a real limit
    _MAX_LIFETIME_S = 300.0     # sampler self-terminates if a call never ends

    def __init__(self, label: str) -> None:
        self.label = label
        self._t0 = time.perf_counter()
        self._cpu0 = time.process_time()
        self._rss0 = _debug_rss_bytes()
        self._root = _StageNode(label)
        # Open-stage stack: a stage entered while another is open becomes its
        # child. This is what makes the output a tree instead of a flat list.
        self._stack: list[_StageNode] = [self._root]
        self._samples: list[tuple[float, int]] = []
        self._stop = threading.Event()
        if self._rss0 is not None:
            self._samples.append((self._t0, self._rss0))
            # Daemon: must never hold up interpreter shutdown. Self-terminates
            # on _MAX_LIFETIME_S so a call that dies before to_dict() can't
            # leak a sampler for the life of the process.
            threading.Thread(
                target=self._sample_loop,
                name="speaker-debug-rss",
                daemon=True,
            ).start()

    def _sample_loop(self) -> None:
        deadline = self._t0 + self._MAX_LIFETIME_S
        try:
            while not self._stop.wait(self._SAMPLE_INTERVAL_S):
                if (
                    len(self._samples) >= self._MAX_SAMPLES
                    or time.perf_counter() > deadline
                ):
                    return
                rss = _debug_rss_bytes()
                if rss is not None:
                    self._samples.append((time.perf_counter(), rss))
        except Exception:
            pass

    def close(self) -> None:
        """Stop the sampler. Idempotent."""
        self._stop.set()

    def _peak_between(
        self, t0: float, t1: float, *points: Optional[int]
    ) -> Optional[int]:
        """Highest RSS observed in [t0, t1], including the endpoint reads."""
        best: Optional[int] = None
        for p in points:
            if p is not None and (best is None or p > best):
                best = p
        for t, rss in list(self._samples):  # snapshot; sampler only appends
            if t0 <= t <= t1 and (best is None or rss > best):
                best = rss
        return best

    @contextmanager
    def stage(self, name: str):
        """Time + measure one stage. Records even when the body raises."""
        node = self._stack[-1].child(name)
        self._stack.append(node)
        t0 = time.perf_counter()
        cpu0 = time.process_time()
        thread0 = time.thread_time()
        rss_before = _debug_rss_bytes()
        try:
            yield
        finally:
            # `finally`, so a stage that REJECTS (VAD trims to nothing, STOI
            # below threshold) still reports its cost — a reject pays for the
            # same inference as a pass, and is exactly what we tune.
            t1 = time.perf_counter()
            cpu_ms = (time.process_time() - cpu0) * 1000.0
            thread_ms = (time.thread_time() - thread0) * 1000.0
            rss_after = _debug_rss_bytes()
            try:
                if self._stack and self._stack[-1] is node:
                    self._stack.pop()
                node.accumulate(
                    (t1 - t0) * 1000.0, cpu_ms, thread_ms,
                    rss_before, rss_after,
                    self._peak_between(t0, t1, rss_before, rss_after),
                )
            except Exception:  # a profiler must never break the service
                pass

    def to_dict(self) -> Optional[dict[str, Any]]:
        """JSON-safe profile for the trace's profile.json."""
        try:
            self.close()
            t1 = time.perf_counter()
            rss_end = _debug_rss_bytes()
            peak = self._peak_between(self._t0, t1, self._rss0, rss_end)
            total_ms = (t1 - self._t0) * 1000.0
            cpu_ms = (time.process_time() - self._cpu0) * 1000.0
            return {
                "total_ms": round(total_ms, 2),
                "cpu_ms": round(cpu_ms, 2),
                "cpu_pct": (round(cpu_ms / total_ms * 100.0, 1) if total_ms > 0 else None),
                "cpu_count": os.cpu_count(),
                "rss_source": _debug_rss_mode,
                "rss_sample_interval_ms": round(self._SAMPLE_INTERVAL_S * 1000.0, 1),
                "rss_samples": len(self._samples),
                "rss_start_mb": _debug_mb(self._rss0),
                "rss_end_mb": _debug_mb(rss_end),
                "rss_peak_mb": _debug_mb(peak),
                "rss_peak_delta_mb": _debug_mb(
                    None if (peak is None or self._rss0 is None) else peak - self._rss0
                ),
                "rss_end_delta_mb": _debug_mb(
                    None if (rss_end is None or self._rss0 is None)
                    else rss_end - self._rss0
                ),
                # Process high-water mark since start-up (Linux VmHWM) — a
                # whole-process number, unrelated to this call's own peak.
                "process_peak_rss_mb": _debug_mb(_debug_peak_rss_bytes()),
                "stages": [k.to_dict() for k in self._root.kids()],
            }
        except Exception as e:
            logger.debug("SPEAKER-DEBUG profile serialize failed: %s", e)
            return None

    def summary_line(self) -> str:
        """One-line `path=<ms>/<cpu%>/<+peak MB>` summary for the log."""
        try:
            parts = [f"total={(time.perf_counter() - self._t0) * 1000.0:.1f}ms"]

            def walk(node: _StageNode, prefix: str) -> None:
                for k in node.kids():
                    path = f"{prefix}{k.name}"
                    seg = f"{path}{'' if k.calls == 1 else f'x{k.calls}'}={k.ms:.1f}ms"
                    if k.ms > 0:
                        seg += f"/{k.cpu_ms / k.ms * 100.0:.0f}%cpu"
                    peak = _debug_mb(k.rss_peak_delta_b)
                    if peak is not None:
                        seg += f"/+{peak:.1f}MB"
                    parts.append(seg)
                    walk(k, path + ".")

            walk(self._root, "")
            return " ".join(parts)
        except Exception:
            return "<unavailable>"


class _SpeakerDebugTracer:
    """SPEAKER-DEBUG: writes per-call trace dirs. Fully self-contained; never raises."""

    def __init__(self) -> None:
        # SPEAKER-DEBUG: OFF by default (production-safe). Set HAL_SPEAKER_DEBUG=true
        # to enable during development — no .env edit needed, any env source works
        # (shell `export`, systemd `Environment=`, docker `-e`, the launch script).
        # Read once at construction, so restart HAL after changing it.
        self.enabled = os.environ.get(
            "HAL_SPEAKER_DEBUG", "false").lower() == "true"
        # Default: a `speaker_logs/` dir right next to this file, so traces are
        # trivial to inspect. Override with HAL_SPEAKER_DEBUG_DIR.
        _default_dir = Path(__file__).resolve().parent / "speaker_logs"
        self._base = Path(os.environ.get("HAL_SPEAKER_DEBUG_DIR", str(_default_dir)))
        try:
            self._max = int(os.environ.get("HAL_SPEAKER_DEBUG_MAX_ENTRIES", "1000"))
        except ValueError:
            self._max = 1000
        if self.enabled:
            # Prefer the source-tree dir; if it's read-only (device deploy),
            # fall back to a writable temp dir instead of silently disabling.
            if not self._try_mkdir(self._base):
                import tempfile
                fallback = Path(tempfile.gettempdir()) / "hal-speaker-debug"
                if self._try_mkdir(fallback):
                    logger.warning(
                        "SPEAKER-DEBUG: %s not writable — using %s",
                        self._base, fallback,
                    )
                    self._base = fallback
                else:
                    logger.warning("SPEAKER-DEBUG disabled (no writable dir)")
                    self.enabled = False
            if self.enabled:
                logger.info("SPEAKER-DEBUG tracing ON -> %s", self._base)

    @staticmethod
    def _try_mkdir(p: Path) -> bool:
        try:
            p.mkdir(parents=True, exist_ok=True)
            return True
        except OSError:
            return False

    @staticmethod
    def _stamp() -> str:
        now = time.time()
        return time.strftime("%Y%m%d-%H%M%S", time.localtime(now)) + f"-{int((now % 1.0) * 1e6):06d}"

    @staticmethod
    def _san(value: Any) -> str:
        s = re.sub(r"[^A-Za-z0-9_.-]+", "_", str(value)).strip("_")
        return s[:48] or "na"

    def record(
        self,
        kind: str,
        *,
        cls: Any = None,
        confidence: Optional[float] = None,
        reason: Optional[str] = None,
        result: Optional[dict[str, Any]] = None,
        wavs: Optional[dict[str, bytes]] = None,
        arrays: Optional[dict[str, Any]] = None,
        profile: Optional[dict[str, Any]] = None,
    ) -> None:
        if not self.enabled:
            return
        try:
            stamp = self._stamp()
            if reason:
                dname = f"{stamp}_FAIL-{self._san(reason)}"
            else:
                conf = f"{float(confidence):.2f}" if confidence is not None else "na"
                dname = f"{stamp}_{self._san(cls)}_{conf}"
            out = self._base / kind / dname
            out.mkdir(parents=True, exist_ok=True)

            payload: dict[str, Any] = {
                "timestamp": stamp,
                "ts": time.time(),
                "service": f"speaker.{kind}",
                "status": "failure" if reason else "prediction",
            }
            if reason:
                payload["reason"] = reason
            if result:
                payload.update(result)
            (out / "result.json").write_text(json.dumps(payload, indent=2, default=str))

            # Latency/memory goes in its OWN file — result.json is already dense
            # with the recognition decision, and mixing timings into it makes
            # both harder to read. Same dir, so a trace stays one unit.
            if profile:
                (out / "profile.json").write_text(
                    json.dumps(profile, indent=2, default=str)
                )

            for fn, wb in (wavs or {}).items():
                if wb:
                    try:
                        (out / fn).write_bytes(wb)
                    except OSError:
                        pass
            for fn, arr in (arrays or {}).items():
                if arr is not None:
                    try:
                        np.save(out / fn, np.asarray(arr))
                    except Exception:
                        pass
            self._prune(kind)
        except Exception as e:  # a debug tracer must never break the service
            logger.debug("SPEAKER-DEBUG trace failed: %s", e)

    def _prune(self, kind: str) -> None:
        if self._max <= 0:
            return
        try:
            kd = self._base / kind
            dirs = sorted((p for p in kd.iterdir() if p.is_dir()), key=lambda p: p.name)
            for old in dirs[: max(0, len(dirs) - self._max)]:
                shutil.rmtree(old, ignore_errors=True)
        except OSError:
            pass

    @staticmethod
    def classify_reason(err: Exception) -> str:
        """SPEAKER-DEBUG: map a recognize/enroll exception to a short FAIL slug."""
        if isinstance(err, EmbeddingAPIUnavailableError):
            return "server-error"
        msg = str(err).lower()
        # On-device preprocessing gate rejections (PreprocessRejected reason
        # codes are embedded in the message as "[<reason>]").
        if "empty_input" in msg:
            return "empty-input"
        if "vad_removed_all" in msg:
            return "no-voice"
        if "low_voice_ratio" in msg:
            return "low-voice"
        if "low_intelligibility" in msg:
            return "low-stoi"
        if "too_short" in msg or "too short" in msg:
            return "too-short"
        if "too silent" in msg:
            return "too-silent"
        if "not configured" in msg:
            return "api-not-configured"
        if "invalid base64" in msg or "cannot decode" in msg or "empty audio" in msg:
            return "bad-audio"
        return "embed-error"
# ===================  end SPEAKER-DEBUG tracer block  =====================


class SpeakerRecognizer:
    """Per-user voice embedding store with external-API embedding computation."""

    def __init__(
        self,
        api_url: Optional[str] = None,
        api_key: Optional[str] = None,
        users_dir: Optional[Path] = None,
        match_threshold: Optional[float] = None,
    ) -> None:
        self._api_url = api_url or _API_URL
        self._api_key = api_key or _API_KEY
        self._users_dir = Path(users_dir) if users_dir else _USERS_DIR
        self._match_threshold = (
            match_threshold if match_threshold is not None else _MATCH_THRESHOLD
        )
        self._mu = threading.Lock()

        self._debug = _SpeakerDebugTracer()  # SPEAKER-DEBUG (remove before deploy)
        # SPEAKER-DEBUG: last stranger-cluster match info, stashed by
        # _assign_voiceprint_hash so the recognize() trace can record the
        # cluster match score / re-appearance without changing that method's
        # return signature. Only ever written when self._debug.enabled.
        self._debug_stranger: Optional[dict[str, Any]] = None
        # SPEAKER-DEBUG: last on-device preprocessing snapshot (cleaned WAV +
        # VAD/STOI metrics), stashed by _prepare_wav_for_embedding so the
        # recognize() trace can log what the gate produced. Reset per recognize.
        self._debug_preproc: Optional[dict[str, Any]] = None
        # SPEAKER-DEBUG: per-call stage profiler (latency + RSS per stage).
        # THREAD-LOCAL, unlike the two snapshots above: recognize()/enroll() are
        # not serialized against each other, and a profile is an accumulating
        # list — two concurrent calls sharing one would interleave their stages
        # and report nonsense timings.
        self._debug_prof = threading.local()

        self._crypto: CryptoSession | None = None
        if config.DL_ENCRYPTION_ENABLED:
            public_key = resolve_public_key(config.DL_PUBLIC_KEY_URL, config.DL_API_KEY, config.DL_PUBLIC_KEY_FILE)
            if public_key is not None:
                self._crypto = CryptoSession(public_key)
                logger.info("Speaker recognizer: encryption enabled")
            elif config.DL_ENCRYPTION_REQUIRED:
                raise RuntimeError("Encryption required but no public key available")

        self._users_dir.mkdir(parents=True, exist_ok=True)
        _UNKNOWN_AUDIO_DIR.mkdir(parents=True, exist_ok=True)

        # Voice stranger clustering state — mirrors FaceRecognizer stranger
        # tracking. Persists to _VOICE_STRANGERS_DIR so reboots don't lose
        # the "same voice seen again" grouping.
        self._stranger_lock = threading.Lock()
        self._stranger_embeds: Optional[np.ndarray] = None  # [N, D] L2-normalized
        self._stranger_labels: Optional[np.ndarray] = None  # [N] str labels
        self._stranger_counter: int = 0
        _VOICE_STRANGERS_DIR.mkdir(parents=True, exist_ok=True)
        self._load_strangers()

        logger.info(
            "SpeakerRecognizer ready (api=%s, threshold=%.2f, users_dir=%s, strangers=%d)",
            self._api_url or "<unset>",
            self._match_threshold,
            self._users_dir,
            0 if self._stranger_embeds is None else len(self._stranger_embeds),
        )

    @property
    def available(self) -> bool:
        return bool(self._api_url)

    # ------------------------------------------------------------------ paths

    def _voice_dir(self, norm: str) -> Path:
        return self._users_dir / norm / _VOICE_SUBDIR

    def _embedding_path(self, norm: str) -> Path:
        return self._voice_dir(norm) / _EMBEDDING_FILE

    def _metadata_path(self, norm: str) -> Path:
        return self._voice_dir(norm) / _METADATA_FILE

    # ------------------------------------------------------------- registry

    def _load_registry(self) -> dict[str, Any]:
        if _REGISTRY_FILE.is_file():
            try:
                return json.loads(_REGISTRY_FILE.read_text())
            except (json.JSONDecodeError, OSError):
                pass
        return {}

    def _save_registry(self, registry: dict[str, Any]) -> None:
        try:
            _REGISTRY_FILE.parent.mkdir(parents=True, exist_ok=True)
            _REGISTRY_FILE.write_text(json.dumps(registry, indent=2))
        except OSError as e:
            logger.warning("failed to save voice registry: %s", e)

    def _update_registry(self, norm: str, meta: dict[str, Any]) -> None:
        with self._mu:
            reg = self._load_registry()
            reg[norm] = {
                "display_name": meta.get("display_name", norm),
                "telegram_username": meta.get("telegram_username", ""),
                "telegram_id": meta.get("telegram_id", ""),
                "has_telegram_identity": meta.get("has_telegram_identity", False),
                "enrollment_sources": meta.get("enrollment_sources", []),
                "last_enrollment_source": meta.get("last_enrollment_source", ""),
                "enrolled_at": meta.get("enrolled_at"),
                "updated_at": meta.get("updated_at"),
                "num_samples": meta.get("num_samples", 0),
                "embedding_dim": meta.get("embedding_dim", 0),
            }
            self._save_registry(reg)

    def _remove_from_registry(self, norm: str) -> None:
        with self._mu:
            reg = self._load_registry()
            if norm in reg:
                del reg[norm]
                self._save_registry(reg)

    # -------------------------------------------------------------- external

    def _call_embedding_api(
        self, audios_b64: list[str], *, return_chunks: bool = False
    ) -> np.ndarray:
        """POST audios to the embedding API.

        Returns the L2-normalized aggregated vector ``[D]`` by default. When
        ``return_chunks=True`` returns the matrix of per-chunk embeddings
        ``[M, D]`` instead — used by ``recognize()`` to do per-chunk voting
        against stored speakers (mirroring perception-service's /recognize logic).
        """
        if not self._api_url:
            raise SpeakerRecognizerError(
                "SPEAKER_EMBEDDING_API_URL not configured"
            )
        if not audios_b64:
            raise SpeakerRecognizerError("no audio to embed")

        logger.info(
            "Calling embedding API with %d audios at %s (return_chunks=%s)",
            len(audios_b64), self._api_url, return_chunks,
        )
        headers = {"Content-Type": "application/json"}
        if self._api_key:
            headers["X-API-Key"] = self._api_key

        body: dict[str, Any] = {
            "audios_b64": audios_b64,
            "preprocess": False,
        }
        if return_chunks:
            body["return_chunks"] = True

        # SPEAKER-DEBUG: `embed_api` = the whole call, `embed_api.request` = the
        # network round-trip alone, `embed_api.decode` = parse + L2 normalize.
        # Both record on failure too (the `finally` in _StageProfiler.stage), so
        # a timeout shows up as its full _API_TIMEOUT_S wait rather than vanishing.
        with self._debug_stage("embed_api"):
            try:
                with self._debug_stage("request"):
                    if self._crypto is not None:
                        resp = requests.post(
                            self._api_url,
                            data=self._crypto.wrap_http_request(json.dumps(body).encode()),
                            headers=headers,
                            timeout=_API_TIMEOUT_S,
                        )
                    else:
                        resp = requests.post(
                            self._api_url,
                            json=body,
                            headers=headers,
                            timeout=_API_TIMEOUT_S,
                        )
            except requests.RequestException as e:
                logger.warning("Embedding server unreachable at %s: %s", self._api_url, e)
                raise EmbeddingAPIUnavailableError(
                    f"embedding API unreachable: {e}"
                ) from e

            if resp.status_code != 200:
                logger.warning(
                    "Embedding server returned HTTP %d: %s",
                    resp.status_code, resp.text[:120],
                )
                # 5xx = server broken → transient, caller should retry.
                # 4xx = server rejected THIS audio (VAD, decode, etc.) → audio-level,
                # caller should skip this sample / re-record.
                if resp.status_code >= 500:
                    raise EmbeddingAPIUnavailableError(
                        f"embedding API {resp.status_code}: {resp.text[:200]}"
                    )
                raise SpeakerRecognizerError(
                    f"embedding API error {resp.status_code}: {resp.text[:200]}"
                )

            with self._debug_stage("decode"):
                try:
                    if self._crypto is not None:
                        payload = json.loads(self._crypto.unwrap_http_response(resp.content))
                    else:
                        payload = resp.json()
                except ValueError as e:
                    raise EmbeddingAPIUnavailableError(
                        f"embedding API returned non-JSON: {e}"
                    ) from e

                if return_chunks:
                    chunks = payload.get("chunk_embeddings")
                    if not chunks:
                        raise EmbeddingAPIUnavailableError(
                            "embedding API response missing 'chunk_embeddings'"
                        )
                    mat = np.asarray(chunks, dtype=np.float32)
                    if mat.ndim != 2 or mat.size == 0:
                        raise EmbeddingAPIUnavailableError(
                            f"chunk_embeddings must be a non-empty 2-D array, got shape {mat.shape}"
                        )
                    norms = np.linalg.norm(mat, axis=1, keepdims=True)
                    norms[norms < 1e-12] = 1.0
                    return (mat / norms).astype(np.float32)

                emb = payload.get("embedding")
                if emb is None:
                    raise EmbeddingAPIUnavailableError(
                        "embedding API response missing 'embedding'"
                    )

                vec = np.asarray(emb, dtype=np.float32)
                if vec.ndim != 1 or vec.size == 0:
                    raise EmbeddingAPIUnavailableError(
                        f"embedding must be a non-empty 1-D array, got shape {vec.shape}"
                    )

                norm = float(np.linalg.norm(vec))
                if norm == 0.0:
                    raise EmbeddingAPIUnavailableError("embedding has zero norm")
                return vec / norm

    def _prepare_wav_for_embedding(self, wav_bytes: bytes) -> list[str]:
        """Run the on-device preprocessing pipeline and wrap the cleaned WAV.

        HAL now runs the full filter/VAD/normalize chain locally (Mono →
        Resample → [HighPass] → [NoiseReduce] → VAD → [STOI] → RMS) — the
        pipeline that used to run inside perception-service, plus the HAL-only
        STOI intelligibility gate. Only audio that PASSES the gate
        is returned for upload; ``_call_embedding_api`` then asks the server to
        skip its own preprocessing (``preprocess=false``) and just compute the
        embedding on this cleaned waveform.

        The ``/embed`` endpoint still windows/chunks the waveform itself, so we
        pass the whole cleaned WAV as a single element and let the server slice
        it (client-side splitting would only add a lossy float32 → PCM_16
        round-trip per slice).

        Raises ``SpeakerRecognizerError`` (audio-level) when the pipeline
        rejects the clip — recognize() maps that to "unknown" and enroll()
        skips the sample, exactly as when the server returned HTTP 400. Note
        this is deliberately NOT ``EmbeddingAPIUnavailableError``: a rejected
        clip is a bad recording, not a server outage.
        """
        # Light submodule imports (Audio / PreprocessRejected) avoid pulling in
        # torch/silero at module import time — those load in _get_audio_processor.
        from hal.drivers.voice.speaker_recognizer.audio_processors.base import Audio
        from hal.drivers.voice.speaker_recognizer.audio_processors.exceptions import (
            PreprocessRejected,
        )

        # SPEAKER-DEBUG: `prof` is None unless tracing is on; every stage below
        # is wrapped so the whole chain (and each gate inside it) is profiled.
        prof: Optional[_StageProfiler] = getattr(self._debug_prof, "cur", None)
        with self._debug_stage("preprocess"):
            with self._debug_stage("decode_wav"):
                waveform = _wav_bytes_to_float32_16k_mono(wav_bytes)
            if waveform.shape[0] == 0:
                raise SpeakerRecognizerError("empty audio for embedding")

            # First call also loads silero-vad + the ONNX STOI session, which is
            # the single largest memory step in the pipeline — worth its own
            # stage so a cold call isn't mistaken for a per-clip cost.
            with self._debug_stage("processor_init"):
                processor = _get_audio_processor()
            try:
                audio_in = Audio(waveform=waveform, sample_rate=_TARGET_SR)
                if prof is not None:  # SPEAKER-DEBUG: per-stage walk of the chain
                    cleaned = self._debug_profiled_process(processor, audio_in, prof)
                else:
                    cleaned = processor.process(audio_in)
            except PreprocessRejected as e:
                # AudioGateRejectedError (not a plain SpeakerRecognizerError) so the
                # enroll path keeps previously-accepted on-disk samples instead of
                # deleting them when the gate gets stricter.
                err = AudioGateRejectedError(
                    f"audio rejected by preprocessing gate [{e.reason}]: {e}"
                )
                err.gate_detail = e.to_dict()  # SPEAKER-DEBUG: reason + VAD/STOI metrics
                raise err from e

            out = np.asarray(cleaned.waveform, dtype=np.float32)
            if out.shape[0] == 0:
                raise SpeakerRecognizerError("preprocessing produced empty audio")

            with self._debug_stage("encode_wav"):
                cleaned_wav = _float32_waveform_to_wav_bytes(out)
                payload = [base64.b64encode(cleaned_wav).decode("ascii")]
            if self._debug.enabled:  # SPEAKER-DEBUG
                self._debug_preproc = self._debug_preproc_snapshot(
                    out, cleaned_wav, processor
                )
        return payload

    def _debug_preproc_snapshot(  # SPEAKER-DEBUG (remove before deploy)
        self, cleaned: np.ndarray, cleaned_wav: bytes, processor: Any
    ) -> dict[str, Any]:
        """Snapshot the on-device preprocessing result for the recognize trace.

        Captures the cleaned/uploaded WAV plus its duration/RMS and the STOI
        gate's pass score (pulled off the SpeechIntelligibilityFilter stage, if
        present). Only called when debug is enabled.
        """
        dur = round(float(cleaned.shape[0]) / _TARGET_SR, 3)
        rms = (round(float(np.sqrt(np.mean(cleaned.astype(np.float64) ** 2))), 6)
               if cleaned.size else 0.0)
        stoi_score: Optional[float] = None
        stoi_threshold: Optional[float] = None
        for p in getattr(processor, "_processors", []):
            if type(p).__name__ == "SpeechIntelligibilityFilter":
                s = getattr(p, "last_score", float("nan"))
                stoi_score = None if (s is None or np.isnan(s)) else round(float(s), 4)
                stoi_threshold = getattr(p, "_threshold", None)
                break
        return {
            "cleaned_wav": cleaned_wav,          # popped into the trace's wavs
            "cleaned_duration_s": dur,
            "cleaned_rms": rms,
            "stoi_score": stoi_score,
            "stoi_threshold": stoi_threshold,
        }

    def _debug_preproc_parts(self) -> tuple[dict[str, bytes], Optional[dict[str, Any]]]:
        """SPEAKER-DEBUG: split the last preprocessing snapshot into
        (wav-attachments, json-safe metrics) for a recognize trace."""
        pp = self._debug_preproc or {}
        wavs: dict[str, bytes] = {}
        if pp.get("cleaned_wav"):
            wavs["preprocessed.wav"] = pp["cleaned_wav"]
        metrics = {k: v for k, v in pp.items() if k != "cleaned_wav"} or None
        return wavs, metrics

    # ------------------------------------ SPEAKER-DEBUG: stage profiling

    def _debug_profile_start(self, label: str) -> None:
        """SPEAKER-DEBUG: begin a per-call profile. No-op when tracing is off."""
        self._debug_prof.cur = _StageProfiler(label) if self._debug.enabled else None

    def _debug_stage(self, name: str) -> Any:
        """SPEAKER-DEBUG: context manager timing one stage; no-op when off.

        Used at every call site so the production path costs one attribute
        lookup and a ``nullcontext``.
        """
        prof = getattr(self._debug_prof, "cur", None)
        return prof.stage(name) if prof is not None else nullcontext()

    def _debug_profile_dict(self) -> Optional[dict[str, Any]]:
        """SPEAKER-DEBUG: finish the profile — log the summary, return the JSON.

        Passed as ``record(profile=...)`` at each trace point, which writes it
        to ``profile.json`` beside result.json in the same per-call dir.
        """
        prof = getattr(self._debug_prof, "cur", None)
        if prof is None:
            return None
        logger.info(
            "SPEAKER-DEBUG profile [%s]: %s", prof.label, prof.summary_line()
        )
        return prof.to_dict()

    def _debug_profiled_process(
        self, processor: Any, audio: Any, prof: _StageProfiler
    ) -> Any:
        """SPEAKER-DEBUG: run the preprocessing chain stage-by-stage.

        Mirrors ``CompositeAudioProcessor._process_impl`` exactly — same order,
        same ``processor.process(...)`` per stage, same exceptions — and only
        adds a timer around each one, so silero-vad and the STOI gate each get
        their own latency + memory numbers instead of one opaque "preprocess"
        total. Kept HERE rather than inside ``audio_processors/`` so the whole
        SPEAKER-DEBUG block stays removable without touching that package.
        Falls back to the plain composite call if the chain isn't introspectable.
        """
        stages = getattr(processor, "_processors", None)
        if not stages:
            with prof.stage("chain"):
                return processor.process(audio)
        result = audio
        for p in stages:
            cls_name = type(p).__name__
            # Leaf name only — the profiler nests it under whatever stage is
            # open (``preprocess``), so the tree carries the path.
            with prof.stage(_DEBUG_STAGE_NAMES.get(cls_name, cls_name.lower())):
                result = p.process(result)
        return result

    # ------------------------------------------------------------- metadata

    def _read_metadata(self, norm: str) -> dict[str, Any]:
        p = self._metadata_path(norm)
        if p.is_file():
            try:
                return json.loads(p.read_text())
            except (json.JSONDecodeError, OSError):
                pass
        return {}

    def _read_shared_metadata(self, norm: str) -> dict[str, Any]:
        """Read the top-level ``/root/local/users/<norm>/metadata.json``.

        Shared with FaceRecognizer — source of truth for telegram_* fields.
        """
        p = self._users_dir / norm / "metadata.json"
        if p.is_file():
            try:
                return json.loads(p.read_text()) or {}
            except (json.JSONDecodeError, OSError):
                pass
        return {}

    def _write_metadata(self, norm: str, meta: dict[str, Any]) -> None:
        self._metadata_path(norm).write_text(json.dumps(meta, indent=2))

    def _load_all_embeddings(self) -> dict[str, np.ndarray]:
        """Load every stored aggregated embedding [D] — source of truth for recognize().

        Mirrors perception-service's per-speaker storage: one L2-normalized vector
        per user. Recognize then runs per-chunk voting against these.
        """
        out: dict[str, np.ndarray] = {}
        if not self._users_dir.is_dir():
            return out
        for entry in sorted(self._users_dir.iterdir()):
            if not entry.is_dir() or entry.name.startswith("."):
                continue
            emb_path = self._voice_dir(entry.name) / _EMBEDDING_FILE
            if not emb_path.is_file():
                continue
            try:
                vec = np.load(emb_path).astype(np.float32)
                if vec.ndim != 1 or vec.size == 0:
                    continue
                n = float(np.linalg.norm(vec))
                if n < 1e-12:
                    continue
                out[entry.name] = (vec / n).astype(np.float32)
            except Exception as e:
                logger.warning("failed to load embedding for %s: %s", entry.name, e)
        return out

    # --------------------------------------------------------- public: enroll

    def enroll(
        self,
        name: str,
        wav_sources: Iterable[str],
        source_type: str = "base64",
        telegram_username: str = "",
        telegram_id: str = "",
        origin: str = "",
    ) -> dict[str, Any]:
        """Enroll or re-enroll a speaker.

        New sample WAVs are appended to the user's ``voice/`` folder and the
        embedding is (re)computed from ALL samples in the folder, producing a
        single aggregated representative vector.

        Identity (``telegram_username`` / ``telegram_id`` / display name) is
        merged into the SHARED ``/root/local/users/<norm>/metadata.json`` —
        the same file face-enroll writes to — so one person's identity is
        consistent across face, voice, mood, and wellbeing skills.

        Each sample is tagged with its origin (``"mic"`` or ``"telegram"``)
        so a user enrolled only via mic (no Telegram identity yet) can later
        be re-enrolled via Telegram without losing their earlier samples.

        Args:
            name: Display name (normalized to folder-safe lowercase).
            wav_sources: List of base64-encoded WAV data or filepaths.
            source_type: ``"base64"`` or ``"filepath"``.
            telegram_username: Optional Telegram @handle (e.g. ``chloe_92``).
            telegram_id: Optional numeric Telegram user ID.
            origin: ``"mic"`` / ``"telegram"`` / ``"other"``. Auto-derived
                (from presence of telegram_id/username) if empty.

        Returns:
            Metadata dict for the enrolled speaker (voice-specific + merged
            identity fields).
        """
        sources = list(wav_sources or [])
        if not sources:
            raise SpeakerRecognizerError("no audio provided")
        if source_type not in ("base64", "filepath"):
            raise SpeakerRecognizerError(
                f"invalid source_type {source_type!r}"
            )
        if not self.available:
            raise SpeakerRecognizerError(
                "embedding API not configured — set SPEAKER_EMBEDDING_API_URL"
            )

        # Infer origin from whether Telegram identity was supplied.
        if not origin:
            origin = (
                "telegram" if (telegram_username or telegram_id) else "mic"
            )
        origin = origin if origin in ("mic", "telegram", "other") else "other"

        # SPEAKER-DEBUG: per-call latency/memory profile. Enroll runs the
        # preprocessing chain + embedding call once per sample, so the shared
        # stages below aggregate (see `calls` / `ms_max` in the profile).
        self._debug_profile_start("enroll")

        norm = _normalize_label(name)
        logger.info(
            "Enrolling speaker: name=%r (norm=%r) new_samples=%d origin=%s tg_identity=%s",
            name, norm, len(sources), origin,
            bool(telegram_username or telegram_id),
        )
        user_dir = self._users_dir / norm
        user_dir.mkdir(parents=True, exist_ok=True)
        voice_dir = self._voice_dir(norm)
        voice_dir.mkdir(parents=True, exist_ok=True)

        # Persist shared identity early, even if embedding fails later.
        shared_identity = _merge_shared_metadata(
            user_dir,
            display_name=name.strip() or None,
            telegram_username=telegram_username or None,
            telegram_id=telegram_id or None,
        )

        # ------------------------------------------------------------
        # Strict policy: the voice/ folder ONLY contains audios that
        # contributed to the final embedding. So we do all the work in
        # memory first — validate, embed, filter — and ONLY THEN commit
        # surviving samples to disk. Any audio that fails validation or
        # the consistency filter never touches the folder.
        # ------------------------------------------------------------

        # Step 1 — Decode + normalize incoming audios (in-memory only).
        new_wavs: list[bytes] = []
        with self._debug_stage("decode_input"):  # SPEAKER-DEBUG
            for src in sources:
                if source_type == "filepath":
                    raw = _read_bytes(src)
                else:
                    try:
                        raw = base64.b64decode(src)
                    except Exception as e:
                        raise SpeakerRecognizerError(f"invalid base64: {e}") from e
                if not raw:
                    raise SpeakerRecognizerError("empty audio")
                new_wavs.append(_ensure_wav_16k_mono(raw))

        # Step 2 — Compute embedding per NEW wav BEFORE writing to disk.
        # _prepare_wav_for_embedding raises on too-short/silent audio;
        # _call_embedding_api raises SpeakerRecognizerError for 4xx
        # (audio-level reject — skip this sample) or
        # EmbeddingAPIUnavailableError for network/5xx (bubble up so the
        # whole enroll aborts cleanly and nothing on disk is touched).
        new_embeddings: list[tuple[bytes, np.ndarray]] = []
        per_sample_errors: list[tuple[int, str]] = []
        for idx, wb in enumerate(new_wavs):
            try:
                payload = self._prepare_wav_for_embedding(wb)
                emb = self._call_embedding_api(payload)
                new_embeddings.append((wb, emb))
            except EmbeddingAPIUnavailableError:
                raise
            except SpeakerRecognizerError as e:
                per_sample_errors.append((idx, str(e)))
                logger.warning(
                    "Enroll: rejected new sample #%d — %s (not saved to disk)",
                    idx, e,
                )

        if not new_embeddings:
            # Surface the actual reason from perception-service (VAD reject text, etc.)
            # or from local gates (too short / silent) — no hardcoded summary.
            if self._debug.enabled:  # SPEAKER-DEBUG
                self._debug.record(
                    "enroll", reason="no-valid-samples",
                    result={
                        "name": norm, "origin": origin, "num_new": len(new_wavs),
                        "per_sample_errors": [
                            {"index": i, "error": m} for i, m in per_sample_errors
                        ],
                    },
                    profile=self._debug_profile_dict(),
                )
            if len(per_sample_errors) == 1:
                raise SpeakerRecognizerError(per_sample_errors[0][1])
            details = "; ".join(
                f"sample #{i}: {msg}" for i, msg in per_sample_errors
            )
            raise SpeakerRecognizerError(f"no valid new samples — {details}")

        # Step 2b — Pull all WAVs from clusters that should contribute samples
        # to this enrollment. Two paths into the cluster set, both unioned:
        #   (a) Explicit claim — any source path whose parent dir is a
        #       ``voice_<N>`` cluster. The caller (LLM via OpenClaw skill)
        #       may pass only one path from a cluster even when the cluster
        #       has more sibling WAVs; we treat passing any path as claiming
        #       the WHOLE cluster, so no audio gets stranded if the LLM
        #       happens to surface only one sample per turn.
        #   (b) Centroid match — any cluster whose stored centroid sits within
        #       merge_threshold of the query mean. Captures distance / volume
        #       drift where the same person got fragmented across voice_<N>
        #       clusters with low pairwise centroid similarity, but each one
        #       still aligns with the new enrollment audio.
        # Filepath sources only — base64 has no path to resolve. Samples
        # pulled in go through the same consistency filter (Step 5) as any
        # other sample, so false matches get their outliers dropped, and
        # _drop_consumed_clusters at the tail cleans up every consumed
        # cluster regardless of how many samples survived.
        if source_type == "filepath" and new_embeddings:
            # SCALED cosine in [0, 1] — same unit as the MATCH threshold.
            # 0.625 sits below the EER band so same-person-different-distance
            # gets absorbed; the Step 5 consistency filter below catches false
            # merges from loose matches.
            merge_threshold = float(
                os.environ.get("HAL_CLUSTER_MERGE_THRESHOLD", "0.625")
            )

            # (a) Path-based claim — collect cluster names from sources.
            try:
                unknown_root = _UNKNOWN_AUDIO_DIR.resolve()
            except OSError:
                unknown_root = None
            claimed_hashes: set[str] = set()
            if unknown_root is not None:
                for src in sources:
                    try:
                        resolved = Path(src).resolve()
                        resolved.relative_to(unknown_root)
                    except (OSError, ValueError):
                        continue
                    parent_name = resolved.parent.name
                    if _VOICE_STRANGER_DIR_RE.match(parent_name):
                        claimed_hashes.add(parent_name)

            # (b) Centroid-based match.
            query_mean = _weighted_aggregate(
                [emb for _wb, emb in new_embeddings]
            )
            matched_hashes = set(self._match_stranger_clusters(
                query_mean, merge_threshold,
            ))

            consume_hashes = sorted(claimed_hashes | matched_hashes)
            if claimed_hashes:
                logger.info(
                    "Cluster claim: source paths claimed %d cluster(s) %s",
                    len(claimed_hashes), sorted(claimed_hashes),
                )

            merged_added = 0
            for h in consume_hashes:
                cluster_dir_path = _UNKNOWN_AUDIO_DIR / h
                if not cluster_dir_path.is_dir():
                    continue
                for wav in sorted(cluster_dir_path.glob("*.wav")):
                    wav_str = str(wav)
                    if wav_str in sources:
                        continue  # caller already passed this path in
                    try:
                        raw = _read_bytes(wav_str)
                    except Exception as e:
                        logger.warning(
                            "Cluster pull: cannot read %s — %s", wav_str, e,
                        )
                        continue
                    try:
                        wb = _ensure_wav_16k_mono(raw)
                        payload = self._prepare_wav_for_embedding(wb)
                        emb = self._call_embedding_api(payload)
                    except EmbeddingAPIUnavailableError:
                        raise
                    except SpeakerRecognizerError as e:
                        logger.info(
                            "Cluster pull: skip %s — %s", wav.name, e,
                        )
                        continue
                    new_wavs.append(wb)
                    new_embeddings.append((wb, emb))
                    sources.append(wav_str)
                    merged_added += 1
            if consume_hashes:
                logger.info(
                    "Cluster pull: %d WAV(s) from %d cluster(s) %s "
                    "(claimed=%d, centroid-matched=%d, threshold=%.2f)",
                    merged_added, len(consume_hashes), consume_hashes,
                    len(claimed_hashes), len(matched_hashes - claimed_hashes),
                    merge_threshold,
                )

        # Step 3 — Load EXISTING samples on disk + compute their embeddings.
        # Two failure modes, handled separately:
        #   a) _prepare_wav_for_embedding fails → the WAV file itself is
        #      corrupt / silent / too short. Safe to delete so the folder
        #      doesn't carry a permanently broken sample.
        #   b) _call_embedding_api fails → the server rejected or is down.
        #      NEVER delete: the file was previously accepted and may be
        #      fine once the API recovers. EmbeddingAPIUnavailableError
        #      also aborts the whole enroll so we don't proceed with a
        #      partial view of existing samples.
        existing_on_disk = sorted(voice_dir.glob("sample_*.wav"))
        existing_embs: dict[Path, np.ndarray] = {}
        for p in existing_on_disk:
            try:
                payload = self._prepare_wav_for_embedding(p.read_bytes())
            except EmbeddingAPIUnavailableError:
                # Preprocessor unavailable (systemic) — abort, never delete.
                raise
            except AudioGateRejectedError as e:
                # The gate (VAD/quality) rejected a previously-accepted sample.
                # The gate is a moving target, so KEEP the file — deleting it
                # would silently shrink an enrollment as thresholds change.
                logger.warning(
                    "Enroll: skipping existing sample %s — gate rejected (%s), file kept",
                    p.name, e,
                )
                continue
            except SpeakerRecognizerError as e:
                # Genuine decode/corrupt failure — safe to delete.
                logger.warning(
                    "Enroll: removing broken existing sample %s — %s",
                    p.name, e,
                )
                try:
                    p.unlink()
                except OSError as ose:
                    logger.warning("Enroll: failed to delete %s: %s", p, ose)
                continue
            try:
                existing_embs[p] = self._call_embedding_api(payload)
            except EmbeddingAPIUnavailableError:
                raise
            except SpeakerRecognizerError as e:
                logger.warning(
                    "Enroll: skipping existing sample %s — server rejected (%s), file kept",
                    p.name, e,
                )

        # Step 4 — Reference = the LATEST NEW wav (user's most recent intent
        # wins). All other samples (new + existing) are scored against it.
        ref_wb, ref_emb = new_embeddings[-1]

        kept_new: list[tuple[bytes, np.ndarray]] = [(ref_wb, ref_emb)]
        dropped_new = 0
        for wb, emb in new_embeddings[:-1]:
            sim = _cosine_similarity(emb, ref_emb)
            if sim >= _ENROLL_CONSISTENCY_THRESHOLD:
                kept_new.append((wb, emb))
            else:
                dropped_new += 1
                logger.info(
                    "Enroll: dropped new sample (sim=%.2f < %.2f) — not written to disk",
                    sim, _ENROLL_CONSISTENCY_THRESHOLD,
                )

        kept_existing: list[tuple[Path, np.ndarray]] = []
        dropped_existing: list[tuple[Path, float]] = []
        for p, emb in existing_embs.items():
            sim = _cosine_similarity(emb, ref_emb)
            if sim >= _ENROLL_CONSISTENCY_THRESHOLD:
                kept_existing.append((p, emb))
            else:
                dropped_existing.append((p, sim))

        # Step 5 — Delete dropped EXISTING samples from disk.
        for p, sim in dropped_existing:
            try:
                p.unlink()
                logger.info(
                    "Enroll: removed stale existing sample %s (sim=%.2f < %.2f)",
                    p.name, sim, _ENROLL_CONSISTENCY_THRESHOLD,
                )
            except OSError as e:
                logger.warning("Enroll: failed to delete %s: %s", p, e)

        # Step 6 — Commit kept NEW wavs to disk. Stable millisecond prefix
        # ensures lex sort = chronological order for later reference picks.
        # Tiny sleep between writes keeps timestamp uniqueness for the rare
        # case where the same ms boundary is hit.
        written_new_paths: list[Path] = []
        for wb, _emb in kept_new:
            fname = f"sample_{origin}_{int(time.time() * 1000)}_{uuid.uuid4().hex[:8]}.wav"
            fpath = voice_dir / fname
            fpath.write_bytes(wb)
            written_new_paths.append(fpath)

        # Step 7 — Aggregate the per-sample embeddings we already computed
        # in Steps 2 + 3 instead of issuing one more /embed call with every
        # kept WAV. Bundling all samples into one server call would concat
        # them into a single waveform before VAD/chunking — a sample-level
        # boundary loss that differs from what perception-service's own register()
        # does (preprocess each file separately, then pool chunks). Each
        # per-sample embedding is already L2-normalized, so the self-
        # consistency weighted mean below matches the server's aggregation
        # math, minus the cross-file concat artifact.
        kept_embeddings = [emb for _p, emb in kept_existing] + [
            emb for _wb, emb in kept_new
        ]
        embedding = _weighted_aggregate(kept_embeddings)
        np.save(self._embedding_path(norm), embedding)

        # Re-read the folder — it now contains ONLY samples that were valid
        # AND passed the consistency filter (kept existing + written new).
        all_samples = sorted(voice_dir.glob("sample_*.wav"))

        logger.info(
            "Enroll committed: new_written=%d new_rejected=%d existing_kept=%d existing_dropped=%d total_on_disk=%d dim=%d",
            len(written_new_paths), dropped_new,
            len(kept_existing), len(dropped_existing),
            len(all_samples), int(embedding.shape[0]),
        )

        # Update voice metadata + registry.
        existing = self._read_metadata(norm)
        now_iso = time.strftime("%Y-%m-%dT%H:%M:%S%z") or time.strftime(
            "%Y-%m-%dT%H:%M:%S"
        )
        enrollment_sources = sorted(
            {_sample_origin(p.name) for p in all_samples} | {origin}
        )
        meta: dict[str, Any] = {
            "name": norm,
            "display_name": shared_identity.get("display_name")
                or existing.get("display_name")
                or name.strip()
                or norm,
            "telegram_username": shared_identity.get("telegram_username", ""),
            "telegram_id": shared_identity.get("telegram_id", ""),
            "has_telegram_identity": bool(
                shared_identity.get("telegram_id")
                or shared_identity.get("telegram_username")
            ),
            "enrollment_sources": enrollment_sources,
            "last_enrollment_source": origin,
            "enrolled_at": existing.get("enrolled_at", now_iso),
            "updated_at": now_iso,
            "num_samples": len(all_samples),
            "sample_files": [p.name for p in all_samples],
            "sample_origins": {p.name: _sample_origin(p.name) for p in all_samples},
            "embedding_dim": int(embedding.shape[0]),
        }
        self._write_metadata(norm, meta)
        self._update_registry(norm, meta)

        # Drop any stranger clusters whose WAVs were just enrolled. Keeping
        # them would leave stale centroids that re-label the now-known
        # speaker as voice_<N> on any recognition below the main threshold.
        if source_type == "filepath":
            self._drop_consumed_clusters(sources)

        logger.info(
            "Enrolled speaker '%s' — %d total samples, dim=%d",
            norm,
            meta["num_samples"],
            meta["embedding_dim"],
        )
        if self._debug.enabled:  # SPEAKER-DEBUG
            # cohesion = mean scaled-cosine of every kept sample to the final
            # aggregated vector — a single "how tight is this enrollment" number.
            try:
                cohesion = round(
                    float(np.mean([_cosine_similarity(e, embedding) for e in kept_embeddings])), 4
                )
            except Exception:
                cohesion = None
            self._debug.record(
                "enroll", cls=norm, confidence=cohesion,
                wavs={
                    f"sample_new_{i:02d}.wav": wb
                    for i, (wb, _e) in enumerate(kept_new)
                },
                arrays={
                    "embedding.npy": embedding,
                    "kept_embeddings.npy": np.stack([_l2(e) for e in kept_embeddings], axis=0),
                },
                result={
                    "name": norm, "cohesion": cohesion, "origin": origin,
                    "consistency_threshold": _ENROLL_CONSISTENCY_THRESHOLD,
                    "num_new": len(new_wavs),
                    "num_kept_new": len(kept_new),
                    "num_new_rejected_by_server": len(per_sample_errors),
                    "num_new_dropped_consistency": dropped_new,
                    "num_existing_kept": len(kept_existing),
                    "num_existing_dropped": len(dropped_existing),
                    "num_samples_total": meta["num_samples"],
                    "embedding_dim": meta["embedding_dim"],
                    "per_sample_errors": [
                        {"index": i, "error": m} for i, m in per_sample_errors
                    ],
                },
                # Stages aggregate across every sample this enroll embedded.
                profile=self._debug_profile_dict(),
            )
        return meta

    def drop_stranger_cluster(self, label: str) -> bool:
        """Drop a single stranger cluster by label.

        Removes the centroid row from the in-memory tables (persisting the
        reduced tables), then ``rmtree`` the on-disk cluster sub-dir.
        Returns ``True`` if anything was removed, ``False`` when the label
        wasn't known and no dir existed (route uses that for 404).
        """
        if not label or not _VOICE_STRANGER_DIR_RE.match(label):
            return False
        removed_centroid = False
        with self._stranger_lock:
            if (
                self._stranger_labels is not None
                and self._stranger_embeds is not None
                and len(self._stranger_labels) > 0
            ):
                mask = np.array(
                    [lbl != label for lbl in self._stranger_labels],
                    dtype=bool,
                )
                if not mask.all():
                    self._stranger_embeds = self._stranger_embeds[mask]
                    self._stranger_labels = self._stranger_labels[mask]
                    self._save_strangers()
                    removed_centroid = True
        removed_dir = False
        cluster_dir = _UNKNOWN_AUDIO_DIR / label
        if cluster_dir.is_dir():
            try:
                shutil.rmtree(cluster_dir)
                removed_dir = True
            except OSError as e:
                logger.warning(
                    "drop_stranger_cluster %s: rmtree failed: %s", label, e,
                )
        if removed_centroid or removed_dir:
            logger.info(
                "drop_stranger_cluster %s (centroid=%s, dir=%s)",
                label, removed_centroid, removed_dir,
            )
        return removed_centroid or removed_dir

    def _drop_consumed_clusters(self, wav_paths: list[str]) -> None:
        """Remove stranger clusters whose WAVs were consumed by enroll().

        Looks at each ``wav_path``'s parent dir — if it's a ``voice_<N>``
        sub-dir inside ``SPEAKER_UNKNOWN_AUDIO_DIR``, that cluster is now
        redundant (the speaker is known) and we drop both:
          - centroid row from ``_stranger_embeds`` / ``_stranger_labels``,
          - the cluster sub-dir on disk.

        Safe no-op if the caller passed paths from outside the cluster tree
        (e.g. Telegram enroll writes into a session temp dir).
        """
        try:
            unknown_root = _UNKNOWN_AUDIO_DIR.resolve()
        except OSError:
            return
        consumed: set[str] = set()
        for p in wav_paths:
            try:
                resolved = Path(p).resolve()
                resolved.relative_to(unknown_root)
            except (OSError, ValueError):
                continue
            parent_name = resolved.parent.name
            if _VOICE_STRANGER_DIR_RE.match(parent_name):
                consumed.add(parent_name)
        if not consumed:
            return

        with self._stranger_lock:
            if (
                self._stranger_labels is not None
                and self._stranger_embeds is not None
                and len(self._stranger_labels) > 0
            ):
                mask = np.array(
                    [lbl not in consumed for lbl in self._stranger_labels],
                    dtype=bool,
                )
                if not mask.all():
                    self._stranger_embeds = self._stranger_embeds[mask]
                    self._stranger_labels = self._stranger_labels[mask]
                    self._save_strangers()
                    logger.info(
                        "Enroll: dropped %d stranger centroid(s) %s after enrollment",
                        int((~mask).sum()), sorted(consumed),
                    )

        for label in consumed:
            cluster_dir = _UNKNOWN_AUDIO_DIR / label
            if not cluster_dir.is_dir():
                continue
            try:
                shutil.rmtree(cluster_dir)
                logger.info("Enroll: removed cluster dir %s", cluster_dir)
            except OSError as e:
                logger.warning(
                    "Enroll: failed to remove cluster dir %s: %s", cluster_dir, e,
                )

    # --------------------------------------------------------- public: remove

    def remove(self, name: str) -> bool:
        """Delete the user's voice folder (embedding + samples + voice metadata).

        Other per-user data (face photos, mood, wellbeing, ...) is preserved —
        we only touch the ``voice/`` subdir. The SHARED identity file
        ``/root/local/users/<norm>/metadata.json`` (telegram_username,
        telegram_id) is left untouched because face-enroll and other skills
        may still depend on it.
        """
        norm = _normalize_label(name)
        voice_dir = self._voice_dir(norm)
        if not voice_dir.is_dir():
            self._remove_from_registry(norm)
            return False
        try:
            shutil.rmtree(voice_dir)
        except OSError as e:
            logger.warning("failed to remove voice dir for %s: %s", norm, e)
            return False
        self._remove_from_registry(norm)
        logger.info("Removed speaker '%s'", norm)
        return True

    # ------------------------------------------------------ public: recognize

    def _debug_safe_assign_hash(  # SPEAKER-DEBUG (remove before deploy)
        self,
        query_chunks: np.ndarray,
        wav_bytes: bytes,
        *,
        resolved_name: str,
        is_match: bool,
        num_enrolled: int,
        source_type: str,
        saved_path: str,
    ) -> Optional[str]:
        """SPEAKER-DEBUG wrapper around _assign_voiceprint_hash.

        Stranger clustering can raise on an embedding-dimension mismatch between
        the stored voice_strangers store and the current backend (e.g. the
        192-vs-256 concatenate error when the embedding model changed). That
        raise happens BEFORE recognize()'s trace hook, so the failure otherwise
        never lands in the logs. This wrapper captures it as a FAIL trace with
        the input audio + dims + error, and degrades to no-cluster so recognize
        returns 'unknown' instead of crashing. Original call was just
        `self._assign_voiceprint_hash(query_chunks)`.
        """
        try:
            return self._assign_voiceprint_hash(query_chunks) or None
        except Exception as e:
            logger.warning("Recognize: voiceprint hash assignment failed — %s", e)
            if self._debug.enabled:
                dur, rms = _debug_audio_stats(wav_bytes)
                self._debug.record(
                    "recognize", reason="stranger-assign-error",
                    wavs={"input.wav": wav_bytes},
                    arrays={"input_chunks.npy": query_chunks},
                    result={
                        "error": repr(e), "name": resolved_name, "match": is_match,
                        "num_enrolled": num_enrolled,
                        "num_query_chunks": int(query_chunks.shape[0]),
                        "embedding_dim": int(query_chunks.shape[1]),
                        "duration_s": dur, "rms": rms, "source_type": source_type,
                        "unknown_audio_path": saved_path,
                    },
                )
            return None

    def recognize(
        self,
        wav_source: str,
        source_type: str = "base64",
    ) -> dict[str, Any]:
        """Recognize a speaker from a single WAV audio.

        Returns a dict with:

        ``name``
            matched user's normalized label, or ``"unknown"``
        ``confidence``
            best match confidence in ``[0, 1]``
        ``match``
            whether the best confidence exceeds ``match_threshold``
        ``unknown_audio_path``
            path to the audio saved under the unknown-audio dir (always set —
            so the skill can reuse the path for later enrollment)
        ``candidates``
            top-3 ``(name, confidence)`` pairs for debugging
        """
        if source_type not in ("base64", "filepath"):
            raise SpeakerRecognizerError(
                f"invalid source_type {source_type!r}"
            )

        logger.info(
            "Recognize start: source_type=%s source=%s",
            source_type,
            wav_source if source_type == "filepath" else f"<base64 {len(wav_source)}B>",
        )

        # SPEAKER-DEBUG: start the per-call latency/memory profile FIRST, so the
        # decode below is inside it. Reset per call, like the snapshots after it.
        self._debug_profile_start("recognize")

        with self._debug_stage("decode_input"):
            if source_type == "filepath":
                raw = _read_bytes(wav_source)
            else:
                try:
                    raw = base64.b64decode(wav_source)
                except Exception as e:
                    raise SpeakerRecognizerError(f"invalid base64: {e}") from e
            if not raw:
                raise SpeakerRecognizerError("empty audio")

            wav_bytes = _ensure_wav_16k_mono(raw)
        # SPEAKER-DEBUG: clear the per-call stranger snapshot so a trace can
        # never report the PREVIOUS call's cluster score when this call's
        # clustering is skipped (empty/zero-norm chunks) or fails.
        self._debug_stranger = None
        self._debug_preproc = None  # SPEAKER-DEBUG: reset per-call preprocessing snapshot

        with self._debug_stage("save_input_wav"):
            saved_path = self._save_incoming_audio(wav_bytes)

        if not self.available:
            logger.warning("Embedding server not configured — set SPEAKER_EMBEDDING_API_URL or DL_BACKEND_URL")
            if self._debug.enabled:  # SPEAKER-DEBUG
                self._debug.record(
                    "recognize", reason="api-not-configured",
                    wavs={"input.wav": wav_bytes},
                    result={
                        "source_type": source_type,
                        "error": "embedding API not configured",
                    },
                    profile=self._debug_profile_dict(),
                )
            return {
                "name": "unknown",
                "confidence": 0.0,
                "match": False,
                "unknown_audio_path": saved_path,
                "voiceprint_hash": None,
                "candidates": [],
                "error": "embedding API not configured",
            }

        try:
            payload = self._prepare_wav_for_embedding(wav_bytes)
            # Per-chunk query embeddings — same per-chunk granularity that
            # perception-service's /recognize uses internally, so per-chunk voting
            # below produces apples-to-apples confidence.
            query_chunks = self._call_embedding_api(
                payload, return_chunks=True
            )  # [M, D]
        except SpeakerRecognizerError as e:
            logger.warning(
                "Recognize: embedding failed for %s — %s", saved_path, e,
            )
            if self._debug.enabled:  # SPEAKER-DEBUG
                dur, rms = _debug_audio_stats(wav_bytes)
                fail_result = {
                    "source_type": source_type, "error": str(e),
                    "duration_s": dur, "rms": rms,
                    "vad_min_duration_s": config.SPEAKER_PROC_VAD_MIN_DURATION_SEC,
                    "unknown_audio_path": saved_path,
                }
                # On-device preprocessing-gate rejection (VAD / STOI / quality):
                # attach the structured reason + metrics (incl. stoi_score).
                gate_detail = getattr(e, "gate_detail", None)
                if gate_detail is not None:
                    fail_result["preprocessing_reject"] = gate_detail
                self._debug.record(
                    "recognize", reason=self._debug.classify_reason(e),
                    wavs={"input.wav": wav_bytes},
                    result=fail_result,
                    # Latency/memory up to the failure — a gate reject still paid
                    # for silero-vad (and STOI, if it got that far).
                    profile=self._debug_profile_dict(),
                )
            return {
                "name": "unknown",
                "confidence": 0.0,
                "match": False,
                "unknown_audio_path": saved_path,
                "voiceprint_hash": None,
                "candidates": [],
                "error": str(e),
            }

        logger.info(
            "Recognize: query embedding chunks=%d dim=%d saved=%s",
            int(query_chunks.shape[0]), int(query_chunks.shape[1]), saved_path,
        )

        with self._debug_stage("load_enrolled"):
            known = self._load_all_embeddings()
        if not known:
            # No enrolled users — every voice is unknown. Still assign a
            # stable cluster hash so repeat speakers can be tracked before
            # anyone is enrolled.
            logger.info("Recognize: no enrolled users — unknown + cluster-only path")
            with self._debug_stage("stranger_cluster"):
                vp_hash = self._debug_safe_assign_hash(  # SPEAKER-DEBUG (was: _assign_voiceprint_hash)
                    query_chunks, wav_bytes, resolved_name="unknown", is_match=False,
                    num_enrolled=0, source_type=source_type, saved_path=saved_path,
                )
            saved_path = self._move_to_cluster(saved_path, vp_hash)
            logger.info(
                "Recognize result: name=unknown confidence=0.00 cluster=%s path=%s",
                vp_hash or "(none)", saved_path,
            )
            if self._debug.enabled:  # SPEAKER-DEBUG
                dur, rms = _debug_audio_stats(wav_bytes)
                st = self._debug_stranger or {}
                st_score = st.get("score")
                pp_wavs, pp_metrics = self._debug_preproc_parts()
                self._debug.record(
                    # No enrolled users → dir named by the stranger cluster and
                    # its match score (how sure this is the same returning voice).
                    "recognize",
                    cls=_debug_stranger_label(vp_hash),
                    confidence=(st_score if st_score is not None else 0.0),
                    wavs={"input.wav": wav_bytes, **pp_wavs},
                    arrays={
                        "input_chunks.npy": query_chunks,
                        "input_embedding.npy": _l2(query_chunks.mean(axis=0)),
                    },
                    result={
                        "name": "unknown", "match": False,
                        "threshold": self._match_threshold,
                        "voiceprint_hash": vp_hash or None,
                        "stranger_match_score": (round(st_score, 4) if st_score is not None else None),
                        "stranger_reappeared": st.get("reappeared"),
                        "closest_cluster": st.get("closest_label"),
                        "num_stranger_clusters": st.get("num_clusters"),
                        "num_enrolled": 0,
                        "num_query_chunks": int(query_chunks.shape[0]),
                        "embedding_dim": int(query_chunks.shape[1]),
                        "candidates": [], "source_type": source_type,
                        "duration_s": dur, "rms": rms,
                        "preprocessing": pp_metrics,
                        "unknown_audio_path": saved_path,
                    },
                    profile=self._debug_profile_dict(),
                )
            return {
                "name": "unknown",
                "confidence": 0.0,
                "match": False,
                "unknown_audio_path": saved_path,
                "voiceprint_hash": vp_hash or None,
                "candidates": [],
            }

        # Per-chunk voting (mirrors perception-service.recognize line 614-645):
        # for each query chunk, pick the highest-confidence speaker, record
        # one vote and one confidence sample. Winner = most votes, tiebreak
        # by avg confidence. Returned confidence = avg of winner's votes.
        with self._debug_stage("match_vote"):
            names = list(known.keys())
            ref_matrix = np.stack([known[n] for n in names], axis=0)  # [K, D]
            sims = query_chunks @ ref_matrix.T                         # [M, K] raw cos
            confs = (sims + 1.0) / 2.0                                  # mapped [0, 1]
            best_idx = confs.argmax(axis=1)                             # [M]
            best_conf_per_chunk = confs[np.arange(confs.shape[0]), best_idx]

            vote_count: dict[str, int] = {}
            conf_sum: dict[str, float] = {}
            for k_idx, c in zip(best_idx.tolist(), best_conf_per_chunk.tolist()):
                n = names[k_idx]
                vote_count[n] = vote_count.get(n, 0) + 1
                conf_sum[n] = conf_sum.get(n, 0.0) + float(c)

            # Tiebreak by avg confidence so a 1-vote winner with high conf
            # never beats a 5-vote one — votes dominate.
            ranked = sorted(
                vote_count.keys(),
                key=lambda n: (vote_count[n], conf_sum[n] / vote_count[n]),
                reverse=True,
            )
            best_name = ranked[0]
            best_conf = conf_sum[best_name] / vote_count[best_name]
            scores = [
                (n, conf_sum[n] / vote_count[n], vote_count[n]) for n in ranked
            ]

        is_match = best_conf >= self._match_threshold
        resolved_name = best_name if is_match else "unknown"

        # Full per-speaker breakdown — lets operator see why a near-miss
        # happened (e.g. speaker_a scored 0.68 with 5 votes vs speaker_b 0.64
        # with 4 votes against threshold 0.70 → both lose, tag as unknown).
        scores_str = ", ".join(
            f"{n}={c:.3f}(v={v})" for n, c, v in scores[:5]
        )
        logger.info(
            "Recognize scores: threshold=%.2f match=%s -> name=%s | %s",
            self._match_threshold, is_match, resolved_name, scores_str,
        )

        # Only assign a stranger cluster hash for unknowns — known speakers
        # already have a stable identity (their name).
        with self._debug_stage("stranger_cluster"):
            vp_hash = None if is_match else self._debug_safe_assign_hash(  # SPEAKER-DEBUG (was: _assign_voiceprint_hash)
                query_chunks, wav_bytes, resolved_name=resolved_name, is_match=is_match,
                num_enrolled=len(known), source_type=source_type, saved_path=saved_path,
            )
        # Move WAV into per-cluster sub-dir so later inspection can group
        # samples by cluster. Known-speaker WAVs stay in the flat dir.
        if vp_hash:
            saved_path = self._move_to_cluster(saved_path, vp_hash)

        logger.info(
            "Recognize result: name=%s confidence=%.3f match=%s cluster=%s path=%s",
            resolved_name, best_conf, is_match, vp_hash or "(none)", saved_path,
        )
        result: dict[str, Any] = {
            "name": resolved_name,
            "confidence": round(best_conf, 4),
            "match": is_match,
            "unknown_audio_path": saved_path,
            "voiceprint_hash": vp_hash,
            "candidates": [
                {"name": n, "confidence": round(c, 4), "votes": v}
                for n, c, v in scores[:3]
            ],
        }
        # Surface identity fields on match.
        if is_match:
            shared = self._read_shared_metadata(best_name)
            result["display_name"] = shared.get("display_name", best_name)
            result["telegram_username"] = shared.get("telegram_username", "")
            result["telegram_id"] = shared.get("telegram_id", "")
            result["has_telegram_identity"] = bool(
                shared.get("telegram_id") or shared.get("telegram_username")
            )
        if self._debug.enabled:  # SPEAKER-DEBUG
            dur, rms = _debug_audio_stats(wav_bytes)
            # Full per-chunk comparison across EVERY enrolled speaker. Voting keeps
            # only each chunk's argmax winner, so a speaker that never wins a chunk
            # gets 0 votes and vanishes from `candidates` — even though it WAS
            # compared on every chunk. Capture the whole [chunk x speaker] matrix
            # so you can see the losers and why each chunk voted the way it did.
            _confs = confs.tolist()  # [M chunks][K speakers], scaled cosine [0, 1]
            per_chunk_scores = [
                {
                    "chunk": ci,
                    "winner": names[int(best_idx[ci])],
                    "scores": {names[k]: round(_confs[ci][k], 4) for k in range(len(names))},
                }
                for ci in range(len(_confs))
            ]
            speaker_summary = {
                names[k]: {
                    "votes": int((best_idx == k).sum()),
                    "mean_conf": round(float(confs[:, k].mean()), 4),
                    "max_conf": round(float(confs[:, k].max()), 4),
                }
                for k in range(len(names))
            }
            dbg_result = {
                "name": resolved_name, "match": is_match,
                # nearest-enrolled similarity (the winning vote), regardless of match
                "confidence": round(best_conf, 4),
                "threshold": self._match_threshold,
                "voiceprint_hash": vp_hash,
                "num_enrolled": len(known),
                "num_query_chunks": int(query_chunks.shape[0]),
                "embedding_dim": int(query_chunks.shape[1]),
                "enrolled_speakers": names,
                # EVERY enrolled speaker (incl. 0-vote losers): votes + mean/max sim.
                "speaker_summary": speaker_summary,
                # Each chunk vs every speaker + which one that chunk voted for.
                "per_chunk_scores": per_chunk_scores,
                # Vote-winners only — the actual decision breakdown.
                "candidates": [
                    {"name": n, "confidence": round(c, 4), "votes": v}
                    for n, c, v in scores
                ],
                "source_type": source_type,
                "duration_s": dur, "rms": rms,
                "preprocessing": self._debug_preproc_parts()[1],
                "unknown_audio_path": saved_path,
            }
            if is_match:
                # Enrolled → dir = <name>_<nearest-enrolled score>.
                dbg_cls, dbg_conf = resolved_name, best_conf
            else:
                # Stranger → dir = stranger-<N>_<cluster match score>, and record
                # the re-appearance detail alongside the nearest-enrolled miss.
                st = self._debug_stranger or {}
                st_score = st.get("score")
                dbg_cls = _debug_stranger_label(vp_hash)
                dbg_conf = st_score if st_score is not None else 0.0
                dbg_result.update({
                    "stranger_match_score": (round(st_score, 4) if st_score is not None else None),
                    "stranger_reappeared": st.get("reappeared"),
                    "closest_cluster": st.get("closest_label"),
                    "num_stranger_clusters": st.get("num_clusters"),
                })
            self._debug.record(
                "recognize", cls=dbg_cls, confidence=dbg_conf,
                wavs={"input.wav": wav_bytes, **self._debug_preproc_parts()[0]},
                arrays={
                    "input_chunks.npy": query_chunks,
                    "input_embedding.npy": _l2(query_chunks.mean(axis=0)),
                    # [chunks x speakers] scaled cosine; columns = enrolled_speakers order.
                    "chunk_scores.npy": confs,
                },
                result=dbg_result,
                # Per-stage latency + RSS delta -> profile.json (silero-vad and
                # the STOI gate each get their own entry under `stages`).
                profile=self._debug_profile_dict(),
            )
        return result

    # ------------------------------------------------------------ public: get

    def get_meta(self, name: str) -> Optional[dict[str, Any]]:
        """Return the full enrollment meta for one user, or None if not enrolled.

        Mirrors the per-row shape of :meth:`list_registered` but skips the
        registry walk. Used for idempotent retries on the enroll route — when
        the caller passes paths that have already been consumed, we can return
        the existing meta instead of erroring out.
        """
        norm = _normalize_label(name)
        if not self._embedding_path(norm).is_file():
            return None
        voice_meta = self._read_metadata(norm)
        shared_meta = self._read_shared_metadata(norm)
        tg_username = shared_meta.get(
            "telegram_username", voice_meta.get("telegram_username", "")
        )
        tg_id = shared_meta.get(
            "telegram_id", voice_meta.get("telegram_id", "")
        )
        return {
            "name": norm,
            "display_name": shared_meta.get("display_name")
            or voice_meta.get("display_name", norm),
            "telegram_username": tg_username,
            "telegram_id": tg_id,
            "has_telegram_identity": bool(tg_username or tg_id),
            "enrollment_sources": voice_meta.get("enrollment_sources", []),
            "last_enrollment_source": voice_meta.get(
                "last_enrollment_source", ""
            ),
            "num_samples": voice_meta.get("num_samples", 0),
            "embedding_dim": voice_meta.get("embedding_dim", 0),
            "enrolled_at": voice_meta.get("enrolled_at"),
            "updated_at": voice_meta.get("updated_at"),
            "sample_files": voice_meta.get("sample_files", []),
            "sample_origins": voice_meta.get("sample_origins", {}),
        }

    # ----------------------------------------------------------- public: list

    def list_registered(self) -> list[dict[str, Any]]:
        """Return users who have a registered voice (embedding file exists).

        Backed by the registry file but cross-verified with on-disk state so
        stale registry rows are skipped. Telegram identity is read fresh from
        the shared ``metadata.json`` on every call so renames propagate.

        Each entry includes ``enrollment_sources`` (e.g. ``["mic"]``,
        ``["telegram"]`` or ``["mic", "telegram"]``) and
        ``has_telegram_identity`` — so the skill can tell whether a mic-only
        user still needs to be linked to a Telegram account for DM targeting.
        """
        reg = self._load_registry()
        out: list[dict[str, Any]] = []
        for norm in sorted(reg.keys()):
            if not self._embedding_path(norm).is_file():
                continue
            voice_meta = self._read_metadata(norm)
            shared_meta = self._read_shared_metadata(norm)
            tg_username = shared_meta.get(
                "telegram_username", voice_meta.get("telegram_username", "")
            )
            tg_id = shared_meta.get(
                "telegram_id", voice_meta.get("telegram_id", "")
            )
            out.append(
                {
                    "name": norm,
                    "display_name": shared_meta.get("display_name")
                    or voice_meta.get("display_name", norm),
                    "telegram_username": tg_username,
                    "telegram_id": tg_id,
                    "has_telegram_identity": bool(tg_username or tg_id),
                    "enrollment_sources": voice_meta.get(
                        "enrollment_sources", []
                    ),
                    "last_enrollment_source": voice_meta.get(
                        "last_enrollment_source", ""
                    ),
                    "num_samples": voice_meta.get("num_samples", 0),
                    "embedding_dim": voice_meta.get("embedding_dim", 0),
                    "enrolled_at": voice_meta.get("enrolled_at"),
                    "updated_at": voice_meta.get("updated_at"),
                    "sample_files": voice_meta.get("sample_files", []),
                    "sample_origins": voice_meta.get("sample_origins", {}),
                }
            )
        return out

    # ------------------------------------ public: identity-focused methods

    def get_telegram_id(self, name: str) -> str | None:
        """Return ``telegram_id`` for a user, or ``None`` if not set.

        Mirrors :meth:`FaceRecognizer.get_telegram_id` so any skill wanting
        to DM a person after voice recognition can use a single lookup.
        """
        norm = _normalize_label(name)
        meta = self._read_shared_metadata(norm)
        val = meta.get("telegram_id") or ""
        return val or None

    def get_telegram_username(self, name: str) -> str | None:
        norm = _normalize_label(name)
        meta = self._read_shared_metadata(norm)
        val = meta.get("telegram_username") or ""
        return val or None

    def lookup_by_telegram_id(self, telegram_id: str) -> str | None:
        """Reverse-lookup: given a Telegram user ID, return the norm label.

        Useful when a Telegram turn arrives and the skill wants to decide
        whether the sender already has a voice profile before enrolling.
        """
        if not telegram_id:
            return None
        reg = self._load_registry()
        for norm, entry in reg.items():
            if entry.get("telegram_id") == telegram_id:
                return norm
        return None

    def update_identity(
        self,
        name: str,
        telegram_username: str = "",
        telegram_id: str = "",
    ) -> dict[str, Any]:
        """Attach / update Telegram identity on an existing voice profile.

        Use this when a user enrolled by mic first (no Telegram info) later
        introduces themselves from Telegram — we can link the two without
        re-uploading audio or recomputing the embedding.
        """
        norm = _normalize_label(name)
        user_dir = self._users_dir / norm
        if not self._embedding_path(norm).is_file():
            raise SpeakerRecognizerError(
                f"no voice profile for '{norm}' — call enroll first"
            )
        shared = _merge_shared_metadata(
            user_dir,
            display_name=name.strip() or None,
            telegram_username=telegram_username or None,
            telegram_id=telegram_id or None,
        )
        # Refresh mirrored fields in voice metadata + registry.
        voice_meta = self._read_metadata(norm)
        voice_meta["telegram_username"] = shared.get("telegram_username", "")
        voice_meta["telegram_id"] = shared.get("telegram_id", "")
        voice_meta["has_telegram_identity"] = bool(
            shared.get("telegram_id") or shared.get("telegram_username")
        )
        voice_meta["display_name"] = shared.get(
            "display_name", voice_meta.get("display_name", norm)
        )
        now_iso = time.strftime("%Y-%m-%dT%H:%M:%S%z") or time.strftime(
            "%Y-%m-%dT%H:%M:%S"
        )
        voice_meta["updated_at"] = now_iso
        self._write_metadata(norm, voice_meta)
        self._update_registry(norm, voice_meta)
        logger.info(
            "Linked Telegram identity to '%s' (username=%s, id=%s)",
            norm, telegram_username, telegram_id,
        )
        return voice_meta

    def reset_all(self) -> int:
        """Delete every registered voice profile.

        Mirrors :meth:`FaceRecognizer.reset_enrolled`. Only the ``voice/``
        subdir of each user is removed — the shared ``metadata.json``
        (telegram identity) is preserved because face / mood / wellbeing
        still depend on it.
        """
        count = 0
        reg = self._load_registry()
        for norm in list(reg.keys()):
            if self.remove(norm):
                count += 1
        # Best-effort: walk disk too in case registry was stale.
        if self._users_dir.is_dir():
            for entry in self._users_dir.iterdir():
                if not entry.is_dir() or entry.name.startswith("."):
                    continue
                voice_dir = self._voice_dir(entry.name)
                if voice_dir.is_dir():
                    try:
                        shutil.rmtree(voice_dir)
                        count += 1
                    except OSError as e:
                        logger.warning("reset_all: failed to drop %s: %s", voice_dir, e)
        # Clear registry file.
        with self._mu:
            self._save_registry({})
        logger.info("reset_all: cleared %d voice profiles", count)
        return count

    # --------------------------------------------------------------- helpers

    def _save_incoming_audio(self, wav_bytes: bytes) -> str:
        """Save the incoming recognize() WAV to the unknown-audio dir.

        We always save — even on a match — so that skills have a stable path
        to reuse for follow-up enrollment flows.
        """
        _UNKNOWN_AUDIO_DIR.mkdir(parents=True, exist_ok=True)
        fname = (
            f"incoming_{int(time.time() * 1000)}_{uuid.uuid4().hex[:8]}.wav"
        )
        fpath = _UNKNOWN_AUDIO_DIR / fname
        try:
            fpath.write_bytes(wav_bytes)
        except OSError as e:
            logger.warning("failed to save incoming audio: %s", e)
            return ""
        return str(fpath)

    def _move_to_cluster(self, saved_path: str, vp_hash: Optional[str]) -> str:
        """Move a saved WAV into a per-cluster sub-dir, return the new path.

        Called right after voiceprint_hash is assigned so later tools (web UI,
        diagnostic scripts) can list all audio for a given cluster via a
        single directory listing. No-op when hash is empty or file missing —
        known-speaker WAVs stay in the flat _UNKNOWN_AUDIO_DIR.
        """
        if not vp_hash or not saved_path:
            return saved_path
        src = Path(saved_path)
        if not src.exists():
            return saved_path
        try:
            cluster_dir = src.parent / vp_hash
            cluster_dir.mkdir(parents=True, exist_ok=True)
            dst = cluster_dir / src.name
            src.rename(dst)
            return str(dst)
        except OSError as e:
            logger.warning(
                "move %s to cluster %s failed: %s", saved_path, vp_hash, e,
            )
            return saved_path

    # ------------------------------------------------- voice stranger clustering

    def _assign_voiceprint_hash(self, query_chunks: np.ndarray) -> str:
        """Return a stable voice_<N> label for an unknown voice.

        Aggregates the per-chunk query embeddings into one L2-normalized
        vector, then compares against saved stranger centroids. A match
        (cosine >= self._match_threshold — the SAME bar as enrolled-user
        matching) reuses the existing label; otherwise a new label is
        allocated and persisted.

        Consumers don't call this directly — recognize() stamps the hash
        into its response when the speaker is unknown.
        """
        if query_chunks is None or len(query_chunks) == 0:
            return ""
        agg = query_chunks.mean(axis=0)
        norm = float(np.linalg.norm(agg))
        if norm == 0.0:
            return ""
        agg = agg / norm

        with self._stranger_lock:
            best_sim_pre = None  # captured for the "new cluster" log path
            best_label_pre: Optional[str] = None
            breakdown_pre = ""
            if self._stranger_embeds is not None and len(self._stranger_embeds) > 0:
                # Both sides L2-normalized → dot product is raw cosine [-1, 1].
                # Convert to scaled [0, 1] so the threshold sits in the same
                # unit as MATCH / CONSISTENCY elsewhere in the file.
                raw_sims = self._stranger_embeds @ agg
                sims = (raw_sims + 1.0) / 2.0
                best_idx = int(np.argmax(sims))
                best_sim = float(sims[best_idx])
                # Capture full breakdown for diagnostics — this is the same
                # data the matching loop sees, so we can surface it whether
                # we matched or fell through to "new cluster".
                breakdown_pre = ", ".join(
                    f"{self._stranger_labels[i]}={float(sims[i]):.3f}"
                    for i in range(len(sims))
                )
                best_sim_pre = best_sim
                best_label_pre = str(self._stranger_labels[best_idx])
                if best_sim >= self._match_threshold:
                    logger.info(
                        "Voiceprint hash: %s (matched existing cluster, "
                        "sim=%.3f, threshold=%.3f) | scores=[%s]",
                        best_label_pre, best_sim,
                        self._match_threshold, breakdown_pre,
                    )
                    if self._debug.enabled:  # SPEAKER-DEBUG
                        self._debug_stranger = {
                            "reappeared": True, "score": best_sim,
                            "closest_label": best_label_pre,
                            "num_clusters": int(len(self._stranger_embeds)),
                        }
                    return best_label_pre

            # No match — allocate a new cluster.
            self._stranger_counter = (self._stranger_counter + 1) % int(1e6)
            label = f"{_VOICE_STRANGER_PREFIX}{self._stranger_counter}"
            new_row = agg.reshape(1, -1).astype(np.float32)
            new_lbl = np.array([label])
            if self._stranger_embeds is None:
                self._stranger_embeds = new_row
                self._stranger_labels = new_lbl
            else:
                self._stranger_embeds = np.concatenate(
                    [self._stranger_embeds, new_row], axis=0,
                )
                self._stranger_labels = np.concatenate(
                    [self._stranger_labels, new_lbl], axis=0,
                )

            # Evict oldest entries once over the cap. Keeps disk bounded
            # without impacting recent speakers the agent still cares about.
            if len(self._stranger_embeds) > _MAX_VOICE_STRANGERS:
                drop = len(self._stranger_embeds) - _MAX_VOICE_STRANGERS
                self._stranger_embeds = self._stranger_embeds[drop:]
                self._stranger_labels = self._stranger_labels[drop:]

            self._save_strangers()
            if best_sim_pre is not None:
                # Hit the "no existing cluster matched" branch — surface the
                # closest miss so operators can spot threshold edge cases
                # (e.g. same speaker scoring 0.73 vs threshold 0.75).
                logger.info(
                    "Voiceprint hash: %s (new cluster, total=%d) | "
                    "closest=%s sim=%.3f below threshold=%.3f | scores=[%s]",
                    label, len(self._stranger_embeds),
                    best_label_pre, best_sim_pre,
                    self._match_threshold, breakdown_pre,
                )
            else:
                logger.info(
                    "Voiceprint hash: %s (new cluster, total=%d) | "
                    "no prior clusters",
                label, len(self._stranger_embeds),
            )
            if self._debug.enabled:  # SPEAKER-DEBUG
                self._debug_stranger = {
                    "reappeared": False, "score": best_sim_pre,
                    "closest_label": best_label_pre,
                    "num_clusters": int(len(self._stranger_embeds)),
                }
            return label

    def _match_stranger_clusters(
        self, query_embedding: np.ndarray, threshold: float,
    ) -> list[str]:
        """Return stranger labels whose centroid cosine-sim to query > threshold.

        Used by enroll() to auto-include clusters that belong to the same
        speaker but fragmented (e.g. distance / volume drift pushed centroids
        below the match threshold so they landed in different clusters even
        though it's one person). Pass a looser threshold than the match
        threshold so the fragmented siblings get pulled back together; the
        consistency filter downstream in enroll() handles any false positive.

        ``threshold`` is SCALED cosine in [0, 1] — same unit as MATCH /
        CONSISTENCY.
        """
        q = _l2(query_embedding)
        if float(np.linalg.norm(q)) == 0.0:
            logger.info("Auto-merge scan: query embedding has zero norm — skip")
            return []
        with self._stranger_lock:
            if (
                self._stranger_embeds is None
                or self._stranger_labels is None
                or len(self._stranger_embeds) == 0
            ):
                logger.info("Auto-merge scan: no stranger centroids to compare")
                return []
            # Both sides L2-normalized → dot product is raw cosine [-1, 1].
            # Convert to scaled [0, 1] so the threshold lives in one unit
            # across the file.
            sims = (self._stranger_embeds @ q + 1.0) / 2.0
            # Log every cluster's scaled similarity (not just matches) so
            # operators can tune threshold from real data: e.g. see voice_5
            # missed at sim=0.62 and decide to lower threshold to 0.60.
            breakdown = ", ".join(
                f"{self._stranger_labels[i]}={float(sims[i]):.3f}"
                for i in range(len(sims))
            )
            logger.info(
                "Auto-merge scan: threshold=%.2f query=[%s]",
                threshold, breakdown,
            )
            return [
                str(self._stranger_labels[i])
                for i, s in enumerate(sims)
                if float(s) > threshold
            ]

    def _save_strangers(self) -> None:
        """Persist stranger state to disk. Caller must hold _stranger_lock."""
        if self._stranger_embeds is None or self._stranger_labels is None:
            return
        try:
            np.save(_VOICE_STRANGERS_DIR / "embeds.npy", self._stranger_embeds)
            np.save(_VOICE_STRANGERS_DIR / "labels.npy", self._stranger_labels)
            np.save(
                _VOICE_STRANGERS_DIR / "counter.npy",
                np.array(self._stranger_counter),
            )
        except OSError as e:
            logger.warning("save voice strangers failed: %s", e)

    def _load_strangers(self) -> None:
        """Load stranger state from disk on startup. Silent on missing files."""
        embeds_path = _VOICE_STRANGERS_DIR / "embeds.npy"
        labels_path = _VOICE_STRANGERS_DIR / "labels.npy"
        counter_path = _VOICE_STRANGERS_DIR / "counter.npy"
        if not (embeds_path.exists() and labels_path.exists()):
            return
        try:
            self._stranger_embeds = np.load(embeds_path)
            self._stranger_labels = np.load(labels_path)
            if counter_path.exists():
                self._stranger_counter = int(np.load(counter_path))
            logger.info(
                "Loaded %d voice strangers (counter=%d)",
                len(self._stranger_embeds), self._stranger_counter,
            )
        except Exception as e:
            logger.warning("load voice strangers failed: %s", e)
            self._stranger_embeds = None
            self._stranger_labels = None
            self._stranger_counter = 0
