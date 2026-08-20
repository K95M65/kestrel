"""Unknown-face snapshot path helper — no ONNX / camera / numpy."""

from __future__ import annotations

import tempfile
import unittest
from pathlib import Path

from hal.drivers.sensing.perceptions.processors.faceid.constants import (
    STRANGER_ID_RE,
    latest_stranger_snapshot,
)


class StrangerSnapshotTests(unittest.TestCase):
    def test_rejects_junk_ids(self) -> None:
        self.assertIsNone(STRANGER_ID_RE.match("../etc"))
        self.assertIsNone(STRANGER_ID_RE.match("unknown"))
        self.assertIsNotNone(STRANGER_ID_RE.match("stranger_1"))

    def test_picks_newest_jpeg(self) -> None:
        with tempfile.TemporaryDirectory() as raw:
            d = Path(raw)
            (d / "stranger_1_100.jpg").write_bytes(b"a")
            (d / "stranger_1_200.jpg").write_bytes(b"b")
            (d / "stranger_2_50.jpg").write_bytes(b"c")
            got = latest_stranger_snapshot(d, "stranger_1")
            self.assertIsNotNone(got)
            self.assertEqual(got.name, "stranger_1_200.jpg")
            self.assertIsNone(latest_stranger_snapshot(d, "stranger_9"))
            self.assertIsNone(latest_stranger_snapshot(d, "../stranger_1"))


if __name__ == "__main__":
    unittest.main()

