"""Qwen Omni Realtime-specific enumerations."""

from enum import StrEnum


class QwenVoice(StrEnum):
    # Voice set of qwen-omni-turbo-realtime (Model Studio realtime docs).
    CHERRY = "Cherry"
    SERENA = "Serena"
    ETHAN = "Ethan"
    CHELSIE = "Chelsie"
