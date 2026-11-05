// Pure logic of a declarative film, shared by the page and the command so both read a film the same way.

const escapeRe = s => s.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
const wordRe = word => new RegExp(`(?<![\\p{L}\\p{N}])${escapeRe(word)}(?![\\p{L}\\p{N}])`, "gu");

// A line's text marks caption chunks with "|" and wraps in [] what the screen already says.
export const spoken = text => text.replace(/[[\]]/g, "").split("|").map(s => s.trim()).join(" ");

// A spelling is tuned to the voice that was heard saying it; another voice reads the word as written.
const forVoice = voice => e => !e.voice || e.voice === voice;

// Longest word first, so a phrase wins over a word inside it.
export function applyPronunciation(text, dictionary, voice) {
  const entries = dictionary.entries.filter(e => e.say && forVoice(voice)(e)).sort((a, b) => b.word.length - a.word.length);
  return entries.reduce((out, e) => out.replace(wordRe(e.word), e.say), text);
}

export function resolveLine(line, dictionary, voice) {
  const say = applyPronunciation(line.say ?? spoken(line.text), dictionary, voice);
  const known = dictionary.takes.filter(forVoice(voice)).find(k => k.say === say);
  return { ...line, say, take: line.take ?? known?.take };
}

// The kit's voices are the approved ones: a film names a voice only to leave the kit's default.
export function resolveVoice(kit, filmVoice = {}) {
  const spec = { tempo: 1, maxPause: 0, ...filmVoice };
  const voice = spec.voice ?? kit.voice, known = kit.voices?.[voice];
  if (!known) throw new Error(`unknown voice ${voice}: approve it by ear and add it to the kit's voices`);
  return { ...known, ...spec, voice };
}

// A line that continues the scene before it shares that scene's instance, so one phone can span three lines.
export function sceneInstances(lines) {
  const out = [];
  for (const line of lines) {
    if (line.continues && out.length) out[out.length - 1].lines.push(line.id);
    else out.push({ scene: line.scene, params: line.params ?? {}, fit: line.fit, caption: line.caption, lines: [line.id] });
  }
  return out;
}

// A hook test is a variant, so the body it shares is written once. Its narration is its own: a variant never
// plays the base film's voice by accident.
export function applyVariant(film, name) {
  if (!name) return film;
  const swap = film.variants?.[name];
  if (!swap) throw new Error(`film has no variant "${name}", it has ${Object.keys(film.variants ?? {}).join(", ") || "none"}`);
  const lines = film.lines.flatMap(l => (swap[l.id] === null ? [] : swap[l.id] ? [{ ...swap[l.id], id: l.id }] : [l]));
  return { ...film, lines, narration: swap.narration };
}

export function validateFilm(film, scenes) {
  const errors = [], ids = new Set();
  if (!Array.isArray(film.lines) || !film.lines.length) return ["film has no lines"];
  film.lines.forEach((l, i) => {
    const at = `line ${l.id ?? i + 1}`;
    if (!l.id) errors.push(`${at}: no id`);
    if (l.id === "narration") errors.push(`${at}: "narration" names a variant's pinned voice, pick another id`);
    if (ids.has(l.id)) errors.push(`${at}: repeated id`);
    ids.add(l.id);
    if (!l.text) errors.push(`${at}: no text`);
    if (l.continues) {
      if (i === 0) errors.push(`${at}: the first line has no scene to continue`);
      return;
    }
    const scene = scenes[l.scene];
    if (!scene) { errors.push(`${at}: unknown scene "${l.scene}", the library has ${Object.keys(scenes).join(", ")}`); return; }
    for (const key of scene.required ?? []) if ((l.params ?? {})[key] === undefined) errors.push(`${at}: scene ${l.scene} needs params.${key}`);
  });
  for (const inst of sceneInstances(film.lines)) {
    const want = scenes[inst.scene]?.lines ?? 1;
    if (want > 1 && inst.lines.length !== want) errors.push(`scene ${inst.scene} spans ${want} lines, it has ${inst.lines.length}: mark the next lines with "continues": true`);
  }
  return errors;
}

export function spokenDigits(lines) {
  return lines.filter(l => /\d/.test(l.say)).map(l => l.id);
}

// A take that opens on air starts before the cut, so the voice still lands on it. The first line has nothing before it.
const AIR_SEC = 0.25;
export function lineLead(line, speechStart, isFirst, fallback) {
  if (line.lead !== undefined) return line.lead;
  return !isFirst && speechStart > AIR_SEC ? +(fallback - speechStart).toFixed(3) : fallback;
}
