"""ONNX runtime wrapper around the 300 KB ten-vad.onnx model.

The model is a small recurrent net: it takes a (batch, 3, 41) stack of the last
three feature frames plus four 64-dim LSTM states, and returns a speech
probability plus the updated states. Everything here is numpy + onnxruntime —
no libten_vad, no sherpa-onnx, no torch.
"""

from __future__ import annotations

import pathlib
from typing import Any

import numpy as np
import numpy.typing as npt
import onnxruntime as ort

from .features import CONTEXT_FRAMES, FEATURE_DIM, HOP_SIZE, FeatureExtractor

DEFAULT_MODEL_PATH = pathlib.Path(__file__).with_name("assets") / "ten-vad.onnx"

HIDDEN_DIM: int = 64  # AUP_AED_MODEL_HIDDEN_DIM
N_STATES: int = 4
# Upstream clears the recurrent state every 1875 frames (30 s) — aed.cc
# `dynamCfg.resetFrameNum`. Kept so probabilities track the C library exactly.
DEFAULT_STATE_RESET_FRAMES: int = 1875


class TenVadModel:
    """Streaming speech-probability estimator (one probability per 256 samples)."""

    def __init__(
        self,
        model_path: str | pathlib.Path = DEFAULT_MODEL_PATH,
        num_threads: int = 1,
        state_reset_frames: int = DEFAULT_STATE_RESET_FRAMES,
        pitch_hz: float | None = None,
    ) -> None:
        opts = ort.SessionOptions()
        opts.intra_op_num_threads = num_threads
        opts.inter_op_num_threads = num_threads
        opts.log_severity_level = 3
        self._session = ort.InferenceSession(
            str(model_path), sess_options=opts, providers=["CPUExecutionProvider"]
        )
        self._input_names = [i.name for i in self._session.get_inputs()]
        self._output_names = [o.name for o in self._session.get_outputs()]
        self._state_reset_frames = state_reset_frames

        self._features = FeatureExtractor(pitch_hz=pitch_hz)
        self._frames_since_reset = 0

        # Persistent I/O buffers. The model is tiny, so per-call marshalling
        # dominates its runtime; binding fixed buffers once and mutating them in
        # place is ~1.35x faster than passing a fresh feed dict per frame, with
        # bit-identical output.
        self._input = np.zeros((1, CONTEXT_FRAMES, FEATURE_DIM), dtype=np.float32)
        self._context = self._input[0]  # view — writes here feed the session
        self._states = [np.zeros((1, HIDDEN_DIM), dtype=np.float32) for _ in range(N_STATES)]
        self._next_states = [np.zeros((1, HIDDEN_DIM), dtype=np.float32) for _ in range(N_STATES)]
        self._prob = np.zeros((1, 1, 1), dtype=np.float32)
        self._binding = self._make_binding()

    def _make_binding(self) -> Any:
        """Bind inputs/outputs to the persistent buffers; None if unsupported."""
        try:
            binding = self._session.io_binding()
            binding.bind_cpu_input(self._input_names[0], self._input)
            for name, state in zip(self._input_names[1:], self._states):
                binding.bind_cpu_input(name, state)
            binding.bind_output(
                self._output_names[0], "cpu", 0, np.float32,
                self._prob.shape, self._prob.ctypes.data,
            )
            for name, state in zip(self._output_names[1:], self._next_states):
                binding.bind_output(
                    name, "cpu", 0, np.float32, state.shape, state.ctypes.data
                )
            return binding
        except Exception:  # pragma: no cover - depends on onnxruntime build
            # Any binding trouble on an unusual build: fall back to plain run().
            return None

    # --- lifecycle ---

    def reset(self) -> None:
        """Clear all streaming state (features, context stack, LSTM states)."""
        self._features.reset()
        self._context[:] = 0.0
        self.reset_states()
        self._frames_since_reset = 0

    def reset_states(self) -> None:
        for state in self._states:
            state[:] = 0.0

    # --- inference ---

    def _run_frame(self, feature: npt.NDArray[np.float32]) -> float:
        context = self._context
        context[:-1] = context[1:]
        context[-1] = feature

        if self._binding is not None:
            self._session.run_with_iobinding(self._binding)
            # Outputs landed in _next_states; roll them into the bound inputs.
            for state, next_state in zip(self._states, self._next_states):
                state[:] = next_state
            prob = float(self._prob[0, 0, 0])
        else:
            inputs = {self._input_names[0]: self._input}
            for name, state in zip(self._input_names[1:], self._states):
                inputs[name] = state
            outputs = self._session.run(self._output_names, inputs)
            for state, out in zip(self._states, outputs[1:]):
                state[:] = np.asarray(out, dtype=np.float32)
            prob = float(np.asarray(outputs[0]).reshape(-1)[0])

        self._frames_since_reset += 1
        if self._state_reset_frames and self._frames_since_reset >= self._state_reset_frames:
            self.reset_states()
            self._frames_since_reset = 0

        return prob

    def process(self, samples: npt.NDArray[np.float32]) -> npt.NDArray[np.float32]:
        """Return one speech probability per 256-sample hop.

        Args:
            samples: int16-scale float samples, length a multiple of 256.
                Trailing partial hops must be handled by the caller.
        """
        feats = self._features.process(samples)
        return np.array([self._run_frame(f) for f in feats], dtype=np.float32)

    def process_hop(self, hop: npt.NDArray[np.float32]) -> float:
        """Single-hop convenience wrapper mirroring ``TenVad.process``."""
        if hop.size != HOP_SIZE:
            raise ValueError(f"expected {HOP_SIZE} samples, got {hop.size}")
        return float(self.process(hop)[0])
