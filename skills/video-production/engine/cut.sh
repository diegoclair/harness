#!/bin/sh
# Short cut: bar-aligned film segments from cut.json, with the soundtrack trimmed to the same segments.
set -e
ENGINE=$(dirname "$0")
AUDIO=${1:-score.wav}
SEGS=$(python3 -c "import json; print(' '.join(f'{a}:{b}' for a, b in json.load(open('cut.json'))))")
EDL=$(python3 -c "import json; print(json.dumps(json.load(open('cut.json')), separators=(',', ':')))")
EDL="$EDL" node "$ENGINE/render.mjs" v pic-v-cut.mp4 >/dev/null
FC=""; CAT=""; i=0
for seg in $SEGS; do
  a=${seg%:*}; b=${seg#*:}; d=$(python3 -c "print($b-$a-0.03)")
  FC="$FC[0:a]atrim=$a:$b,asetpts=PTS-STARTPTS,afade=t=in:d=0.03,afade=t=out:st=$d:d=0.03[s$i];"; CAT="$CAT[s$i]"; i=$((i+1))
done
ffmpeg -y -loglevel error -i "$AUDIO" -filter_complex "${FC}${CAT}concat=n=$i:v=0:a=1[a]" -map "[a]" "${AUDIO%.wav}-cut.wav"
