"""Maps a film's cues onto its short cut, so the cut's own voice mix is levelled like any other film.

usage: cutcues.py cues.json cut.json out.json
"""
import json, sys

cues, segs = json.load(open(sys.argv[1])), json.load(open(sys.argv[2]))
out, offset, voice = dict(cues), 0.0, []
for a, b in segs:
    for v in cues.get("voice", []):
        # A line cut in the middle is worse than no line, so only whole lines cross into the cut.
        if a <= v["at"] and v["speech"][1] <= b:
            d = offset - a
            voice.append({**v, "at": v["at"] + d, "speech": [s + d for s in v["speech"]]})
    offset += b - a
dropped = {v["id"] for v in cues.get("voice", [])} - {v["id"] for v in voice}
if dropped:
    sys.exit(f"lines cut mid-way by the segments: {', '.join(sorted(dropped))}")
out.update(duration=offset, voice=voice)
json.dump(out, open(sys.argv[3], "w"))
print(f"cut: {offset} s, {len(voice)} lines")
