"""Instruments and master shared by every film score; each film keeps its own arrangement."""
import numpy as np
from scipy.signal import butter, sosfilt, fftconvolve
from scipy.io import wavfile

SR = 48000
# One seeded stream feeds every noise source, so the same cues always give the same file.
rng = np.random.default_rng(7)


class Track:
    def __init__(self, dur):
        self.dur = dur
        self.N = int(dur * SR)
        self.L = np.zeros(self.N)
        self.R = np.zeros(self.N)

    def add(self, sig, at, pan=0.0, gain=1.0):
        i = idx(at)
        if i >= self.N: return
        sig = sig[: self.N - i] * gain
        self.L[i:i + len(sig)] += sig * np.sqrt(0.5 * (1 - pan))
        self.R[i:i + len(sig)] += sig * np.sqrt(0.5 * (1 + pan))


def t_(n): return np.arange(n) / SR
def idx(s): return int(s * SR)


def env(n, a=0.005, d=0.2, curve=4.0):
    t = t_(n)
    e = np.minimum(t / max(a, 1e-4), 1.0) * np.exp(-np.maximum(t - a, 0) * curve / max(d, 1e-4))
    return e


def lp(x, f, o=2):
    return sosfilt(butter(o, min(f, SR / 2 - 100) / (SR / 2), "low", output="sos"), x)


def hp(x, f, o=2):
    return sosfilt(butter(o, f / (SR / 2), "high", output="sos"), x)


def bp(x, lo, hi, o=2):
    return sosfilt(butter(o, [lo / (SR / 2), hi / (SR / 2)], "band", output="sos"), x)


def note(n): return 440.0 * 2 ** ((n - 69) / 12)


# ---------- instruments ----------
def kick(punch=1.0, length=0.45):
    n = idx(length); t = t_(n)
    f = 45 + 110 * np.exp(-t * 28)
    ph = 2 * np.pi * np.cumsum(f) / SR
    body = np.sin(ph) * np.exp(-t * 7)
    click = hp(rng.standard_normal(n), 2500) * np.exp(-t * 180) * 0.25
    return np.tanh((body + click) * 1.6 * punch)


def sub_boom(length=4.0, f0=62, f1=28):
    n = idx(length); t = t_(n)
    f = f1 + (f0 - f1) * np.exp(-t * 2.2)
    ph = 2 * np.pi * np.cumsum(f) / SR
    s = np.sin(ph) * np.exp(-t * 1.1)
    return np.tanh(s * 2.2) * 0.9


def impact(length=5.0):
    n = idx(length); t = t_(n)
    noise = rng.standard_normal(n)
    crack = hp(noise, 1200) * np.exp(-t * 30) * 0.6
    body = lp(noise, 900) * np.exp(-t * 3.5) * 1.4
    return sub_boom(length) + crack + body


def braam(root_midi, length=3.5):
    n = idx(length); t = t_(n)
    s = np.zeros(n)
    for m in (root_midi - 12, root_midi, root_midi + 7, root_midi + 12):
        for det in (-0.12, 0.0, 0.11):
            f = note(m + det)
            s += 2 * ((t * f) % 1.0) - 1
    s = lp(s, 380, 2) + lp(s, 1400, 2) * np.exp(-t * 3) * 0.6
    e = np.minimum(t / 0.04, 1) * np.exp(-t * 0.9)
    return np.tanh(s * 0.12 * e * 3) * 0.7


def ping(high=True):
    n = idx(0.9); t = t_(n)
    f1, f2 = (1318.5, 1975.5) if high else (987.8, 1318.5)
    a = np.sin(2 * np.pi * f1 * t) * env(n, 0.002, 0.25, 5)
    b = np.sin(2 * np.pi * f2 * t) * env(n, 0.002, 0.35, 5)
    b[: idx(0.07)] = 0
    return (a * 0.5 + np.roll(b, idx(0.07)) * 0.5) * 0.35


def tick():
    n = idx(0.05); t = t_(n)
    return hp(rng.standard_normal(n), 3000) * np.exp(-t * 400) * 0.5


def hat(open_=False):
    n = idx(0.25 if open_ else 0.06); t = t_(n)
    return hp(rng.standard_normal(n), 7000) * np.exp(-t * (18 if open_ else 90)) * 0.22


def clap():
    n = idx(0.3); t = t_(n)
    x = bp(rng.standard_normal(n), 900, 5000)
    e = np.zeros(n)
    for o in (0, 0.011, 0.022):
        i = idx(o); e[i:] += np.exp(-t[: n - i] * 60)
    e += np.exp(-t * 14) * 0.5
    return x * e * 0.45


def tap_click():
    n = idx(0.12); t = t_(n)
    s = np.sin(2 * np.pi * 1800 * t) * np.exp(-t * 90) + hp(rng.standard_normal(n), 4000) * np.exp(-t * 200) * 0.4
    return s * 0.5


def whoosh(length, rise=True):
    n = idx(length); t = t_(n)
    noise = rng.standard_normal(n)
    out = np.zeros(n)
    seg = 2048
    for s0 in range(0, n, seg):
        p = (s0 / n) if rise else 1 - s0 / n
        f = 300 + 7000 * p ** 2
        chunk = noise[s0:s0 + seg]
        out[s0:s0 + len(chunk)] = bp(chunk, max(f * 0.6, 100), min(f * 1.4, 20000), 1)
    e = (t / length) ** 2.2 if rise else (1 - t / length) ** 2
    return out * e * 0.5


def riser_tone(length, f0=110, f1=880):
    n = idx(length); t = t_(n)
    f = f0 * (f1 / f0) ** (t / length)
    ph = 2 * np.pi * np.cumsum(f) / SR
    s = (2 * ((ph / (2 * np.pi)) % 1) - 1) + (2 * ((ph * 1.007 / (2 * np.pi)) % 1) - 1)
    return lp(s, 3000) * (t / length) ** 2 * 0.12


def pluck(m, length=0.35, bright=4000):
    n = idx(length); t = t_(n)
    f = note(m)
    s = 2 * ((t * f) % 1) - 1 + 0.5 * (2 * ((t * f * 1.005) % 1) - 1)
    s = lp(s, bright, 2) * env(n, 0.003, length, 6)
    return s * 0.16


def pad(midis, length, cutoff=1800):
    n = idx(length); t = t_(n)
    s = np.zeros(n)
    for m in midis:
        for det in (-0.08, 0.0, 0.09):
            f = note(m + det)
            s += 2 * ((t * f + rng.random()) % 1) - 1
    s = lp(s, cutoff, 2)
    e = np.minimum(t / 0.3, 1) * np.minimum((length - t) / 0.4, 1)
    return s * e * 0.035


def bassline(m, length):
    n = idx(length); t = t_(n)
    f = note(m)
    s = np.tanh(np.sin(2 * np.pi * f * t) * 2 + 0.3 * (2 * ((t * f) % 1) - 1))
    return lp(s, 600) * env(n, 0.005, length, 2.5) * 0.35


# ---------- master: sidechain, reverb, glue ----------
def master(tr, out, pump=(), beat=0.5):
    """pump: the beats whose kick ducks everything above the bass."""
    N, L, R = tr.N, tr.L, tr.R
    side = np.ones(N)
    for tb in pump:
        i = idx(tb); n = idx(beat)
        side[i:i + n] = np.minimum(side[i:i + n], 0.35 + 0.65 * np.minimum(t_(n)[: max(0, min(n, N - i))] / 0.22, 1))
    low_L, low_R = lp(L, 150), lp(R, 150)
    L = low_L + (L - low_L) * side
    R = low_R + (R - low_R) * side

    ir_n = idx(2.6); ir_t = t_(ir_n)
    irL = rng.standard_normal(ir_n) * np.exp(-ir_t * 2.4); irR = rng.standard_normal(ir_n) * np.exp(-ir_t * 2.4)
    irL, irR = lp(irL, 5000), lp(irR, 5000)
    wetL = fftconvolve(hp(L, 250), irL)[:N]; wetR = fftconvolve(hp(R, 250), irR)[:N]
    L = L + wetL / np.max(np.abs(wetL)) * 0.18 * np.max(np.abs(L))
    R = R + wetR / np.max(np.abs(wetR)) * 0.18 * np.max(np.abs(R))

    fade = np.ones(N); fn = idx(1.5); fade[-fn:] = np.linspace(1, 0, fn)
    st = np.stack([L, R], 1) * fade[:, None]
    st = st / np.max(np.abs(st)) * 1.6
    st = np.tanh(st) / np.tanh(1.6) * 0.93
    wavfile.write(out, SR, (st * 32767).astype(np.int16))
