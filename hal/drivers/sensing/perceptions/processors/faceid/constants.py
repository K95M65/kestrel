"""Shared on-disk paths and constants for the v2 face pipeline.

Kept in one place so ``recognizer`` (enrollment bank) and ``perception`` (state
machine) agree on where user photos and stranger state live.
"""

from pathlib import Path

import hal.config as config

# Sentinel score used when an embedding bank is empty (no match possible).
_NO_MATCH = -2.0

# Per-user data directory (face photos, wellbeing notes, mood history).
USERS_DIR = Path(config.USERS_DIR)
USERS_DIR.mkdir(parents=True, exist_ok=True)

# Persisted stranger bank / snapshots. EdgeFace embeddings are not comparable
# with the old buffalo_sc embeds.npy (different dimensionality / metric), so the
# v2 bank keeps its own directory.
STRANGER_STATE_DIR = Path(config.STRANGERS_DIR)
STRANGER_STATE_DIR.mkdir(exist_ok=True, parents=True)
_STRANGER_STATS_FILE = USERS_DIR / ".stranger_stats.json"
_STRANGER_SNAPSHOTS_DIR = STRANGER_STATE_DIR / "snapshots"
