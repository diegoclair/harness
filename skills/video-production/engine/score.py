"""The shared arrangement of a declarative film: the groove waits for the hook to land, and each hit falls on a cue.

usage: score.py cues.json score.wav
A film that needs its own arc keeps a music.py beside film.json, and the film command prefers it.
"""
import json, sys
from synth import *

cues = json.load(open(sys.argv[1]))
DUR, beat, finale = cues["duration"], cues["beat"], cues["finale"]
hit = cues.get("hit", 0.0)
tr = Track(DUR)
add = tr.add
ev = lambda key: cues.get(key, [])

prog = [(57, [57, 60, 64]), (53, [53, 57, 60]), (48, [48, 52, 55]), (55, [55, 59, 62])]
bar = 4 * beat
t0, b = 0.0, 0
while t0 < finale - 0.01:
    root, chord = prog[b % 4]
    add(pad([c + 12 for c in chord], bar + 0.2), t0, gain=0.9)
    for k in range(4):
        tb = t0 + k * beat
        add(hat(), tb + beat / 2, pan=0.35)
        # Holding the kick back makes the hook's turn the moment the film starts to move.
        if tb >= hit - 0.01:
            add(kick(1.0), tb, gain=0.8)
            if k in (1, 3): add(clap(), tb, pan=-0.1, gain=0.6)
            add(bassline(root - 24, beat * 0.9), tb + beat / 2)
    arp = [chord[0] + 12, chord[1] + 12, chord[2] + 12, chord[1] + 24]
    for s in range(8):
        add(pluck(arp[s % 4], 0.3, 2500 + 400 * (b % 4)), t0 + s * beat / 2, pan=(0.4 if s % 2 else -0.4))
    b += 1; t0 += bar

if hit:
    add(impact(2.0), hit, gain=0.3)
    add(kick(1.0), hit, gain=1.0)
for at in cues["cuts"][:-1]:
    add(whoosh(0.4, rise=False), at - 0.05, gain=0.5)
if "roll" in cues:
    a, z = cues["roll"]
    add(riser_tone(z - a, 880, 110), a, gain=0.5)
if "land" in cues:
    add(sub_boom(1.5), cues["land"], gain=0.6)
    add(ping(True), cues["land"], gain=0.5)
for i, at in enumerate(ev("ticks")):
    add(tick(), at, gain=1.0, pan=-0.2 + 0.2 * i)
for at in ev("taps") + ev("strikes"):
    add(tap_click(), at, gain=1.0)
for at in ev("pings"):
    add(ping(True), at, gain=0.75, pan=0.2)
for at in ev("beeps") + ev("done"):
    add(ping(False), at, gain=0.7)
for at in ev("zooms") + ev("softCuts"):
    add(whoosh(0.25, rise=False), at - 0.03, gain=0.3)
for i, at in enumerate(ev("cascade")):
    add(pluck([69, 76, 72, 81][i % 4] + 12, 0.6, 4200), at, gain=1.1, pan=(-0.3 + 0.2 * i))

add(whoosh(2 * beat), finale - 2 * beat, gain=0.9)
add(riser_tone(2 * beat, 220, 1760), finale - 2 * beat, gain=0.7)
add(impact(5.0), finale, gain=0.4)
add(braam(33, 4.0), finale, gain=0.25)
add(pad([57, 61, 64, 69, 73], DUR - finale - 0.1, cutoff=2400), finale + 0.1, gain=0.9)
for i, m in enumerate([69, 73, 76, 81]):
    add(pluck(m, 1.2, 5000), finale + 0.6 + i * 0.18, gain=0.6, pan=(-0.3 + 0.2 * i))

master(tr, sys.argv[2], pump=np.arange(hit, finale, beat), beat=beat)
print("wrote", sys.argv[2], DUR, "s")
