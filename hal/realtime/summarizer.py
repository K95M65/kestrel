"""LLM-based memory summarizer using the Anthropic Messages API."""

import logging
import threading

import hal.config as app_config
from hal.realtime.constants import RESOURCES_DIR

logger = logging.getLogger(__name__)

SUMMARIZE_PROMPT_PATH = RESOURCES_DIR / "summarize_prompt.md"


class RealtimeSummarizer:
    """Summarize text entries using the Anthropic Messages API."""

    MAX_INPUT_CHARS: int = 100_000

    def __init__(
        self,
        api_key: str = app_config.REALTIME_SUMMARIZER_API_KEY,
        base_url: str | None = app_config.REALTIME_SUMMARIZER_BASE_URL or None,
        model: str = app_config.REALTIME_SUMMARIZER_MODEL,
    ) -> None:
        # anthropic imports lazily on first summarize(): the SDK costs ~1.3s of
        # import time on device and every summarize() runs on a background
        # thread, so cold boot shouldn't pay for it.
        self._api_key = api_key
        self._base_url = base_url
        self._client = None
        self._client_lock = threading.Lock()
        self._model: str = model
        try:
            self._system_prompt: str = SUMMARIZE_PROMPT_PATH.read_text(encoding="utf-8").strip()
        except FileNotFoundError:
            logger.warning("[realtime] Summarize prompt not found at %s", SUMMARIZE_PROMPT_PATH)
            self._system_prompt = "Summarize the following entries concisely."

    def _get_client(self):
        if self._client is None:
            with self._client_lock:
                if self._client is None:
                    import anthropic
                    self._client = anthropic.Anthropic(
                        api_key=self._api_key,
                        base_url=self._base_url,
                        timeout=120.0,
                    )
        return self._client

    def summarize(self, entries: list[str]) -> str:
        """Summarize a list of text entries into a concise summary.

        Returns the summary text, or an empty string if entries are empty
        or the API call fails.
        """
        entries = [e.strip() for e in entries if e.strip()]
        if not entries:
            return ""

        user_content: str = "\n\n---\n\n".join(entries)
        if len(user_content) > self.MAX_INPUT_CHARS:
            logger.info("[realtime] Truncating summarizer input: %d → %d chars", len(user_content), self.MAX_INPUT_CHARS)
            user_content = user_content[-self.MAX_INPUT_CHARS :]

        try:
            response = self._get_client().messages.create(
                model=self._model,
                max_tokens=4096,
                system=self._system_prompt,
                messages=[
                    {"role": "user", "content": user_content},
                ],
            )
            summary: str = response.content[0].text.strip()
            logger.info(
                "[realtime] Summarized %d entries (%d chars) → %d chars",
                len(entries), len(user_content), len(summary),
            )
            return summary
        except Exception as e:
            logger.warning("[realtime] Summarization failed: %s", e)
            return ""
