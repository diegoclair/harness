"""The one speech model the engine transcribes with; film setup fetches it before any film needs it."""
from faster_whisper import WhisperModel

NAME = "medium"
LANGUAGE = "pt"


def load():
    return WhisperModel(NAME, device="cpu", compute_type="int8")


if __name__ == "__main__":
    load()
