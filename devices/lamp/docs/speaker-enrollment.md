# Speaker Voice Enrollment — Technical Spec

**Status: IMPLEMENTED** (2026-04)

## Overview

Lamp identifies who is speaking via **WeSpeaker ResNet34** (256-dim embedding, ONNX Runtime). When a speaker is not recognized, HAL saves the audio and optionally nudges the AI agent to enroll the voice. Enrollment is **self-service only** — each person enrolls their own voice.

## Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│  HAL (Python, port 5001)                                         │
│                                                                     │
│  VoiceService._stream_session()                                     │
│    ├─ STT transcript ready                                          │
│    ├─ _identify_and_decorate(transcript)                            │
│    │   ├─ audio_buffer → WAV → on-device preprocess (VAD gate)     │
│    │   │   └─ Mono→Resample→[HPF]→[NR]→VAD→[STOI]→RMS; reject clip  │
│    │   ├─ POST /audio-recognizer/embed  (preprocess=false)         │
│    │   │   └─ WeSpeaker ONNX → 256-dim L2-normalized (embed only)  │
│    │   ├─ Per-chunk voting vs enrolled embeddings                   │
│    │   ├─ Match ≥ 0.7 → "Speaker - Name: transcript"               │
│    │   └─ No match → _format_unknown_speaker()                     │
│    │       ├─ _should_request_enroll() gate                         │
│    │       │   ├─ ≥ 25 words in transcript                          │
│    │       │   └─ ≥ 5s audio duration                               │
│    │       ├─ PASS → "Unknown Speaker: ... (audio save at <path>,   │
│    │       │          auto enroll ...)"                              │
│    │       └─ FAIL → "Unknown Speaker: ..." (no enroll instruction) │
│    └─ POST /api/sensing/event → Lamp (Go)                          │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────────────┐
│  Lamp (Go, port 5000)                                               │
│                                                                     │
│  Two paths (both call domain.AppendEnrollNudge):                    │
│                                                                     │
│  1. Direct path (handler.go)                                        │
│     └─ Agent idle → send immediately to OpenClaw                    │
│                                                                     │
│  2. Drain path (service.go)                                         │
│     └─ Agent busy → queue → replay when idle                        │
│                                                                     │
│  AppendEnrollNudge(msg) — domain/voice.go:                          │
│    ├─ Check: contains "Unknown Speaker:" + "audio save at"          │
│    ├─ Cooldown: skip if < 5 min since last nudge                    │
│    └─ Append: "[REQUIRED: Follow speaker-recognizer/SKILL.md ...]"  │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────────────┐
│  OpenClaw Agent                                                     │
│                                                                     │
│  speaker-recognizer/SKILL.md                                        │
│    ├─ Detects self-introduction ("I'm X", "my name is X")           │
│    ├─ curl POST /speaker/enroll with wav_path + name                │
│    ├─ Two-turn: ask "Who are you?" → enroll with both paths         │
│    └─ Confirm: "Nice to meet you, Name!"                            │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

## Anti-Spam Gates

Four layers prevent the agent from repeatedly asking "who are you?":

| Layer | Where | Gate | Purpose |
|-------|-------|------|---------|
| **Audio duration** | HAL `voice_service.py` | `duration_s < SPEAKER_MIN_AUDIO_S` (0.8s) | Skip recognition entirely for very short audio |
| **Enroll instruction** | HAL `_should_request_enroll()` | `≥ 15 words AND ≥ 2s audio` | Don't append full enroll instruction for short utterances (short variant with multi-turn combine hint is still sent) |
| **Lamp-side nudge cooldown** | Lamp `domain/voice.go` | `5 min since last nudge` | Don't inject SKILL.md instruction more than once per 5 min |
| **Per-voiceprint nudge cooldown** | HAL `voice_service.py` | `30 min per voiceprint_hash` (`HAL_ENROLL_NUDGE_COOLDOWN_S`) | Don't repeat "ask user's name" for the same unknown voice cluster; plain `Unknown Speaker:` message sent instead |

## Model & Embedding

| Property | Value |
|----------|-------|
| Model | WeSpeaker ResNet34 (VoxCeleb trained) |
| Embedding dim | 256 |
| Runtime | ONNX Runtime (CPU) on perception-service (RunPod) |
| Endpoint | `POST {DL_BACKEND_URL}/lelamp/api/dl/audio-recognizer/embed` |
| Auth | `X-API-Key` header |
| Timeout | 15s |

### Recognition Algorithm

1. Audio → **on-device** preprocess on HAL (`Mono → Resample → [HighPass] → [NoiseReduce] → VAD → [STOI] → RMS`). Clips that fail the VAD/STOI/quality gate are rejected locally (treated as "unknown") and **never uploaded**.
2. Cleaned WAV → `POST /audio-recognizer/embed` with `preprocess=false`; the server skips its own preprocessing and only extracts per-chunk embeddings `[M, 256]` (it still windows/chunks the waveform itself)
3. Cosine similarity against all enrolled speaker embeddings
4. Per-chunk voting: each chunk votes for its closest match
5. Winner = most votes (tiebreak by average confidence)
6. `confidence ≥ 0.7` → match; else unknown

### Audio preprocessing (on-device)

The filter/VAD/normalize pipeline that used to run inside perception-service now runs on HAL, next to the mic — the same processors, in the same order, with the same defaults, ported to `hal/drivers/voice/speaker_recognizer/audio_processors/` (mirrors `AudioProcessorFactory` in perception-service). This keeps rejected audio off the network and puts the gate decision on the device.

- **Default chain**: `MonoConverter → Resampler(16k) → VoiceActivityFilter(silero) → SpeechIntelligibilityFilter(0.75) → RMSNormalizer(0.1)`. `HighPassFilter` and `NoiseReducer` exist but are **off by default** (same as perception).
- **VAD gate** (silero-vad): trims leading/trailing non-voice and rejects a clip when it removes all speech, the remaining audio is `< 0.5s`, or the voice ratio is `< 0.4`. A rejected clip raises `PreprocessRejected` → HAL returns "unknown" for recognize and skips the sample for enroll — exactly the behavior it had when perception returned HTTP 400.
- **STOI gate** (`SpeechIntelligibilityFilter`, reference-free SQUIM-OBJECTIVE STOI): runs **after VAD, before RMS**. Scores the trimmed clip in 5 s chunks and **mean**-aggregates, then rejects when the mean STOI is `< 0.75` (a NaN chunk from silence also rejects), raising `PreprocessRejected(reason="low_intelligibility")` → the same audio-level reject path as VAD (recognize → "unknown", enroll → skip the sample, keeping existing on-disk samples). The ONNX estimator (~20 MB, downloaded on first use from the CDN into `/root/local/models/squimm_stoi.onnx` — see `audio_processors/model_store.py`, same convention as the pose/faceid weights — onnxruntime CPU with the mem-arena off) loads once as a lazy singleton alongside silero VAD and only runs after VAD passes — at most once per utterance. If the weight can't be resolved (unreachable CDN / unknown filename) the gate is skipped with a warning (no crash).
- **Server flag**: HAL sends `preprocess=false`; perception's `/embed` is embed-only and now defaults to `preprocess=false` too (HAL is the only caller). A caller that uploads raw audio can pass `preprocess=true` to have the server clean it.
- **Consistency**: enroll and recognize share this one pipeline, so enrollments made after the move stay self-consistent. Voices enrolled under the **old server-side** pipeline should be re-enrolled if match quality drops.

### Embedding-model version tracking & migration

A stored embedding is only comparable to a query embedding produced by the **same** server model. If the perception-service embedding model is swapped, every previously-stored vector silently becomes meaningless to compare against — cosine similarity still returns a number, so the failure is a **wrong match**, not an error. HAL guards against this by stamping each profile with the model identity and re-embedding when it changes. Because every enrollment WAV is retained on disk, this is an automatic background job — no user has to re-record.

- **Model identity**: perception's `/audio-recognizer/embed` response (and `/health`) return `embed_model_version` — `<model-name>:<sha256(weights)[:12]>`, computed once when the model loads. `<model-name>` is the `AUDIO_EMBEDDER__MODEL` config value (`resnet293` / `resnet34` / `campplus` / `ecapa-tdnn1024`), e.g. `resnet293:1a2b3c4d5e6f`. Hashing the weights file catches even a **same-dimension checkpoint swap** that the `embedding_dim` check would miss. Only the model is fingerprinted; the on-device preprocessing config is deliberately **not** part of it.
- **On enroll**: HAL always takes the freshest version seen from that enroll's `/embed` calls and writes it to the voice `metadata.json` as `embed_model_version` (mirrored into the registry).
- **On recognize**: after embedding the query (which refreshes the known server version), HAL compares each enrolled profile's stored version against it. Profiles whose version **differs** are excluded from matching (so they read as **"unknown"**), a `Recognize: server embedding model is … stale …` line is logged, and a one-shot background re-embed migration is started. Fresh profiles match normally in the same call.
- **On HAL restart**: a background thread polls `/health` for the current `audio_embedder_version` (a few retries to cover server boot), cheaply scans profile metadata for staleness **before** loading the heavy preprocessing model, and migrates any stale profiles — so recognition is correct from the first turn instead of waiting for a recognize to notice.
- **Migration (re-embed)**: for each stale profile, HAL re-embeds its retained `sample_*.wav` files under the new model (same `_prepare_wav_for_embedding` → `/embed` → `_weighted_aggregate` path as enroll), then **atomically** swaps `embedding.npy` (temp file + rename) and updates `embed_model_version` / `embedding_dim` / `updated_at`. Guarded so only one migration runs at a time; a mid-migration server outage (`EmbeddingAPIUnavailableError`) **halts** cleanly with nothing corrupted and is retried on the next restart/recognize.
- **Un-migratable profiles**: a profile whose WAVs are all gone (or all rejected by today's gate) can't be re-embedded — it is flagged `needs_reenroll: true` in its metadata and the registry (surfaced on `/speaker/list-owners` and enroll/identity responses) and must be **physically re-enrolled**. This is the only case that needs a human.

### Enrollment Quality

1. Each WAV sample → on-device preprocess (as above) → embedding via perception-service (`preprocess=false`)
2. Filter by consistency threshold `0.7` (cosine similarity between samples)
3. Aggregate remaining embeddings via weighted average
4. Store L2-normalized vector at `/root/local/users/{name}/voice/embedding.npy`

### Voice Cluster Tracking (`voiceprint_hash`)

Every unknown voice is locally clustered so the server can say "this is the same unknown speaker we heard 3 minutes ago" without needing any backend support. Lets the agent combine multiple short utterances into one enroll call.

1. After embedding the query audio, the recognizer aggregates per-chunk embeddings into a single L2-normalized vector.
2. Compare against stored stranger-cluster centroids (cosine similarity).
3. Match ≥ `SPEAKER_MATCH_THRESHOLD` (default `0.75` — the **same** threshold as known-speaker matching; there is no separate stranger threshold) → reuse existing label `voice_N`.
4. No match → allocate new label `voice_{counter}`, append centroid to on-disk state.
5. Cap at `HAL_MAX_VOICE_STRANGERS` (default `50`) — oldest evicted when exceeded; eviction drops **both** the centroid row and that cluster's on-disk `voice_N/` WAV dir (an evicted cluster can never be matched again, so keeping its folder is dead disk).
6. The assigned hash is:
   - returned on the recognize response as `voiceprint_hash: "voice_N"` (null for known speakers)
   - surfaced in the nudge message as `[voice:voice_N]` tag so the skill can correlate turns
   - used to subdir-group the saved WAV (see Storage)

**Model change wipes the cluster store.** Stranger centroids are only comparable to a query from the **same** embedding model — so unlike enrolled profiles (which are *re-embedded* from retained WAVs), the whole stranger store is **wiped** when the model changes. The store is stamped with the model version it was built under (`voice_strangers/version.txt`); before any compare, HAL keeps the store **only when it can prove same-model provenance** — the live server version is known, the store's stamp **equals** it, **and** the stored dim equals the query dim. Anything else — a **missing** stamp, a **different** stamp, or a **different** dim — cannot prove the centroids came from the current model, so HAL drops the in-memory centroids, deletes the `embeds.npy`/`labels.npy` and every on-disk `voice_N/` WAV dir, and re-stamps. (An unstamped store is **never assumed** current: a same-dim checkpoint swap under a different model would otherwise slip through.) When the server reports **no** version at all, HAL falls back to a dim-only guard. `_stranger_counter` is kept **monotonic** so a freshly minted `voice_N` never collides with a leftover dir. Wiping (not re-embedding) is deliberate: strangers are anonymous and short-lived, so re-embedding throwaway clusters isn't worth the network cost.

**Trailing-silence trim**: before the WAV goes to the embedding API, the speaker-ID buffer is truncated at the last speech frame + 200 ms tail. Without this a 3-second utterance ends up as ~5.5 s with ~45% silence, diluting the embedding. Only affects the speaker-ID path — STT still receives the full stream.

## Configuration

| Parameter | Default | Env var | Description |
|-----------|---------|---------|-------------|
| Match threshold | 0.75 | `SPEAKER_MATCH_THRESHOLD` | Min confidence for speaker match |
| Enroll consistency | 0.75 | `SPEAKER_ENROLL_CONSISTENCY_THRESHOLD` | Min cosine similarity between enrollment samples |
| API timeout | 15s | `SPEAKER_EMBEDDING_API_TIMEOUT_S` | HTTP timeout for embedding API |
| Min audio for recognition | 0.8s | `HAL_SPEAKER_MIN_AUDIO_S` | Skip recognition below this |
| Min words for enroll nudge | 15 | Hardcoded in `_should_request_enroll()` | Transcript word count gate |
| Min duration for enroll nudge | 2.0s | Hardcoded in `_should_request_enroll()` | Audio duration gate |
| Lamp nudge cooldown | 5 min | Hardcoded in `domain/voice.go` | Don't re-inject SKILL instruction globally |
| Per-voiceprint nudge cooldown | 30 min | `HAL_ENROLL_NUDGE_COOLDOWN_S` | Don't re-ask name for same voiceprint cluster |
| Voice stranger match threshold | _(shared)_ | `SPEAKER_MATCH_THRESHOLD` | Reuses the known-speaker match threshold to cluster an unknown voice into an existing `voice_N` — no separate knob |
| Max voice strangers | 50 | `HAL_MAX_VOICE_STRANGERS` | Cluster cap; oldest evicted when exceeded |
| Voice strangers dir | `/root/local/voice_strangers` | `HAL_VOICE_STRANGERS_DIR` | Persist cluster embeddings (survives reboot) |
| Speaker recognition enabled | true | `HAL_SPEAKER_RECOGNITION_ENABLED` | Master toggle (default on; gated on the `audio` capability) |

### On-device preprocessing knobs

Mirror perception's `AudioProcessorSetting` defaults; override via env (all prefixed `HAL_SPEAKER_PROC_`).

| Parameter | Default | Env var | Description |
|-----------|---------|---------|-------------|
| Target sample rate | 16000 | `HAL_SPEAKER_PROC_TARGET_SR` | Resampler target |
| Mono | on | `HAL_SPEAKER_PROC_ENABLE_MONO` | Downmix to mono |
| Resample | on | `HAL_SPEAKER_PROC_ENABLE_RESAMPLE` | Resample to target SR |
| High-pass | off | `HAL_SPEAKER_PROC_ENABLE_HIGH_PASS` / `..._HIGH_PASS_CUTOFF_HZ` (80.0) | Butterworth HPF |
| Noise reduce | off | `HAL_SPEAKER_PROC_ENABLE_NOISE_REDUCE` / `..._NOISE_STATIONARY` | `noisereduce` (lazy import) |
| VAD | on | `HAL_SPEAKER_PROC_ENABLE_VAD` | silero-vad gate |
| VAD min duration | 0.5s | `HAL_SPEAKER_PROC_VAD_MIN_DURATION_SEC` | Reject if stripped audio shorter |
| VAD min voice ratio | 0.4 | `HAL_SPEAKER_PROC_VAD_MIN_VOICE_RATIO` | Reject if voice fraction lower |
| VAD speech-prob threshold | 0.6 | `HAL_SPEAKER_PROC_VAD_SPEECH_PROB_THRESHOLD` | Silero onset threshold (offset = −0.15); higher trims trailing/leading silence more (silero default 0.5) |
| STOI gate | on | `HAL_SPEAKER_PROC_ENABLE_STOI` | SQUIM-OBJECTIVE intelligibility gate (after VAD, before RMS) |
| STOI model path | `/root/local/models/squimm_stoi.onnx` | `HAL_SPEAKER_PROC_STOI_MODEL_PATH` | ONNX estimator (~20 MB), downloaded from CDN on first use; gate skipped if unresolvable |
| STOI threshold | 0.75 | `HAL_SPEAKER_PROC_STOI_THRESHOLD` | Reject if mean STOI below this |
| STOI chunk | 5.0s | `HAL_SPEAKER_PROC_STOI_CHUNK_SEC` | Chunk length scored, then mean-aggregated |
| RMS normalize | on | `HAL_SPEAKER_PROC_ENABLE_RMS_NORMALIZE` / `..._RMS_TARGET` (0.1) | Fixed-loudness normalize |

### Debug tracing (temporary)

`speaker_recognizer.py` carries a self-contained diagnostic tracer, tagged `SPEAKER-DEBUG` throughout the file, for tuning recognition thresholds on real audio. **It is OFF by default (production-safe) — set `HAL_SPEAKER_DEBUG=true` to enable it during development**, and it is meant to be deleted entirely before a final deploy. `grep -n "SPEAKER-DEBUG"` finds every line belonging to it; no other module or config file is involved.

Each `recognize()` / `enroll()` call writes one directory:

```
<root>/recognize/<ts>_<class>_<confidence>/     class = enrolled name | stranger-<N> | unknown
<root>/recognize/<ts>_FAIL-<reason>/            no-voice | low-voice | low-stoi | too-short | server-error | …
<root>/enroll/<ts>_<norm>_<cohesion>/           cohesion = mean sim of kept samples to the centroid
<root>/enroll/<ts>_FAIL-<reason>/
```

holding `input.wav` (raw) plus `preprocessed.wav` (post VAD/STOI/RMS — the audio actually uploaded) / `sample_new_NN.wav`, the embeddings as `.npy`, and `result.json`. A recognize records a `preprocessing` block (cleaned duration/RMS, the STOI score the clip passed with, and the threshold it cleared) so you can tell a "bad audio" miss from a "wrong speaker" miss; a clip killed by the gate instead files a `FAIL-<reason>` dir whose `preprocessing_reject` holds the structured reason and its measurements. For a recognize the JSON carries the **full** decision breakdown — not just the top-3 `candidates` the API returns, but `speaker_summary` (votes + mean/max similarity for *every* enrolled speaker, including 0-vote losers) and `per_chunk_scores` (each chunk vs every speaker, plus which speaker that chunk voted for). The same matrix is saved as `chunk_scores.npy` (`[chunks × speakers]`, columns in `enrolled_speakers` order). Unknown speakers also record the stranger-cluster match score and which cluster was closest.

| Parameter | Default | Env var | Description |
|-----------|---------|---------|-------------|
| Debug tracing | **off** | `HAL_SPEAKER_DEBUG` | Set `true` to enable. Read once at construction — restart HAL after changing |
| Output root | `speaker_logs/` next to `speaker_recognizer.py` | `HAL_SPEAKER_DEBUG_DIR` | Falls back to a temp dir if the source tree is read-only (device deploy) |
| Max entries | 1000 | `HAL_SPEAKER_DEBUG_MAX_ENTRIES` | Per-kind directory cap, oldest pruned; `0` = unbounded |

The default output dir is git-ignored — never commit trace output. The tracer swallows all of its own errors, so a failing trace can never break recognition.

## Storage

```
/root/local/users/{name}/
  metadata.json                      # Shared identity (telegram, display_name)
  voice/
    embedding.npy                    # L2-normalized aggregated vector [256]
    metadata.json                    # num_samples, dim, timestamps,
                                     #   embed_model_version, needs_reenroll
    sample_{origin}_{ts}_{uuid}.wav  # Individual enrollment samples (16kHz mono)

/tmp/hal-unknown-voice/
  incoming_{ts}_{uuid}.wav           # Known-speaker audio (flat)
  voice_{N}/
    incoming_{ts}_{uuid}.wav         # Unknown audio — grouped by voiceprint cluster

/root/local/voice_strangers/
  embeds.npy                         # Stranger cluster centroids [N, 256] (deleted while store is wiped)
  labels.npy                         # Cluster labels ["voice_1", "voice_2", ...] (deleted while store is wiped)
  counter.npy                        # Monotonic counter for next new label (survives a wipe)
  version.txt                        # Embed-model version the centroids were built under; mismatch → wipe
```

## API Endpoints (HAL, port 5001)

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/speaker/enroll` | Enroll voice from wav_paths + name |
| `POST` | `/speaker/record-enroll` | Record from the device mic (`arecord`, `duration_sec` 1–60, default 15) then enroll the capture |
| `POST` | `/speaker/recognize` | Recognize speaker from wav_path |
| `POST` | `/speaker/identity` | Link Telegram identity to existing profile |
| `POST` | `/speaker/remove` | Remove voice profile by name |
| `POST` | `/speaker/reset` | Remove all voice profiles |
| `GET`  | `/speaker/list` | List enrolled speakers |

### Error contract

`/speaker/enroll` distinguishes two failure classes:

| HTTP | When | Skill behavior |
|------|------|----------------|
| `400` | Audio-level reject (too short, silent, VAD found no speech, mean STOI below threshold → `low_intelligibility`, perception-service returned 4xx) | Ask user to re-record / speak more clearly |
| `503` | Embedding service unreachable (network, 5xx, malformed response) | Tell user to try again shortly — nothing on disk was modified |

`/speaker/recognize` never fails with 5xx for embedding outages — it returns `200` with `{name: "unknown", error: "<reason>"}` so the skill can gracefully degrade. Only input-level problems (missing WAV, bad base64) return `400`.

### Mic ownership during record-enroll

ALSA capture is exclusive — only one process may hold the mic. `/speaker/record-enroll` therefore stops the voice pipeline, records with `arecord`, and restarts the pipeline from its own `finally` block.

Any other path that starts the pipeline while that recording is in flight steals the capture device and **both** sides fail with `audio open error: Device or resource busy`: the enroll returns `500` and the voice loop dies on the same error. So every caller goes through `state.start_voice_service(reason)`, which refuses (and logs the reason) while `state._enrolling` is set. The single exception is record-enroll's own restore — it owns the stop and runs after the flag is cleared.

## Key Code Locations

| Component | File | Function/Struct |
|-----------|------|-----------------|
| STT → speaker ID | `hal/drivers/voice/voice_service.py` | `_identify_and_decorate()` |
| Enroll gate | `hal/drivers/voice/voice_service.py` | `_should_request_enroll()` |
| Message formatting | `hal/drivers/voice/voice_service.py` | `_format_unknown_speaker()` |
| Speaker recognizer | `hal/drivers/voice/speaker_recognizer/speaker_recognizer.py` | `SpeakerRecognizer` |
| Mic-ownership gate | `hal/app_state.py` | `start_voice_service()` |
| Record + enroll route | `hal/routes/speaker.py` | `speaker_record_enroll()` |
| Nudge injection + cooldown | `system/domain/voice.go` | `AppendEnrollNudge()` |
| Direct event path | `system/server/sensing/delivery/http/handler.go` | `PostEvent()` |
| Drain/replay path | `runtimes/openclaw/service.go` | `drainPendingEvents()` |
| Agent skill | `lamp/resources/openclaw-skills/speaker-recognizer/SKILL.md` | — |
| Embedding model | `integrations/perception-service/src/core/audio_recognition/audio_recognizer.py` | `ResNet34Recognizer` (default), `EcapaTdnn1024Recognizer`, `CamPPlusRecognizer` — chọn qua env `AUDIO_RECOGNIZER_ENGINE` |
| Embedding endpoint | `integrations/perception-service/src/protocols/htpp/audio_recognizer.py` | `embed_audio()` |
| Config | `hal/config.py` | `SPEAKER_*` constants |

## Message Flow Examples

### Short utterance (blocked)
```
User says: "hey" (2 words, 0.9s audio)
→ HAL: skip recognition (< SPEAKER_MIN_AUDIO_S)
→ Message: "hey" (no prefix, no enroll instruction)
```

### Medium utterance (recognized but no enroll nudge)
```
User says: "turn on the lights please" (5 words, 3s audio)
→ HAL: recognize → unknown, _should_request_enroll(5 words, 3s) = false
→ Message: "Unknown Speaker: turn on the lights please"
→ Lamp: no "audio save at" in message → AppendEnrollNudge returns unchanged
→ Agent: responds normally, doesn't ask who user is
```

### Multi-turn combine (same voice cluster)
```
User turn 1: "nice to meet you today. Okay." (5 words)
→ HAL: recognize → unknown, voiceprint_hash=voice_5
→ WAV moved to /tmp/hal-unknown-voice/voice_5/incoming_A.wav
→ Message: "Unknown Speaker: [voice:voice_5] nice to meet you today. Okay. (audio saved at ..._A.wav. Note: audio is too short for single enrollment. If prior turns tagged the same voice_5, combine their saved paths with this one...)"
→ Agent: asks "Could you tell me your name?"

User turn 2: "I'm Alex." (2 words)
→ HAL: voiceprint_hash=voice_5 (same cluster, sim=0.75)
→ WAV moved to /tmp/hal-unknown-voice/voice_5/incoming_B.wav
→ Message: "Unknown Speaker: [voice:voice_5] I'm Alex. (audio saved at ..._B.wav...)"
→ Agent: scans prior turns for same [voice:voice_5] tag → finds path A
→ Agent: POST /speaker/enroll with wav_paths=[path_A, path_B], name="Alex"
→ Agent: "Nice to meet you, Alex!"
```

### Long utterance (full enroll flow)
```
User says: "Hi my name is Leo and I just got home from work..." (30 words, 8s audio)
→ HAL: recognize → unknown, _should_request_enroll(30 words, 8s) = true
→ Message: "Unknown Speaker: Hi my name is Leo... (audio save at /tmp/hal-unknown-voice/incoming_xxx.wav, auto enroll...)"
→ Lamp: AppendEnrollNudge → cooldown OK → append "[REQUIRED: Follow speaker-recognizer/SKILL.md...]"
→ Agent: detects "my name is Leo" → POST /speaker/enroll → "Nice to meet you, Leo!"
```

### Cooldown (blocked)
```
Same unknown speaker, 2 minutes later:
→ HAL: _should_request_enroll = true (long enough)
→ Message has "audio save at"
→ Lamp: AppendEnrollNudge → cooldown NOT elapsed (< 5 min) → skip instruction
→ Agent: sees "Unknown Speaker: ..." without SKILL instruction → responds normally
```
