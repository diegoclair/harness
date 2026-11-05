"""Where each line sits inside one narration, read from the words the voice actually said."""
import difflib, json, re, sys, unicodedata

import whisper_model

audio, lines_json = sys.argv[1], sys.argv[2]
lines = json.load(open(lines_json))


def tokens(text):
    text = unicodedata.normalize("NFKD", re.sub(r"<[^>]*>", " ", text)).encode("ascii", "ignore").decode().lower()
    return re.findall(r"[a-z0-9]+", text)


segments, _ = whisper_model.load().transcribe(audio, language=whisper_model.LANGUAGE, beam_size=5, word_timestamps=True, condition_on_previous_text=False)
words = [w for s in segments for w in s.words]
heard, owner = [], []
for i, w in enumerate(words):
    for t in tokens(w.word):
        heard.append(t)
        owner.append(i)

script, line_of = [], []
for k, text in enumerate(lines):
    for t in tokens(text):
        script.append(t)
        line_of.append(k)

# A figure the voice says in words comes back as digits, so a line is placed by the words that do match.
matched = [[] for _ in lines]
for a, b, n in difflib.SequenceMatcher(None, script, heard, autojunk=False).get_matching_blocks():
    for j in range(n):
        matched[line_of[a + j]].append(owner[b + j])

out = []
for k, text in enumerate(lines):
    total = sum(1 for x in line_of if x == k)
    heard_well = matched[k] and len(matched[k]) >= total / 2
    out.append({"start": round(words[min(matched[k])].start, 3), "end": round(words[max(matched[k])].end, 3)} if heard_well else None)
# A line spoken as figures transcribes as digits; it is whatever the voice says between its neighbours.
for k, span in enumerate(out):
    if span: continue
    prev, nxt = (out[k - 1] if k else None), (out[k + 1] if k + 1 < len(out) else None)
    if (k and not prev) or (k + 1 < len(out) and not nxt):
        sys.exit(f"lines {k} to {k + 2} were not heard in the narration: {lines[k]}")
    inner = [w for w in words if (not prev or w.start >= prev["end"]) and (not nxt or w.end <= nxt["start"])]
    if not inner:
        sys.exit(f"line {k + 1} was not heard in the narration: {lines[k]}")
    out[k] = {"start": round(inner[0].start, 3), "end": round(inner[-1].end, 3)}
print(json.dumps(out))
