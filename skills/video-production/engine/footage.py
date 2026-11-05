"""What every clips.py needs from real footage: take a shot out, blur its private regions, and sample it for the privacy check.

Library: from footage import render_shot, read_frames, write_frames, blur, mask_at, key_at
Command: footage.py sample <clips folder> <out folder>
A region is a dict from source second to a box x0, y0, x1, y1 in footage pixels, generous because the camera is handheld.
Raw footage never enters git, so a source inside a repository and an output git would track are refused.
"""
import os, subprocess, sys
import numpy as np
from scipy.ndimage import gaussian_filter

W, H, FPS = 576, 1024, 30
OUT_SCALE = "scale=1080:1920:flags=lanczos,unsharp=5:5:0.4"
PRIVACY_STEP_SEC = 0.5
FEATHER = 7


def inside_git(path):
    folder = path if os.path.isdir(path) else os.path.dirname(os.path.abspath(path))
    return subprocess.run(["git", "-C", folder, "rev-parse", "--is-inside-work-tree"], capture_output=True).returncode == 0


def refuse_private_leak(source, out):
    if inside_git(source):
        raise SystemExit(f"raw footage inside a git repository: move {source} out of it")
    os.makedirs(out, exist_ok=True)
    if inside_git(out) and subprocess.run(["git", "-C", out, "check-ignore", "-q", "."]).returncode != 0:
        raise SystemExit(f"{out} is not ignored by git: treated frames of real footage stay out of history")


def key_at(keys, t):
    ts = sorted(keys)
    if t <= ts[0]: return keys[ts[0]]
    if t >= ts[-1]: return keys[ts[-1]]
    for a, b in zip(ts, ts[1:]):
        if a <= t <= b:
            k = (t - a) / (b - a)
            return tuple(u + (v - u) * k for u, v in zip(keys[a], keys[b]))


def mask_at(regions, t, size=(H, W)):
    m = np.zeros(size)
    for keys in regions:
        x0, y0, x1, y1 = (int(v) for v in key_at(keys, t))
        m[max(0, y0):y1, max(0, x0):x1] = 1
    # The mask only grows at its edge: a feather that eats inward would let a sharp sliver through.
    return np.maximum(m, gaussian_filter(m, FEATHER))


def blur(f, regions, t, sigma=9):
    if not regions: return f
    m = mask_at(regions, t, f.shape[:2])[..., None]
    return f * (1 - m) + gaussian_filter(f, (sigma, sigma, 0)) * m


def read_frames(src, a, b, dur):
    n = round(dur * FPS)
    speed = (b - a) / dur
    dec = subprocess.Popen(["ffmpeg", "-v", "error", "-ss", str(a), "-t", str(b - a + 0.2), "-i", src,
                            "-vf", f"setpts=PTS/{speed},fps={FPS},scale={W}:{H}", "-f", "rawvideo", "-pix_fmt", "rgb24", "-"], stdout=subprocess.PIPE)
    for i in range(n):
        raw = dec.stdout.read(W * H * 3)
        if len(raw) < W * H * 3: raise SystemExit(f"footage ended at frame {i} of {n}")
        yield a + i / FPS * speed, np.frombuffer(raw, np.uint8).reshape(H, W, 3).astype(np.float32)
    dec.kill()


def write_frames(frames, out):
    enc = subprocess.Popen(["ffmpeg", "-y", "-v", "error", "-f", "rawvideo", "-pix_fmt", "rgb24", "-s", f"{W}x{H}", "-r", str(FPS), "-i", "-",
                            "-vf", OUT_SCALE, "-q:v", "3", os.path.join(out, "%04d.jpg")], stdin=subprocess.PIPE)
    count = 0
    for f in frames:
        enc.stdin.write(f.clip(0, 255).astype(np.uint8).tobytes())
        count += 1
    enc.stdin.close(); enc.wait()
    return count


def render_shot(name, shot, footage, root="clips"):
    """shot: src, a, b, dur in seconds, regions, and optional sigma; `light` regions get a blur that keeps the subject readable."""
    src, out = os.path.join(footage, shot["src"]), os.path.join(root, name)
    refuse_private_leak(src, out)
    sigma = shot.get("sigma", 9)
    n = write_frames((blur(blur(f, shot["regions"], t, sigma), shot.get("light", []), t, 4) for t, f in read_frames(src, shot["a"], shot["b"], shot["dur"])), out)
    print(name, n, "frames")


def sample(clips, out):
    """Every half second of every shot goes to one sheet per shot, for a person to look at: no detector clears real data."""
    os.makedirs(out, exist_ok=True)
    every = round(PRIVACY_STEP_SEC * FPS)
    made = []
    for root, _, files in sorted(os.walk(clips)):
        jpgs = sorted(f for f in files if f.endswith(".jpg"))
        if not jpgs: continue
        name = os.path.relpath(root, clips).replace(os.sep, "-")
        picked = os.path.join(out, f"{name}-privacidade.jpg")
        cols = 6
        rows = max(1, -(-len(jpgs[::every]) // cols))
        subprocess.run(["ffmpeg", "-y", "-v", "error", "-framerate", str(FPS), "-i", os.path.join(root, "%04d.jpg"),
                        "-vf", f"select='not(mod(n\\,{every}))',scale=300:-1,tile={cols}x{rows}", "-frames:v", "1", "-update", "1", picked], check=True)
        made.append(picked)
    return made


if __name__ == "__main__":
    if len(sys.argv) != 4 or sys.argv[1] != "sample": sys.exit(__doc__)
    print("\n".join(sample(sys.argv[2], sys.argv[3])))
