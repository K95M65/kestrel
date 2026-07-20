# HAL Voice VAD Presets

Copy one preset to the device HAL `.env`, then restart HAL.

Suggested order:

1. `01-balanced-cost-safe.env` - first choice; sensitive enough for normal speech while keeping WebRTC and Silero noise filters enabled.
2. `02-sensitive-filtered.env` - use if the first syllable still needs extra force.
3. `03-max-sensitive-debug.env` - short debug run only; disables WebRTC and Silero to prove whether the filters are rejecting quiet speech.

Keep any existing secrets and device-specific audio routes from the target `.env`.
These files only contain voice/VAD/realtime cost guards.
