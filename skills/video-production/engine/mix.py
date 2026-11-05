"""Voice over score: the music steps back while a line is spoken, and the mix lands at the loudness the networks play.

usage: mix.py cues.json score.wav out.wav
"""
import json, re, subprocess, sys, tempfile, os
import numpy as np
from scipy.io import wavfile
from scipy.ndimage import minimum_filter1d, uniform_filter1d
from scipy.signal import resample_poly

SR = 48000
VOICE_RMS_DB = -20.0
# The music stays part of the piece between lines, and steps back without disappearing while one is spoken.
BED_DB = -3.0
DUCK_DB = -8.0
PRE, POST, RAMP = 0.15, 0.3, 0.25
TARGET_I, TARGET_TP = -14.0, -1.0

cues = json.load(open(sys.argv[1]))
sr, score = wavfile.read(sys.argv[2])
assert sr == SR, "score must be 48 kHz"
music = score.astype(np.float64) / 32768.0
N = len(music)


def rms_db(x):
    return 20 * np.log10(np.sqrt(np.mean(x ** 2)) + 1e-12)


def read_voice(path):
    vsr, v = wavfile.read(path)
    v = v.astype(np.float64) / 32768.0
    if v.ndim > 1: v = v.mean(axis=1)
    return resample_poly(v, SR, vsr) if vsr != SR else v


def place(v, at, windows):
    """One gain for the whole clip, so the voice never changes level between the lines it carries."""
    voiced = np.concatenate([v[max(0, int(a * SR)):int(b * SR)] for a, b in windows])
    v = v * 10 ** ((VOICE_RMS_DB - rms_db(voiced)) / 20)
    i = int(at * SR)
    if i < 0: v, i = v[-i:], 0
    v = v[: max(0, N - i)]
    voice[i:i + len(v)] += v


voice = np.zeros(N)
spans = [tuple(line["speech"]) for line in cues["voice"]]
narrations = {}
for line in cues["voice"]:
    if "offset" in line:
        narrations.setdefault(line["file"], []).append(line)
        continue
    a, b = line["speech"]
    place(read_voice(line["file"]), line["at"], [(a - line["at"], b - line["at"])])
# A narration plays whole from where its first line sits, so no word is ever cut between two lines.
for path, group in narrations.items():
    start = group[0]["at"] - group[0]["offset"]
    place(read_voice(path), start, [(l["speech"][0] - start, l["speech"][1] - start) for l in group])

# Ducking follows the timeline rather than a detector: it starts before the first syllable instead of
# reacting to it, never pumps on a pause inside a line, and gives the same file on every render.
mask = np.zeros(N)
for a, b in spans:
    mask[max(0, int((a - PRE) * SR)):min(N, int((b + POST) * SR))] = 1
k = np.hanning(int(RAMP * SR)); k /= k.sum()
mask = np.clip(np.convolve(mask, k, mode="same"), 0, 1)

music *= 10 ** ((VOICE_RMS_DB + BED_DB - rms_db(music)) / 20)
duck = (10 ** (DUCK_DB * mask / 20))[:, None]
bed = music * duck
mixed = bed + voice[:, None]


def limit(x, ceiling_db):
    """Peaks held under the ceiling, so the loudness step can stay one flat gain."""
    c = 10 ** (ceiling_db / 20)
    g = np.minimum(1.0, c / np.maximum(np.abs(x).max(axis=1), 1e-9))
    n = int(0.005 * SR)
    g = uniform_filter1d(minimum_filter1d(g, n), n)
    return x * g[:, None], g


def write(path, x):
    wavfile.write(path, SR, (np.clip(x, -1, 1) * 32767).astype(np.int16))


def momentary(path):
    log = subprocess.run(["ffmpeg", "-hide_banner", "-nostats", "-i", path, "-af", "ebur128", "-f", "null", "-"], capture_output=True, text=True).stderr
    return [(float(t), float(m)) for t, m in re.findall(r"t:\s*([\d.]+)\s+TARGET.*?M:\s*(-?[\d.]+)", log)]


def energy_mean(values):
    v = [x for x in values if x > -70]
    return 10 * np.log10(np.mean([10 ** (x / 10) for x in v])) if v else float("nan")


def measured(path):
    log = subprocess.run(["ffmpeg", "-hide_banner", "-nostats", "-i", path, "-af", "ebur128=peak=true", "-f", "null", "-"], capture_output=True, text=True).stderr
    summary = log[log.rindex("Summary:"):]
    return float(re.search(r"I:\s*(-?[\d.]+) LUFS", summary)[1]), float(re.search(r"True peak:\s*Peak:\s*(-?[\d.]+) dBFS", summary)[1])


with tempfile.TemporaryDirectory() as tmp:
    pre = os.path.join(tmp, "pre.wav")
    # Reaching the target with one flat gain would push peaks past the ceiling, so the peaks are held first.
    # Sharp transients peak between samples, so the true peak is measured after the gain and the ceiling tightened.
    g, margin = np.ones(N), 1.0
    for _ in range(6):
        for _ in range(3):
            write(pre, mixed * g[:, None])
            need = TARGET_I - measured(pre)[0]
            limited, g = limit(mixed, TARGET_TP - margin - need)
        write(pre, limited)
        gain = 10 ** ((TARGET_I - measured(pre)[0]) / 20)
        write(sys.argv[3], limited * gain)
        out_i, out_tp = measured(sys.argv[3])
        if out_tp <= TARGET_TP - 0.1: break
        margin += out_tp - TARGET_TP + 0.2
    stems = {}
    for name, x in (("voice", voice[:, None] * g[:, None]), ("ducked", bed * g[:, None]), ("full", music * g[:, None])):
        stems[name] = os.path.join(tmp, f"{name}.wav")
        write(stems[name], np.repeat(x, 2, axis=1) * gain if x.shape[1] == 1 else x * gain)
    # A momentary reading covers the 400 ms before its timestamp, so only windows wholly inside a line count.
    inside = lambda t: any(a <= t - 0.4 and t <= b for a, b in spans)
    lvl = {k: energy_mean([m for t, m in momentary(f) if inside(t)]) for k, f in stems.items()}
    print(f"mix: {out_i:.1f} LUFS integrated, true peak {out_tp:.1f} dBTP")
    print(f"while a line is spoken: voice {lvl['voice']:.1f} LUFS, music {lvl['full']:.1f} LUFS without ducking, {lvl['ducked']:.1f} LUFS ducked ({lvl['ducked'] - lvl['full']:+.1f} dB)")
