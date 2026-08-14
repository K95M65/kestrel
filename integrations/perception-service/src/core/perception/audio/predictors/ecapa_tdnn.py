"""ECAPA-TDNN 1024 speaker embedding model."""

from pathlib import Path

from core.enums.audio import AudioEmbedderEnum
from core.enums.files import ModelEnum
from core.perception.audio.predictors.base import AudioEmbedder
from core.utils.files import get_default_cdn_url, get_default_model_path


class EcapaTdnn1024Embedder(AudioEmbedder):
    """WeSpeaker ECAPA-TDNN-1024-LM speaker embedder (192-dim).

    "1024" is the TDNN channel width, not the embedding size — the projected
    speaker embedding is 192-dim (config.yaml ``model_args.embed_dim``).
    """

    MODEL_NAME: AudioEmbedderEnum | None = AudioEmbedderEnum.ECAPA_TDNN_1024
    DEFAULT_MODEL_PATH: Path | None = get_default_model_path(ModelEnum.WESPEAKER_ECAPA_TDNN_1024)
    DEFAULT_REMOTE_URL: str | None = get_default_cdn_url(ModelEnum.WESPEAKER_ECAPA_TDNN_1024)
