import { copyFileSync, existsSync, mkdirSync, readFileSync, writeFileSync } from "node:fs";
import path from "node:path";
import { execFileSync } from "node:child_process";
import { kitFile, loadKit } from "./kit.mjs";
import { applyVariant, resolveLine, resolveVoice } from "./film-core.mjs";
import { filmVoice, measure, narrationSpans, paced, pacedName, readsWhole, speak, voiceKey } from "./narration.mjs";
import { VOICE_CACHE } from "./setup.mjs";

const readJson = file => JSON.parse(readFileSync(file, "utf8"));
const kit = loadKit(process.env.FILM_KIT);
const keys = { file: kit.env ? kitFile(kit, kit.env) : null, names: kit.keys };
const film = applyVariant(readJson("film.json"), new URLSearchParams(process.env.FILM_QUERY ?? "").get("variant"));
const spec = resolveVoice(kit, film.voice);
const dictionary = readJson(kitFile(kit, kit.dictionary));
const lines = film.lines.map(l => resolveLine(l, dictionary, spec.voice));
const spendingOff = !!process.env.FILM_NO_NEW_VOICE;
const LINE_BREAK = " <long pause> ";

// One reader for a narration's spans, pinned or found, so a line the spans do not hold fails naming it.
function narrationLines({ file, spans }) {
  for (const f of [file, spans]) if (!existsSync(f)) throw new Error(`the film's narration ${f} is missing`);
  const byLine = readJson(spans).lines;
  return lines.map(l => {
    if (!byLine[l.id]) throw new Error(`${spans} has no line ${l.id}`);
    return { file, ...byLine[l.id] };
  });
}

async function cachedOrSpoken(name, say) {
  const cached = path.join(VOICE_CACHE, `${name}.wav`);
  if (existsSync(cached)) return cached;
  if (spendingOff) throw new Error(`no narration voice/${name}.wav in the film and none in ${VOICE_CACHE}, and spending is off: ${say}`);
  await speak(spec, say, keys, cached);
  return cached;
}

function keep(name, source, spans) {
  mkdirSync("voice", { recursive: true });
  copyFileSync(source, path.join("voice", `${name}.wav`));
  writeFileSync(path.join("voice", `${name}.spans.json`), `${JSON.stringify(spans, null, 1)}\n`);
}

async function wholeNarration() {
  const say = lines.map(l => l.say).join(LINE_BREAK), name = voiceKey(spec, say, spec.take);
  if (!filmVoice(".", name)) {
    const file = await cachedOrSpoken(name, say);
    const spans = narrationSpans(file, lines.map(l => l.say));
    keep(name, file, { lines: Object.fromEntries(lines.map((l, i) => [l.id, spans[i]])) });
  }
  return narrationLines({ file: path.join("voice", `${name}.wav`), spans: path.join("voice", `${name}.spans.json`) });
}

async function lineClips() {
  const out = [];
  for (const l of lines) {
    const key = voiceKey(spec, l.say, l.take), name = pacedName(key, spec.tempo, spec.maxPause);
    if (!filmVoice(".", name)) {
      const clip = path.join(VOICE_CACHE, `${name}.wav`);
      if (!existsSync(clip)) {
        const raw = await cachedOrSpoken(key, l.say);
        if (raw !== clip) paced(raw, clip, spec.tempo, spec.maxPause);
      }
      keep(name, clip, measure(clip));
    }
    out.push({ file: path.join("voice", `${name}.wav`), ...readJson(path.join("voice", `${name}.spans.json`)) });
  }
  return out;
}

async function sample(lineId, outDir, voices) {
  const line = lines.find(l => l.id === lineId);
  if (!line) throw new Error(`no line ${lineId}`);
  mkdirSync(outDir, { recursive: true });
  for (const voice of voices.split(",")) {
    const other = resolveVoice(kit, { ...film.voice, voice });
    const raw = path.join(VOICE_CACHE, `${voiceKey(other, line.say)}.wav`);
    if (!existsSync(raw)) await speak(other, line.say, keys, raw);
    const out = path.join(outDir, `${lineId}-${voice}.mp3`);
    execFileSync("ffmpeg", ["-y", "-loglevel", "error", "-i", raw, "-af", "loudnorm=I=-16:TP=-1.5", "-ar", "48000", "-b:a", "192k", out]);
    console.log(out, measure(raw));
  }
}

const [flag, ...rest] = process.argv.slice(2);
if (flag === "--sample") await sample(...rest);
else {
  const measured = film.narration ? narrationLines(film.narration) : readsWhole(spec.model) ? await wholeNarration() : await lineClips();
  const entries = lines.map((l, i) => {
    const m = measured[i], burst = (m.pauses[0]?.[0] ?? m.speech[1]) - m.speech[0];
    // A stumbled first syllable shows up as a tiny voiced burst before the first pause.
    console.log(`${l.id}: ${m.speech[1].toFixed(2)} s of speech, first burst ${burst.toFixed(2)} s${burst < 0.35 ? " (check the start)" : ""}`);
    return [l.id, { say: l.say, take: l.take ?? null, voice: spec.voice, model: spec.model, instructions: spec.instructions, tempo: spec.tempo, pin: film.narration?.spans ?? null, ...m }];
  });
  mkdirSync(".build", { recursive: true });
  writeFileSync(path.join(".build", "voice.json"), JSON.stringify({ lines: Object.fromEntries(entries) }, null, 1));
}
