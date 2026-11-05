"""What the render says, one timestamped line per phrase, for QA to compare against the script.

usage: transcribe.py <media> <out.md>
"""
import os, subprocess, sys, tempfile

import whisper_model

media, out = sys.argv[1], sys.argv[2]
with tempfile.TemporaryDirectory() as tmp:
    wav = os.path.join(tmp, "audio.wav")
    subprocess.run(["ffmpeg", "-y", "-loglevel", "error", "-i", media, "-ac", "1", "-ar", "16000", wav], check=True)
    segments, _ = whisper_model.load().transcribe(wav, language=whisper_model.LANGUAGE, vad_filter=True, beam_size=5)
    lines = [f"`{int(s.start // 60):02d}:{int(s.start % 60):02d}` {s.text.strip()}  " for s in segments]
os.makedirs(os.path.dirname(os.path.abspath(out)), exist_ok=True)
with open(out, "w", encoding="utf-8") as f:
    f.write("\n".join([f"# {os.path.basename(media)}", "", *lines, ""]))
