"""HTTP contract for POST /voice/preview-audio (browser TTS preview)."""
import tempfile
import unittest
import wave
from pathlib import Path

from fastapi import FastAPI
from fastapi.testclient import TestClient

import hal.app_state as state
from hal.routes.voice import router


def _tiny_wav(path: Path) -> None:
    with wave.open(str(path), "wb") as wav:
        wav.setnchannels(1)
        wav.setsampwidth(2)
        wav.setframerate(16000)
        wav.writeframes(b"\x00\x00" * 160)


class _FakeBackend:
    available = True
    _api_key = "k"
    _base_url = "http://example"


class _FakeTTS:
    available = True
    speaking = False
    _provider = "openai"
    _voice = "Rachel"
    _backend = _FakeBackend()
    _sd = None

    def __init__(self, path: Path):
        self._path = path

    def render_preview_wav(self, text: str) -> Path:
        assert text
        _tiny_wav(self._path)
        return self._path


class TestVoicePreviewAudio(unittest.TestCase):
    def setUp(self):
        self._prev = state.tts_service
        self._tmp = tempfile.TemporaryDirectory()
        wav = Path(self._tmp.name) / "preview.wav"
        state.tts_service = _FakeTTS(wav)
        app = FastAPI()
        app.include_router(router)
        self.client = TestClient(app)

    def tearDown(self):
        state.tts_service = self._prev
        self._tmp.cleanup()

    def test_returns_wav_without_playing(self):
        r = self.client.post("/voice/preview-audio", json={"text": "Hello there", "voice": "Rachel"})
        self.assertEqual(r.status_code, 200)
        self.assertIn("audio/wav", r.headers.get("content-type", ""))
        self.assertGreater(len(r.content), 40)

    def test_rejects_empty_text(self):
        r = self.client.post("/voice/preview-audio", json={"text": ""})
        self.assertEqual(r.status_code, 422)

    def test_503_when_tts_missing(self):
        state.tts_service = None
        r = self.client.post("/voice/preview-audio", json={"text": "Hello"})
        self.assertEqual(r.status_code, 503)
