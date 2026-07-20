"""Boot-scoped sidecar persistence for TTL dedup maps.

Several perceptions dedup outbound sensing events with a
{(user, bucket): last_sent_ts} TTL map held in RAM. A HAL service restart
(deploy/OTA) wipes that map, so the first flush afterwards re-fires the
last-known event as if it were news — one wasted agent turn per emitter.
This sidecar persists the map to tmpfs, scoped to the current kernel
boot_id: a service restart restores it, a full device reboot starts fresh
on purpose (same pattern as the motion/presence/scene/LED sidecars).

Writes happen only on send/reset — a few per hour.
"""

import json
import logging
from pathlib import Path

logger = logging.getLogger(__name__)


def current_boot_id() -> str:
    try:
        return Path("/proc/sys/kernel/random/boot_id").read_text().strip()
    except Exception:
        return ""


class DedupStateSidecar:
    """Persist/restore a {(user, bucket): last_sent_ts} dedup map."""

    def __init__(self, path: str, log_tag: str):
        self._path: Path = Path(path)
        self._tag: str = log_tag

    def load(self) -> dict[tuple[str, str], float]:
        """Return the persisted map, or {} when missing/stale/unreadable."""
        try:
            if not self._path.exists():
                return {}
            data = json.loads(self._path.read_text())
            if data.get("boot_id") != current_boot_id():
                self._path.unlink(missing_ok=True)
                return {}
            entries = {
                (str(user), str(bucket)): float(ts)
                for user, bucket, ts in data.get("entries") or []
            }
            if entries:
                logger.info(
                    "[%s] dedup state restored — %d keys (no re-fire)",
                    self._tag,
                    len(entries),
                )
            return entries
        except Exception as e:
            logger.warning("[%s] dedup state load failed: %s", self._tag, e)
            return {}

    def save(self, entries: dict[tuple[str, str], float]) -> None:
        """Write the sidecar; an empty map unlinks it so a restart can't
        resurrect state a reset just cleared."""
        try:
            if not entries:
                self._path.unlink(missing_ok=True)
                return
            self._path.write_text(
                json.dumps(
                    {
                        "boot_id": current_boot_id(),
                        "entries": [
                            [user, bucket, ts]
                            for (user, bucket), ts in entries.items()
                        ],
                    }
                )
            )
        except Exception as e:
            logger.warning("[%s] dedup state save failed: %s", self._tag, e)
