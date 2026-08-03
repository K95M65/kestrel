"""Unit tests for the strict Deepgram interim wake-word matcher."""

import threading

from hal.drivers.voice._internal.speaker_decorate import SpeakerDecorator, merge_wake_words


def _decorator(words: list[str]) -> SpeakerDecorator:
    decorator = object.__new__(SpeakerDecorator)
    decorator._wake_words = words
    decorator._wake_words_lock = threading.Lock()
    return decorator


def test_wake_word_match_is_case_and_punctuation_insensitive():
    decorator = _decorator(["hey luna", "luna ơi", "này luna"])

    assert decorator.starts_with_wake_word("Hey Luna, thời tiết hôm nay?")
    assert decorator.starts_with_wake_word("LUNA ƠI! kể chuyện đi")
    assert decorator.starts_with_wake_word("này Luna xem giúp mình")


def test_wake_word_match_requires_a_prefix_and_word_boundary():
    decorator = _decorator(["hey luna", "luna"])

    assert not decorator.starts_with_wake_word("Mình vừa gặp Luna ngoài đường")
    assert not decorator.starts_with_wake_word("lunar calendar")
    assert decorator.starts_with_wake_word("luna, nghe mình nói này")


def test_device_type_alias_is_retained_alongside_agent_name():
    words = merge_wake_words(["hello lamp", "hey lamp"], ["hey luna", "luna"])

    assert words == ["hello lamp", "hey lamp", "hey luna", "luna"]
