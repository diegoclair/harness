import { createHash } from "node:crypto";
import { execFileSync, spawnSync } from "node:child_process";
import { existsSync, mkdirSync, readFileSync, renameSync, unlinkSync, writeFileSync } from "node:fs";
import path from "node:path";
import { ENGINE } from "./mounts.mjs";
import { PYTHON } from "./setup.mjs";

// A re-render never pays twice: the clip is keyed by everything that shapes the audio.
// A take number asks for another generation of the same text; without one the key stays what it always was.
export function voiceKey({ model, voice, instructions }, say, take) {
  return createHash("sha256").update(JSON.stringify({ model, voice, instructions, say, ...(take ? { take } : {}) })).digest("hex").slice(0, 24);
}

// A key never leaves memory: the environment wins, and the kit's untracked env file is the fallback, where the kit may
// keep the provider's key under a name of its own.
function apiKey({ file, names = {} }, name) {
  if (process.env[name]) return process.env[name];
  const stored = names[name] ?? name;
  const line = file && existsSync(file) && readFileSync(file, "utf8").split("\n").find(l => l.startsWith(`${stored}=`));
  if (!line) throw new Error(`no key: set ${name}${file ? ` or ${stored} in ${file}` : ""}`);
  return line.slice(line.indexOf("=") + 1).trim().replace(/^["']|["']$/g, "");
}

async function openaiSpeech({ model, voice, instructions }, say, keys) {
  const res = await fetch("https://api.openai.com/v1/audio/speech", {
    method: "POST",
    headers: { Authorization: `Bearer ${apiKey(keys, "OPENAI_API_KEY")}`, "Content-Type": "application/json; charset=utf-8" },
    body: Buffer.from(JSON.stringify({ model, voice, instructions, input: say, response_format: "wav" }), "utf8"),
  });
  if (!res.ok) throw new Error(`speech ${res.status}: ${(await res.text()).slice(0, 300)}`);
  return Buffer.from(await res.arrayBuffer());
}

// Gemini reads the text as a verbatim transcript, so the delivery goes in the style annotation, never in the text.
async function geminiSpeech({ model, voice, instructions }, say, keys) {
  const res = await fetch("https://generativelanguage.googleapis.com/v1beta/interactions", {
    method: "POST",
    headers: { "x-goog-api-key": apiKey(keys, "GEMINI_API_KEY"), "Content-Type": "application/json; charset=utf-8" },
    body: Buffer.from(JSON.stringify({
      model,
      input: [{ type: "user_input", content: [{ type: "text", text: say, annotations: [{ type: "speech_metadata", style: instructions }] }] }],
      response_format: { type: "audio" },
      generation_config: { speech_config: [{ voice }] },
    }), "utf8"),
  });
  if (!res.ok) throw new Error(`speech ${res.status}: ${(await res.text()).slice(0, 300)}`);
  const audio = (await res.json()).steps?.filter(s => s.type === "model_output").flatMap(s => s.content ?? []).filter(c => c.type === "audio").at(-1);
  if (!audio) throw new Error(`speech without audio: ${say}`);
  return Buffer.from(audio.data, "base64");
}

// Gemini picks a new delivery on every call, so a line generated alone sounds like another person: its films are read
// in one breath and play whole, and each line is only located inside the narration.
export const readsWhole = model => model.startsWith("gemini-");

export async function speak(spec, say, keys, file) {
  const audio = await (readsWhole(spec.model) ? geminiSpeech : openaiSpeech)(spec, say, keys);
  // A streamed wav carries an open-ended size in its header; a rewritten header reads cleanly everywhere.
  // Films run in parallel share the cache, so a clip only ever appears whole.
  const part = `${file}.${process.pid}.part`, tmp = `${part}.wav`;
  mkdirSync(path.dirname(file), { recursive: true });
  writeFileSync(part, audio);
  execFileSync("ffmpeg", ["-y", "-loglevel", "error", "-f", "wav", "-i", part, "-c", "copy", tmp]);
  renameSync(tmp, file);
  unlinkSync(part);
}

export const pacedName = (key, tempo = 1, maxPause = 0) => (tempo === 1 && !maxPause ? key : `${key}-t${tempo}${maxPause ? `-p${maxPause}` : ""}`);

// The model ignores pace in its instructions, so a short-video pace is a pitch-preserving stretch of a line's clip.
export function paced(file, out, tempo = 1, maxPause = 0) {
  // A fast-cut piece keeps the words and loses the air between them, since stretching stops at 1.05x.
  const tight = maxPause ? [`silenceremove=stop_periods=-1:stop_duration=${maxPause}:stop_threshold=-40dB:stop_silence=${maxPause}`] : [];
  const filters = [...tight, ...(tempo === 1 ? [] : [`atempo=${tempo}`])].join(",");
  const tmp = `${out}.${process.pid}.wav`;
  execFileSync("ffmpeg", ["-y", "-loglevel", "error", "-i", file, ...(filters ? ["-af", filters] : []), tmp]);
  renameSync(tmp, out);
}

// The clip's own silence is not speech: captions and scene ends follow the voiced span.
export function measure(file) {
  const dur = +execFileSync("ffprobe", ["-v", "error", "-show_entries", "format=duration", "-of", "csv=p=0", file]).toString().trim();
  const log = spawnSync("ffmpeg", ["-hide_banner", "-i", file, "-af", "silencedetect=noise=-40dB:d=0.08", "-f", "null", "-"]).stderr.toString();
  const starts = [...log.matchAll(/silence_start: ([\d.]+)/g)].map(m => +m[1]);
  const ends = [...log.matchAll(/silence_end: ([\d.]+)/g)].map(m => +m[1]);
  const gaps = starts.map((a, i) => [a, ends[i] ?? dur]);
  const head = gaps.length && gaps[0][0] < 0.01 ? gaps.shift()[1] : 0;
  const tail = gaps.length && gaps[gaps.length - 1][1] > dur - 0.02 ? gaps.pop()[0] : dur;
  const r = x => +x.toFixed(3);
  return { dur: r(dur), speech: [r(head), r(tail)], pauses: gaps.map(([a, b]) => [r(a), r(b)]) };
}

function locate(file, says) {
  const list = `${file}.${process.pid}.lines.json`;
  writeFileSync(list, JSON.stringify(says));
  const res = spawnSync(PYTHON, [path.join(ENGINE, "align.py"), file, list], { encoding: "utf8", env: { ...process.env, OMP_NUM_THREADS: "2" } });
  unlinkSync(list);
  if (res.status !== 0) throw new Error(`align ${file}: ${(res.stderr || String(res.error)).trim().split("\n").at(-1)}`);
  return JSON.parse(res.stdout);
}

// Each line owns the narration from the middle of the silence before it, so a scene cut never lands on a word.
function lineSpans(dur, pauses, spans) {
  // Word timestamps run a little into the silence around a word; the measured silence is where the voice really stops.
  const inside = t => pauses.find(([a, b]) => a <= t && t <= b);
  const voiced = spans.map(({ start, end }) => ({ start: inside(start)?.[1] ?? start, end: inside(end)?.[0] ?? end }));
  const cuts = [0, ...voiced.slice(1).map((s, i) => {
    const prev = voiced[i].end, gap = pauses.filter(([a, b]) => b > prev && a < s.start).sort((x, y) => (y[1] - y[0]) - (x[1] - x[0]))[0];
    return gap ? (gap[0] + gap[1]) / 2 : (prev + s.start) / 2;
  }), dur];
  const r = x => +x.toFixed(3);
  return voiced.map((s, i) => ({
    offset: r(cuts[i]), dur: r(cuts[i + 1] - cuts[i]),
    speech: [r(s.start - cuts[i]), r(s.end - cuts[i])],
    pauses: pauses.filter(([a, b]) => a > s.start && b < s.end).map(([a, b]) => [r(a - cuts[i]), r(b - cuts[i])]),
  }));
}

export function narrationSpans(file, says) {
  const { dur, pauses } = measure(file);
  return lineSpans(dur, pauses, locate(file, says));
}

export function filmVoice(filmDir, name) {
  const wav = path.join(filmDir, "voice", `${name}.wav`), spans = path.join(filmDir, "voice", `${name}.spans.json`);
  return existsSync(wav) && existsSync(spans) ? { wav, spans } : null;
}
