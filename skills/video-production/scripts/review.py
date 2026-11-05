"""Measures a rendered video against its spoken script, so a reviewer looks at numbers and frames, not at a hunch.

Usage: python review.py <video.mp4> <script.txt> <out-dir>
The script file holds the spoken text, one line of narration per line, in the order the video says them.
It needs ffmpeg on PATH and a Python with numpy and faster-whisper (pip install numpy faster-whisper).
"""
import difflib, json, math, os, re, subprocess, sys, unicodedata

try:
    import numpy as np
    from faster_whisper import WhisperModel
except ImportError:
    sys.exit("needs numpy and faster-whisper: pip install numpy faster-whisper (a venv outside the project is fine)")

video, script_path, out = sys.argv[1:4]
os.makedirs(out, exist_ok=True)
lines = [l.strip() for l in open(script_path, encoding="utf8") if l.strip()]
SR = 16000

# A pause inside a word longer than this is heard as the word breaking ("tx… t"); a stop consonant's own closure
# stays under it.
BROKEN_WORD_GAP = 0.12
# Narrations approved by ear spread under six semitones of median pitch across lines; a line generated apart from
# the others moved about nine in the incident this check was born from.
PITCH_SPREAD_SEMITONES = 7
# A side margin that differs from the other by more than this reads as off-centre on a phone.
CENTRE_TOLERANCE_PX = 30


def norm(text):
    text = unicodedata.normalize("NFKD", re.sub(r"<[^>]*>", " ", text)).encode("ascii", "ignore").decode().lower()
    return re.findall(r"[a-z0-9]+", text)


def run(*cmd):
    return subprocess.run(cmd, capture_output=True, text=True)


wav = os.path.join(out, "audio.wav")
run("ffmpeg", "-v", "error", "-y", "-i", video, "-ac", "1", "-ar", str(SR), wav)
pcm = np.frombuffer(subprocess.run(["ffmpeg", "-v", "error", "-i", wav, "-f", "s16le", "-"], capture_output=True).stdout, np.int16).astype(float) / 32768
duration = len(pcm) / SR

silence_log = run("ffmpeg", "-hide_banner", "-i", wav, "-af", "silencedetect=noise=-35dB:d=0.04", "-f", "null", "-").stderr
silences = list(zip(map(float, re.findall(r"silence_start: ([\d.]+)", silence_log)), map(float, re.findall(r"silence_end: ([\d.]+)", silence_log))))

model = WhisperModel("medium", device="cpu", compute_type="int8")
segments, _ = model.transcribe(wav, language=os.environ.get("REVIEW_LANG", "pt"), beam_size=5, word_timestamps=True, condition_on_previous_text=False)
words = [w for s in segments for w in s.words]
heard = [(t, i) for i, w in enumerate(words) for t in norm(w.word)]
script = [(t, k) for k, l in enumerate(lines) for t in norm(l)]

report = {"duration": round(duration, 2), "findings": []}
find = lambda kind, at, text: report["findings"].append({"kind": kind, "at": round(at, 2), "text": text})

sm = difflib.SequenceMatcher(None, [t for t, _ in script], [t for t, _ in heard], autojunk=False)
owner = [[] for _ in lines]
for op, a0, a1, b0, b1 in sm.get_opcodes():
    if op == "equal":
        for j in range(a1 - a0):
            owner[script[a0 + j][1]].append(heard[b0 + j][1])
        continue
    at = words[heard[b0][1]].start if b0 < len(heard) else duration
    said = " ".join(t for t, _ in heard[b0:b1]) or "(nothing)"
    wrote = " ".join(t for t, _ in script[a0:a1]) or "(nothing)"
    find("transcript", at, f"script '{wrote}', heard '{said}' (figures and names often mishear: confirm by ear)")

spans = [(words[min(o)].start, words[max(o)].end) if o else None for o in owner]
for k, span in enumerate(spans):
    if span is None:
        find("missing-line", 0, f"line {k + 1} not heard at all: {lines[k]}")

for w in words:
    for a, b in silences:
        if a > w.start + 0.03 and b < w.end - 0.03 and b - a >= BROKEN_WORD_GAP:
            find("broken-word", a, f"'{w.word.strip()}' stops for {b - a:.2f} s in the middle")

# A line that pauses longer inside itself than before the next line runs two lines together.
for k in range(len(lines) - 1):
    if not spans[k] or not spans[k + 1]:
        continue
    between = spans[k + 1][0] - spans[k][1]
    inner = max((b - a for a, b in silences if a > spans[k][0] and b < spans[k][1]), default=0)
    if inner >= between:
        find("pause-order", spans[k][1], f"line {k + 1} pauses {inner:.2f} s inside but only {between:.2f} s before line {k + 2}")


def median_pitch(a, b):
    x, found, n = pcm[int(a * SR):int(b * SR)], [], 640
    gate = 0.15 * math.sqrt((x ** 2).mean() + 1e-12)
    for i in range(0, len(x) - n, 320):
        w = x[i:i + n]
        if math.sqrt((w ** 2).mean()) < gate:
            continue
        w = w - w.mean()
        c = np.correlate(w, w, "full")[n - 1:]
        lo, hi = SR // 400, SR // 70
        k = lo + int(np.argmax(c[lo:hi]))
        if c[k] > 0.4 * c[0]:
            found.append(SR / k)
    return float(np.median(found)) if found else None


pitches = [median_pitch(*s) if s else None for s in spans]
known = [p for p in pitches if p]
report["pitch_by_line_hz"] = [round(p) if p else None for p in pitches]
if len(known) > 1:
    spread = 12 * math.log2(max(known) / min(known))
    report["pitch_spread_semitones"] = round(spread, 1)
    if spread > PITCH_SPREAD_SEMITONES:
        find("voice-drift", 0, f"median pitch spans {spread:.1f} semitones across lines: the voice sounds like another person")

first = next((s[0] for s in spans if s), None)
report["first_word_at"] = round(first, 2) if first is not None else None


def frame(t, path, extra=""):
    run("ffmpeg", "-v", "error", "-y", "-ss", f"{t}", "-i", video, "-frames:v", "1", *(["-vf", extra] if extra else []), path)


frame(0, os.path.join(out, "frame0.png"))

# The last frame with content stands alone on screen, so its content box should sit in the middle.
end_t = max(0, duration - 0.6)
raw = subprocess.run(["ffmpeg", "-v", "error", "-ss", f"{end_t}", "-i", video, "-frames:v", "1", "-f", "rawvideo", "-pix_fmt", "gray", "-"], capture_output=True).stdout
probe = run("ffprobe", "-v", "error", "-select_streams", "v:0", "-show_entries", "stream=width,height", "-of", "csv=p=0", video).stdout.strip()
width, height = map(int, probe.split(",")[:2])
img = np.frombuffer(raw, np.uint8)
if img.size == width * height:
    img = img.reshape(height, width).astype(int)
    mask = np.abs(img - np.median(img[:, :20])) > 18
    cols = np.where(mask.any(0))[0]
    if cols.size:
        left, right = int(cols.min()), int(width - 1 - cols.max())
        report["end_frame_margins_px"] = [left, right]
        if abs(left - right) > CENTRE_TOLERANCE_PX:
            find("off-centre", end_t, f"end frame content has {left} px left and {right} px right")

times = sorted({round(x, 2) for s in spans if s for x in (s[0] + 0.1, (s[0] + s[1]) / 2, s[1] - 0.05)} | {0.0, round(end_t, 2)})
tiles = os.path.join(out, "tiles")
os.makedirs(tiles, exist_ok=True)
for i, t in enumerate(times):
    frame(t, os.path.join(tiles, f"{i:03d}.png"), f"scale=240:-1,drawtext=text='{t}':x=6:y=6:fontsize=20:fontcolor=yellow")
cols_n = 8
run("ffmpeg", "-v", "error", "-y", "-pattern_type", "glob", "-i", os.path.join(tiles, "*.png"), "-vf", f"tile={cols_n}x{math.ceil(len(times) / cols_n)}", os.path.join(out, "grid.png"))

report["lines"] = [{"line": lines[k], "start": round(s[0], 2), "end": round(s[1], 2)} if s else {"line": lines[k]} for k, s in enumerate(spans)]
json.dump(report, open(os.path.join(out, "review.json"), "w"), ensure_ascii=False, indent=1)
print(json.dumps(report, ensure_ascii=False, indent=1))
