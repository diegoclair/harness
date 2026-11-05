import test from "node:test";
import assert from "node:assert/strict";
import { execFileSync, spawnSync } from "node:child_process";
import { chmodSync, mkdirSync, mkdtempSync, readFileSync, symlinkSync, writeFileSync } from "node:fs";
import os from "node:os";
import path from "node:path";
import vm from "node:vm";
import { applyPronunciation, applyVariant, lineLead, resolveLine, resolveVoice, sceneInstances, spoken, validateFilm } from "./film-core.mjs";
import { captionSpans, compareTranscript, diffWords, hookEnd, longestStill, lumaStd, parseTranscript, runChecks, shortCaptions, speechCuts, tokens } from "./qa-checks.mjs";
import { resolveRequest } from "./serve.mjs";
import { findKit, serverRoot } from "./kit.mjs";
import { ENGINE, mountTable, registerMounts } from "./mounts.mjs";
import { voiceKey } from "./narration.mjs";
import { scaffold } from "./scaffold.mjs";
import { drawCaption } from "./scenes/captions.mjs";

const timeline = readFileSync(path.join(ENGINE, "timeline.js"), "utf8");
const page = vm.runInNewContext(`${timeline};({ layoutScenes, timeChunks, measurementHolds, captionAt })`, { location: {}, document: {} });
const temp = () => mkdtempSync(path.join(os.tmpdir(), "film-test-"));
const write = (file, text) => { mkdirSync(path.dirname(file), { recursive: true }); writeFileSync(file, typeof text === "string" ? text : JSON.stringify(text)); };
const tone = (file, seconds) => { mkdirSync(path.dirname(file), { recursive: true }); execFileSync("ffmpeg", ["-v", "error", "-y", "-f", "lavfi", "-i", `sine=frequency=300:duration=${seconds}`, "-ar", "24000", file]); };

const dictionary = {
  entries: [
    { word: "Acme", say: "Ácme" },
    { word: "Acme Pay", say: "Ácme Pêi" },
    { word: "txt", say: "tê xis tê", voice: "line-voice" },
  ],
  takes: [{ say: "Oi, Ácme.", take: 2, voice: "line-voice" }],
};
const kit = { voice: "whole-voice", voices: { "whole-voice": { model: "gemini-x", instructions: "calm" }, "line-voice": { model: "tts-x", instructions: "quick" } } };
const LIBRARY = { card: { name: "card", required: ["title"] }, phone: { name: "phone", lines: 2 } };

test("a spelling replaces whole words, the longest phrase first, only for the voice it was heard in", () => {
  assert.equal(applyPronunciation("Acme Pay e Acme, sem Acmes.", dictionary, "whole-voice"), "Ácme Pêi e Ácme, sem Acmes.");
  assert.equal(applyPronunciation("um txt", dictionary, "whole-voice"), "um txt");
  assert.equal(applyPronunciation("um txt", dictionary, "line-voice"), "um tê xis tê");
});

test("the spoken text drops the caption marks, and a known phrase takes the take that worked unless the line asks", () => {
  assert.equal(spoken("[Oi,]|Acme."), "Oi, Acme.");
  assert.equal(resolveLine({ text: "[Oi,]|Acme." }, dictionary, "line-voice").take, 2);
  assert.equal(resolveLine({ text: "[Oi,]|Acme.", take: 3 }, dictionary, "line-voice").take, 3);
  assert.equal(resolveLine({ text: "[Oi,]|Acme." }, dictionary, "whole-voice").take, undefined);
});

test("a film takes the kit's default voice and may only name another the kit approved", () => {
  assert.deepEqual(resolveVoice(kit, {}), { model: "gemini-x", instructions: "calm", tempo: 1, maxPause: 0, voice: "whole-voice" });
  assert.equal(resolveVoice(kit, { voice: "line-voice", take: 3 }).model, "tts-x");
  assert.throws(() => resolveVoice(kit, { voice: "stranger" }), /unknown voice stranger/);
});

test("lines marked as continuing share the scene before them, and a fit or a caption switch belongs to the scene's first line", () => {
  const out = sceneInstances([{ id: "a", scene: "x", fit: { x: 1, y: 2, scale: 0.5 } }, { id: "b", continues: true }, { id: "c", scene: "y", caption: false }]);
  assert.deepEqual(out.map(i => i.lines), [["a", "b"], ["c"]]);
  assert.deepEqual(out[0].fit, { x: 1, y: 2, scale: 0.5 });
  assert.deepEqual(out.map(i => i.caption), [undefined, false]);
});

test("a variant swaps or drops whole lines, keeps the body, and plays only its own narration", () => {
  const pin = { file: "voice/a.wav", spans: "voice/a.spans.json" };
  const film = {
    narration: pin, lines: [{ id: "hook", text: "a", take: 5 }, { id: "also", text: "b" }, { id: "body", text: "c" }],
    variants: { v1: { hook: { text: "d" }, also: null }, v2: { hook: { text: "e" }, narration: { file: "voice/b.wav", spans: "voice/b.spans.json" } } },
  };
  assert.deepEqual(applyVariant(film, "v1").lines, [{ id: "hook", text: "d" }, { id: "body", text: "c" }]);
  assert.equal(applyVariant(film, "v1").narration, undefined);
  assert.equal(applyVariant(film, "v2").narration.file, "voice/b.wav");
  assert.equal(applyVariant(film, null), film);
  assert.throws(() => applyVariant(film, "v9"), /no variant "v9"/);
});

test("a film is refused before any spend", () => {
  assert.deepEqual(validateFilm({ lines: [{ id: "a", text: "x", scene: "card", params: { title: "t" } }] }, LIBRARY), []);
  assert.match(validateFilm({ lines: [{ id: "a", text: "x", scene: "nope" }] }, LIBRARY)[0], /unknown scene "nope"/);
  assert.match(validateFilm({ lines: [{ id: "a", text: "x", scene: "card" }] }, LIBRARY)[0], /params\.title/);
  assert.match(validateFilm({ lines: [{ id: "a", text: "x", scene: "phone" }] }, LIBRARY)[0], /spans 2/);
  assert.match(validateFilm({ lines: [{ id: "narration", text: "x", scene: "card", params: { title: "t" } }] }, LIBRARY)[0], /pinned voice/);
});

test("a take that opens on air starts before the cut, except on the first line", () => {
  assert.equal(lineLead({}, 0.42, false, 0.05), -0.37);
  assert.equal(lineLead({}, 0.42, true, 0.05), 0.05);
  assert.equal(lineLead({}, 0, false, 0.05), 0.05);
  assert.equal(lineLead({ lead: 0.2 }, 0.42, false, 0.05), 0.2);
});

test("the server maps the engine, the kit and the kit's mounts, and never reads outside them nor an env file", () => {
  const root = temp(), kitDir = path.join(root, "brand", "video-kit"), ui = temp();
  write(path.join(kitDir, "kit.json"), {}); write(path.join(ui, "logo.svg"), "<svg/>"); write(path.join(root, "secret.env"), "K=1");
  write(path.join(root, "films", "a", "film.json"), {});
  const mounts = mountTable({ dir: kitDir, mounts: { "/ui/": path.relative(kitDir, ui) } }), player = path.join(ENGINE, "player.html");
  assert.equal(resolveRequest(root, mounts, player, "/__engine/player.mjs"), path.join(ENGINE, "player.mjs"));
  assert.equal(resolveRequest(root, mounts, player, "/__kit/kit.json"), path.join(kitDir, "kit.json"));
  assert.equal(resolveRequest(root, mounts, player, "/ui/logo.svg"), path.join(ui, "logo.svg"));
  assert.equal(resolveRequest(root, mounts, player, "/films/a/film.html"), player);
  assert.equal(resolveRequest(root, mounts, player, "/films/film.html"), null);
  assert.equal(resolveRequest(root, mounts, player, "/secret.env"), null);
  assert.equal(resolveRequest(root, mounts, player, "/../../etc/passwd"), null);
  assert.equal(resolveRequest(root, mounts, player, "/__kit/../../../etc/passwd"), null);
});

test("no encoding, mount prefix or symlink reaches outside the root and the mounts, nor a secret by another name", () => {
  const outside = temp(), root = temp(), kitDir = path.join(root, "video-kit"), ui = temp();
  write(path.join(outside, "secret.txt"), "s"); write(path.join(root, "a.txt"), "a"); write(path.join(ui, "logo.svg"), "<svg/>");
  write(path.join(root, ".env.local"), "K=1"); write(path.join(kitDir, "kit.json"), {});
  symlinkSync(path.join(outside, "secret.txt"), path.join(root, "leak.txt"));
  symlinkSync(outside, path.join(ui, "out"));
  symlinkSync(path.join(root, ".env.local"), path.join(root, "harmless.txt"));
  const mounts = mountTable({ dir: kitDir, mounts: { "/ui/": ui } }), player = path.join(ENGINE, "player.html");
  const serve = url => resolveRequest(root, mounts, player, url);
  const escape = path.relative(root, path.join(outside, "secret.txt"));
  for (const url of [`/${escape}`, `/%2e%2e/%2e%2e${outside}/secret.txt`, `/..%2f..%2f${escape}`, `/ui/..%2f..%2f${escape}`, `/__kit/%2e%2e%2f${escape}`,
    "/leak.txt", "/ui/out/secret.txt", "/.env.local", "/harmless.txt", "/%2eenv.local", "/a.txt%00.svg", "/%E0%A4%A"]) assert.equal(serve(url), null, url);
  assert.equal(serve("/a.txt"), path.join(root, "a.txt"));
  assert.equal(serve("/ui/logo.svg"), path.join(ui, "logo.svg"));
});

test("scenes imported in Node reach the engine and the kit by the paths the page uses", async () => {
  const kitDir = path.join(temp(), "video-kit");
  write(path.join(kitDir, "scenes", "index.mjs"), `import { spoken } from "/__engine/film-core.mjs"; export const SCENES = { said: spoken("[a]|b") };`);
  registerMounts(mountTable({ dir: kitDir }));
  assert.equal((await import("/__kit/scenes/index.mjs")).SCENES.said, "a b");
});

test("a film answers to the nearest kit above it, and its server root is the flag, then the kit's, then the repository", () => {
  const top = temp(), inner = path.join(top, "a", "video-kit"), film = path.join(top, "a", "b", "film");
  write(path.join(top, "video-kit", "kit.json"), { root: ".." }); write(path.join(inner, "kit.json"), {}); mkdirSync(film, { recursive: true });
  const nearest = findKit(film);
  assert.equal(nearest.dir, inner);
  assert.equal(serverRoot(nearest, film, "/srv"), "/srv");
  assert.equal(serverRoot(findKit(path.join(top, "c")), film), top);
  assert.throws(() => serverRoot(nearest, film), /pass --root/);
  execFileSync("git", ["init", "-q", top]);
  assert.equal(serverRoot(nearest, film), execFileSync("git", ["-C", top, "rev-parse", "--show-toplevel"], { encoding: "utf8" }).trim());
  assert.throws(() => findKit(os.tmpdir()), /no video-kit/);
});

test("a narration key is what it always was, so voices generated before still reach their film", () => {
  assert.equal(voiceKey({ model: "m", voice: "v", instructions: "i" }, "oi"), "40e1258ab1aa9a83f9a7b19c");
  assert.notEqual(voiceKey({ model: "m", voice: "v", instructions: "i" }, "oi", 2), voiceKey({ model: "m", voice: "v", instructions: "i" }, "oi"));
});

test("a scaffolded film versions its narration and nothing else it renders", () => {
  const repo = temp(), kitDir = path.join(repo, "video-kit"), film = path.join(repo, "films", "new");
  write(path.join(kitDir, "template.json"), { title: "", lines: [] });
  execFileSync("git", ["init", "-q", repo]);
  scaffold(film, "New", { dir: kitDir, template: "template.json" });
  assert.equal(JSON.parse(readFileSync(path.join(film, "film.json"), "utf8")).title, "New");
  for (const f of ["voice/a.wav", "voice/a.spans.json"]) write(path.join(film, f), "");
  for (const f of ["mix.wav", "cues.json", "out/a.mp4"]) write(path.join(film, f), "");
  const ignored = f => spawnSync("git", ["-C", film, "check-ignore", "-q", f]).status === 0;
  assert.deepEqual(["voice/a.wav", "voice/a.spans.json", "mix.wav", "cues.json", "out/a.mp4"].map(ignored), [false, false, true, true, true]);
});

function voiceFilm(film, kitExtra = {}) {
  const dir = temp(), home = temp(), kitDir = path.join(dir, "video-kit");
  write(path.join(kitDir, "kit.json"), { ...kit, dictionary: "dictionary.json", ...kitExtra });
  write(path.join(kitDir, "dictionary.json"), dictionary);
  write(path.join(dir, "film", "film.json"), film);
  const step = (env = {}) => spawnSync(process.execPath, [path.join(ENGINE, "voice.mjs")], { cwd: path.join(dir, "film"), encoding: "utf8", env: { ...process.env, HOME: home, FILM_KIT: kitDir, FILM_NO_NEW_VOICE: "1", GEMINI_API_KEY: "", OPENAI_API_KEY: "", ...env } });
  const measured = () => JSON.parse(readFileSync(path.join(dir, "film", ".build", "voice.json"), "utf8")).lines;
  return { film: path.join(dir, "film"), kitDir, home, step, measured };
}

test("a pinned narration plays as pinned, whatever the kit's voices say, and a missing pin names the file", () => {
  const pin = { file: "voice/n.wav", spans: "voice/n.spans.json" };
  const v = voiceFilm({ narration: pin, lines: [{ id: "a", text: "Oi." }, { id: "b", text: "Tchau." }] });
  assert.match(v.step().stderr, /narration voice\/n\.wav is missing/);
  tone(path.join(v.film, pin.file), 2);
  write(path.join(v.film, pin.spans), { lines: { a: { offset: 0, dur: 1, speech: [0, 0.9], pauses: [] }, b: { offset: 1, dur: 1, speech: [0.1, 0.8], pauses: [] } } });
  assert.equal(v.step().status, 0);
  assert.deepEqual(v.measured().b, { say: "Tchau.", take: null, voice: "whole-voice", model: "gemini-x", instructions: "calm", tempo: 1, pin: pin.spans, file: pin.file, offset: 1, dur: 1, speech: [0.1, 0.8], pauses: [] });
});

test("a narration read whole is located once, kept in the film, and a line its spans do not hold fails naming it", () => {
  const v = voiceFilm({ lines: [{ id: "a", text: "Oi." }, { id: "b", text: "Tchau." }] });
  const key = voiceKey({ ...kit.voices["whole-voice"], voice: "whole-voice" }, "Oi. <long pause> Tchau.");
  const cached = path.join(v.home, ".cache", "video-production", "voice", `${key}.wav`);
  mkdirSync(path.dirname(cached), { recursive: true });
  execFileSync("ffmpeg", ["-v", "error", "-y", "-f", "lavfi", "-i", "sine=frequency=300:duration=1.9", "-f", "lavfi", "-i", "anullsrc=r=24000:cl=mono:d=0.6",
    "-f", "lavfi", "-i", "sine=frequency=300:duration=3", "-filter_complex", "[0]aresample=24000[a];[2]aresample=24000[c];[a][1][c]concat=n=3:v=0:a=1", "-ar", "24000", "-ac", "1", cached]);
  // The words the transcription would hear, each running a little into the silence around it.
  const python = path.join(v.home, ".cache", "video-production", "venv", "bin", "python");
  write(python, `#!/bin/sh\necho '[{"start": 0.1, "end": 2.0}, {"start": 2.45, "end": 5.4}]'\n`); chmodSync(python, 0o755);
  assert.equal(v.step().status, 0);
  const { a, b } = v.measured(), near = (x, y) => assert.ok(Math.abs(x - y) < 0.03, `${x} vs ${y}`);
  near(a.speech[1], 1.9); near(b.offset, 2.2); near(b.speech[0], 0.3); near(b.offset + b.dur, 5.5);
  assert.equal(b.file, path.join("voice", `${key}.wav`));
  const spans = path.join(v.film, "voice", `${key}.spans.json`);
  write(spans, { lines: { a: JSON.parse(readFileSync(spans, "utf8")).lines.a } });
  assert.match(v.step().stderr, new RegExp(`voice/${key}\\.spans\\.json has no line b`));
});

test("an unpinned narration is looked up in the film, then in the cache, and with spending off a missing one names its file", () => {
  const v = voiceFilm({ voice: { voice: "line-voice" }, lines: [{ id: "a", text: "Oi, Acme." }] });
  const key = voiceKey({ ...kit.voices["line-voice"], voice: "line-voice" }, "Oi, Ácme.", 2);
  assert.match(v.step().stderr, new RegExp(`no narration voice/${key}\\.wav`));
  tone(path.join(v.home, ".cache", "video-production", "voice", `${key}.wav`), 1.5);
  assert.equal(v.step().status, 0);
  assert.equal(v.measured().a.file, path.join("voice", `${key}.wav`));
  assert.deepEqual(JSON.parse(readFileSync(path.join(v.film, "voice", `${key}.spans.json`), "utf8")).speech, v.measured().a.speech);
});

test("a paced clip already in the cache is used without its raw take", () => {
  const v = voiceFilm({ voice: { voice: "line-voice", tempo: 1.05 }, lines: [{ id: "a", text: "Tchau." }] });
  const name = `${voiceKey({ ...kit.voices["line-voice"], voice: "line-voice" }, "Tchau.")}-t1.05`;
  tone(path.join(v.home, ".cache", "video-production", "voice", `${name}.wav`), 1);
  assert.equal(v.step().status, 0);
  assert.equal(v.measured().a.file, path.join("voice", `${name}.wav`));
});

test("a measured line is current only while it was read from the same pin, or from the same voice settings", () => {
  const voice = { voice: "v", model: "m", instructions: "i", tempo: 1 }, line = { say: "Oi.", take: 2 };
  const unpinned = { say: "Oi.", take: 2, voice: "v", model: "m", instructions: "i", tempo: 1, pin: null };
  assert.equal(page.measurementHolds(unpinned, line, voice), true);
  assert.equal(page.measurementHolds({ ...unpinned, take: 3 }, line, voice), false);
  assert.equal(page.measurementHolds({ ...unpinned, pin: "voice/a.spans.json" }, line, voice), false);
  assert.equal(page.measurementHolds({ ...unpinned, pin: "voice/a.spans.json", voice: "other" }, line, { ...voice, pin: "voice/a.spans.json" }), true);
  assert.equal(page.measurementHolds(unpinned, line, { ...voice, pin: "voice/a.spans.json" }), false);
});

test("a folder with both a page and a film.json is refused, since the server would play the page", () => {
  const dir = temp(), kitDir = path.join(dir, "video-kit"), film = path.join(dir, "film");
  write(path.join(kitDir, "kit.json"), { ...kit, dictionary: "dictionary.json", scenes: "scenes/index.mjs", root: ".." });
  write(path.join(kitDir, "dictionary.json"), dictionary);
  write(path.join(kitDir, "scenes", "index.mjs"), "export const SCENES = { card: { name: 'card' } };");
  write(path.join(film, "film.json"), { lines: [{ id: "a", text: "Oi.", scene: "card" }] });
  write(path.join(film, "film.html"), "<!doctype html>");
  const r = spawnSync(process.execPath, [path.join(ENGINE, "film.mjs"), "run", film], { encoding: "utf8" });
  assert.equal(r.status, 1);
  assert.match(r.stderr, /film\.html beside its film\.json/);
});

test("a provider key is asked by its standard name, and the kit's env file may keep it under another", () => {
  const v = voiceFilm({ voice: { voice: "line-voice" }, lines: [{ id: "a", text: "Oi." }] }, { env: "keys.env", keys: { OPENAI_API_KEY: "TEAM_OPENAI" } });
  write(path.join(v.kitDir, "keys.env"), "OTHER=1\n");
  assert.match(v.step({ FILM_NO_NEW_VOICE: "" }).stderr, /no key: set OPENAI_API_KEY or TEAM_OPENAI in .*keys\.env/);
  const plain = voiceFilm({ voice: { voice: "line-voice" }, lines: [{ id: "a", text: "Oi." }] }, { env: "keys.env" });
  write(path.join(plain.kitDir, "keys.env"), "AI_CLI_OPENAI_API_KEY=x\n");
  assert.match(plain.step({ FILM_NO_NEW_VOICE: "" }).stderr, /no key: set OPENAI_API_KEY or OPENAI_API_KEY in/);
});

test("a caption held past its line shows over the next scene, unless that scene turns captions off", () => {
  globalThis.captionAt = page.captionAt;
  const lines = { a: { id: "a", chunks: [{ text: "Pra você.", a: 1, b: 2, onScreen: false }] }, b: { id: "b", chunks: [{ text: "Oferta", a: 2.5, b: 4, onScreen: true }] } };
  const ctx = (id, a, b, caption) => ({ a, b, caption, params: {}, parts: [{ line: lines[id] }] });
  const shown = (t, ctxs) => { const box = { style: {} }, wrap = { style: {} }; drawCaption(t, lines, lines.a.chunks[0], box, wrap, ctxs); return wrap.style.opacity === 1 ? box.textContent : null; };
  assert.equal(shown(2.1, [ctx("a", 0, 2.05), ctx("b", 2.05, 5)]), "Pra você.");
  assert.equal(shown(2.1, [ctx("a", 0, 2.05), ctx("b", 2.05, 5, false)]), null);
  assert.equal(shown(1.5, [ctx("a", 0, 2.05, false), ctx("b", 2.05, 5)]), null);
  assert.equal(shown(1.5, [ctx("a", 0, 2.05), ctx("b", 2.05, 5, false)]), "Pra você.");
  delete globalThis.captionAt;
});

const good = {
  cues: { duration: 20, voice: [{ id: "a", speech: [0.1, 2] }], scenes: [["a", 0, 20]] },
  facts: { target: 20, lines: [{ id: "a", chunks: [{ text: "Oi.", b: 1 }] }], endCard: { text: "sete dias", readyAt: 15, end: 20 } },
  compare: { total: 10, missed: 0, ratio: 0, coverage: [{ id: "a", ratio: 1 }], wrong: [], nameHunks: [], names: [] },
  lufs: -14, truePeak: -1.9, fileDuration: 20, still: { seconds: 1, at: 0 }, frame0Std: 40, cuts: [], spans: [],
};

test("figures read the same in the script and in the transcript", () => {
  assert.deepEqual(tokens("Cinquenta por cento e sete dias."), tokens("50% e 7 dias"));
  assert.deepEqual(tokens("[Acme.]|[Sete dias grátis.]"), ["acme", "sete", "dias", "gratis"]);
});

test("the word diff lists what disagrees and nothing else", () => {
  const { hunks } = diffWords(["a", "b", "c", "d"], ["a", "x", "c", "d", "e"]);
  assert.deepEqual(hunks.map(h => [h.script, h.heard]), [[["b"], ["x"]], [[], ["e"]]]);
});

test("a proper name goes to the ear, and a missing line fails the transcript", () => {
  const names = { entries: [{ word: "Acme", heard_as: ["acne"] }] };
  const lines = [{ id: "a", text: "Na Acme, você clica." }, { id: "b", text: "O estoque baixa nos outros canais todos." }];
  const heard = compareTranscript(lines, "Na acne, você clica.", names);
  assert.equal(heard.nameHunks.length, 1);
  assert.deepEqual(heard.coverage.map(c => c.ratio >= 0.5), [true, false]);
  assert.equal(compareTranscript(lines, "Na acne, você clica. O estoque baixa nos outros canais todos.", names).missed, 0);
});

test("the transcript file is read back to its spoken text", () => {
  assert.equal(parseTranscript("# t\n\n`00:00` Quanto você vendeu?  \n`00:03` Não o que entrou.  \n"), "Quanto você vendeu? Não o que entrou.");
});

test("the hook is the first spoken sentence", () => {
  assert.equal(hookEnd([{ chunks: [{ text: "Etiqueta é isso.", b: 1.9 }, { text: "Baixa um txt.", b: 3.2 }] }]), 1.9);
  assert.equal(hookEnd([{ chunks: [{ text: "Vendeu", b: 0.8 }, { text: "com prejuízo", b: 1.4 }] }]), 1.4);
});

test("speech that passes its scene's end is reported", () => {
  const scenes = [["a", 0, 3], ["b", 3, 6]];
  assert.deepEqual(speechCuts([{ id: "a", speech: [0.1, 2.9] }], scenes), []);
  assert.equal(speechCuts([{ id: "a", speech: [0.1, 3.4] }], scenes)[0].id, "a");
});

test("a caption lasts until the next chunk, and a flash is reported", () => {
  const spans = captionSpans([{ id: "a", chunks: [{ text: "um dois", a: 1, b: 1.2, onScreen: false }, { text: "tres", a: 1.2, b: 2, onScreen: false }] }], 0.3);
  assert.deepEqual(spans.map(s => +s.seconds.toFixed(2)), [0.2, 1.1]);
  assert.equal(shortCaptions(spans).length, 1);
});

const frame = v => new Uint8Array(100).fill(v);

test("a freeze is the longest still stretch, and slow drift never adds up to one", () => {
  const frozen = [frame(0), frame(40), ...Array(30).fill(frame(80)), frame(120)];
  const drift = Array.from({ length: 60 }, (_, i) => frame(i * 0.5));
  assert.equal(longestStill(frozen, 10, 1).seconds, 3);
  assert.ok(longestStill(drift, 10, 1).seconds < 0.5);
});

test("an empty frame 0 is seen", () => {
  assert.equal(lumaStd(frame(0)), 0);
  assert.ok(lumaStd(Uint8Array.from({ length: 100 }, (_, i) => (i % 2 ? 200 : 0))) > 50);
});

test("every limit fails when crossed", () => {
  assert.ok(runChecks(good).every(c => c.ok));
  const bad = {
    duration: { fileDuration: 19 }, loudness: { lufs: -12 }, peak: { truePeak: -0.5 }, "first-speech": { cues: { ...good.cues, voice: [{ id: "a", speech: [0.9, 2] }] } },
    hook: { facts: { ...good.facts, lines: [{ id: "a", chunks: [{ text: "Oi.", b: 4 }] }] } }, "speech-cut": { cuts: [{ id: "a", over: 1, scene: "a" }] },
    transcript: { compare: { ...good.compare, coverage: [{ id: "a", ratio: 0.2 }] } }, captions: { spans: [{ onScreen: false, seconds: 0.1, words: 2, text: "x" }] },
    still: { still: { seconds: 4, at: 2 } }, frame0: { frame0Std: 1 }, "end-card": { facts: { ...good.facts, endCard: { text: "um dois tres quatro cinco seis", readyAt: 19, end: 20 } } },
  };
  for (const [id, change] of Object.entries(bad)) {
    const failed = runChecks({ ...good, ...change }).filter(c => !c.ok).map(c => c.id);
    assert.deepEqual(failed, [id]);
  }
});

test("layout keeps the engine's rule: scenes end on the beat, after the line", () => {
  const lines = { a: { speech: [0, 2.3], text: "x", pauses: [] } };
  assert.equal(page.layoutScenes([{ id: "a", line: "a", lead: 0.05, tail: 0.1 }], lines, { grid: 0.5 }).duration, 2.5);
});

// The page modules only run inside the browser, so a parse error there surfaces as a render failure far from its cause.
test("the modules the film page loads parse", () => {
  for (const file of ["player.mjs", "timeline.js", "film-core.mjs", "scenes/captions.mjs", "scenes/index.mjs"]) {
    const r = spawnSync(process.execPath, ["--check", path.join(ENGINE, file)], { encoding: "utf8" });
    assert.equal(r.status, 0, `${file}: ${r.stderr}`);
  }
});

// Measured on a real narration, where the nearest pause to one edge lay past the next one.
test("a caption chunk never ends before it starts, whatever pause lies nearby", () => {
  const chunks = page.timeChunks({ text: "Longe da mesa,|ele mostra a peça,|a cor|e quantas caixas.", at: 0, speech: [0.13, 3.396], pauses: [[0.161, 0.368], [1.232, 1.391], [2.47, 2.55]] });
  for (const c of chunks) assert.ok(c.b > c.a, `${c.text}: ${c.a}..${c.b}`);
});
