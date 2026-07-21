"""Back-compat shim for the v2 face pipeline.

The implementation moved from this single ~2.6k-line module into the
``faceid`` package (split by concern: detector, landmark, embedder, pipeline,
recognizer, perception, model store). This module re-exports the public surface
so existing imports keep working unchanged, e.g.::

    from hal.drivers.sensing.perceptions.processors.facerecognizer_v2 import (
        FacePerception,
        USERS_DIR,
    )

New code should import from ``.faceid`` directly.
"""

from .faceid import FacePerception, FaceRecognizer, USERS_DIR

__all__ = ["FacePerception", "FaceRecognizer", "USERS_DIR"]
