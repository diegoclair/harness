#!/usr/bin/env node
import { spawn } from "node:child_process";
import { existsSync, mkdirSync, readdirSync, readFileSync } from "node:fs";
import path from "node:path";
import { run as runTests } from "node:test";
import { spec as specReporter } from "node:test/reporters";
import { findKit, kitFile, loadKit, serverRoot } from "./kit.mjs";
import { ENGINE, mountTable, registerMounts } from "./mounts.mjs";
import { startServer } from "./serve.mjs";
import { applyVariant, validateFilm, resolveLine, resolveVoice, spokenDigits } from "./film-core.mjs";
import { sceneLibrary } from "./scenes/index.mjs";
import { scaffold } from "./scaffold.mjs";
import { PYTHON, setup } from "./setup.mjs";

const STEPS = ["voice", "measure", "render", "music", "mix", "mux", "transcribe", "sheet", "qa"];
// numpy's thread pools fight each other when films run side by side, and a mix changes nothing with more of them.
const SINGLE_THREAD = { OPENBLAS_NUM_THREADS: "1", OMP_NUM_THREADS: "1" };
const SHEET_TILES = { cols: 9, rows: 4, width: 216 };
const USAGE = `film setup                                fetch what the engine needs outside the skill folder, once per machine
film new <folder> [--title "..."]         scaffold a film from the kit's template
film run <folder> [--from step] [--to step] [--query k=v] [--root dir] [--no-new-voice]
film serve <folder> [--root dir]          open the film in a browser, with its audio
film scenes <folder>                      what the kit and the engine offer, their params and where their words live
film sample <folder> <line> <out folder> <voice,voice>   the same line in other approved voices, to hear side by side
film test <kit folder>                    run the kit's tests against this engine
steps: ${STEPS.join(" ")}`;

function run(cmd, args, { cwd, env = {}, quiet } = {}) {
  return new Promise((resolve, reject) => {
    const child = spawn(cmd, args, { cwd, env: { ...process.env, ...env }, stdio: quiet ? ["ignore", "ignore", "pipe"] : "inherit" });
    let err = "";
    child.stderr?.on("data", d => { err += d; });
    child.on("error", reject);
    child.on("close", code => (code ? reject(new Error(`${path.basename(cmd)} ${args[0] ?? ""} exited ${code}${err ? `\n${err.slice(-600)}` : ""}`)) : resolve()));
  });
}

const readJson = file => JSON.parse(readFileSync(file, "utf8"));

// Kit scenes may import from the engine or the kit by the same paths the page uses, so Node learns those paths first.
async function openKit(dir) {
  const kit = findKit(dir);
  registerMounts(mountTable(kit));
  return { kit, scenes: await sceneLibrary(`/__kit/${kit.scenes}`) };
}

function checkFilm(dir, kit, scenes) {
  const file = path.join(dir, "film.json");
  if (!existsSync(file)) throw new Error(`${dir} has no film.json`);
  if (existsSync(path.join(dir, "film.html"))) throw new Error(`${dir} has a film.html beside its film.json, and the server would play the page: keep one`);
  const base = readJson(file), dictionary = readJson(kitFile(kit, kit.dictionary));
  for (const name of [null, ...Object.keys(base.variants ?? {})]) {
    const film = applyVariant(base, name), label = name ? `film.json, variant ${name}` : "film.json";
    const errors = validateFilm(film, scenes);
    if (errors.length) throw new Error(`${label}:\n  ${errors.join("\n  ")}`);
    const voice = resolveVoice(kit, film.voice).voice;
    const digits = spokenDigits(film.lines.map(l => resolveLine(l, dictionary, voice)));
    if (digits.length) console.log(`warning: ${label}: figures in the spoken text of ${digits.join(", ")}: the voice reads a written number as text, spell it out in say`);
  }
}

function outName(dir, query, kit) {
  const base = path.basename(path.resolve(dir));
  return path.join("out", `${kit.output ? `${kit.output}-` : ""}${base}${query ? `-${query.replace(/\W+/g, "-")}` : ""}-9x16.mp4`);
}

function music(dir, kit) {
  if (existsSync(path.join(dir, "music.py"))) return "music.py";
  return kit.music ? kitFile(kit, kit.music) : path.join(ENGINE, "score.py");
}

const filmSteps = ({ dir, env, query, kit }) => {
  const script = name => path.join(ENGINE, name);
  const out = outName(dir, query, kit);
  return {
    voice: () => run("node", [script("voice.mjs")], { cwd: dir, env }),
    measure: async () => { await run("node", [script("cues.mjs")], { cwd: dir, env }); await run("node", [script("facts.mjs")], { cwd: dir, env }); },
    render: () => run("node", [script("render.mjs"), "v", "pic-v.mp4"], { cwd: dir, env }),
    // A film's own music.py imports the engine's instruments by name.
    music: () => run(PYTHON, [music(dir, kit), "cues.json", "score.wav"], { cwd: dir, env: { ...env, ...SINGLE_THREAD, PYTHONPATH: ENGINE }, quiet: true }),
    mix: () => run(PYTHON, [script("mix.py"), "cues.json", "score.wav", "mix.wav"], { cwd: dir, env: { ...env, ...SINGLE_THREAD } }),
    mux: () => {
      const { duration } = readJson(path.join(dir, "cues.json"));
      mkdirSync(path.join(dir, "out"), { recursive: true });
      // Music stops mid-note at the last frame, and the cut would click without this.
      const fade = `afade=t=out:st=${(duration - 0.15).toFixed(3)}:d=0.15`;
      return run("ffmpeg", ["-y", "-loglevel", "error", "-i", "pic-v.mp4", "-i", "mix.wav", "-map", "0:v", "-map", "1:a", "-af", fade, "-c:v", "copy", "-c:a", "aac", "-b:a", "320k", "-shortest", "-movflags", "+faststart", out], { cwd: dir, env });
    },
    transcribe: () => run(PYTHON, [script("transcribe.py"), out, ".build/transcript.md"], { cwd: dir, env: { ...env, OMP_NUM_THREADS: "2" }, quiet: true }),
    sheet: () => {
      const { duration } = readJson(path.join(dir, "cues.json")), { cols, rows, width } = SHEET_TILES;
      return run("ffmpeg", ["-y", "-loglevel", "error", "-i", out, "-vf", `fps=${(cols * rows) / duration},scale=${width}:-1,tile=${cols}x${rows}`, "-frames:v", "1", "-update", "1", ".build/sheet.jpg"], { cwd: dir, env });
    },
    qa: () => run("node", [script("qa.mjs"), out], { cwd: dir, env }),
  };
};

function parseArgs(argv) {
  const [verb, dir, ...rest] = argv, opts = {}, args = [];
  for (let i = 0; i < rest.length; i++) {
    if (rest[i] === "--no-new-voice") opts.noNewVoice = true;
    else if (rest[i].startsWith("--")) opts[rest[i].slice(2)] = rest[++i];
    else args.push(rest[i]);
  }
  return { verb, dir, opts, args };
}

// The kit's prepare script builds what its scenes read from the film folder, so it runs before any page opens.
async function startFilmServer(dir, opts) {
  const { kit, scenes } = await openKit(dir);
  checkFilm(dir, kit, scenes);
  const root = serverRoot(kit, dir, opts.root);
  if (path.relative(root, path.resolve(dir)).startsWith("..")) throw new Error(`${dir} is outside the server root ${root}`);
  const server = await startServer(root, mountTable(kit), path.join(ENGINE, "player.html"));
  const env = { FILM_ORIGIN: server.origin, FILM_ROOT: root, FILM_KIT: kit.dir };
  try {
    if (kit.prepare) await run("node", ["--import", path.join(ENGINE, "register-mounts.mjs"), kitFile(kit, kit.prepare)], { cwd: dir, env });
  } catch (e) {
    server.stop();
    throw e;
  }
  return { kit, root, server, env };
}

async function runFilm(dir, opts) {
  const from = STEPS.indexOf(opts.from ?? STEPS[0]), to = STEPS.indexOf(opts.to ?? STEPS[STEPS.length - 1]);
  if (from < 0 || to < 0) throw new Error(`steps are: ${STEPS.join(" ")}`);
  const { kit, server, env: base } = await startFilmServer(dir, opts);
  const stop = () => { server.stop(); process.exit(130); };
  process.once("SIGINT", stop); process.once("SIGTERM", stop);
  const env = { ...base, ...(opts.query ? { FILM_QUERY: opts.query } : {}), ...(opts.noNewVoice ? { FILM_NO_NEW_VOICE: "1" } : {}) };
  const steps = filmSteps({ dir, env, query: opts.query, kit });
  let current;
  try {
    for (const name of STEPS.slice(from, to + 1)) {
      current = name;
      const t0 = Date.now();
      await steps[name]();
      console.log(`[film] ${name} done in ${((Date.now() - t0) / 1000).toFixed(1)} s`);
    }
  } catch (e) {
    // QA already printed its failures; the other steps need a way back in.
    throw new Error(current === "qa" ? "QA failed: see .build/qa.md" : `${current} failed, resume with --from ${current}\n${e.message}`);
  } finally {
    server.stop();
  }
}

async function serveFilm(dir, opts) {
  const { root, server } = await startFilmServer(dir, opts);
  console.log(`${server.origin}/${path.relative(root, path.resolve(dir)).split(path.sep).join("/")}/film.html?fmt=v`);
  console.log("click the page to play it with the mix; ?t=12 freezes a second; Ctrl-C stops");
  process.once("SIGINT", () => { server.stop(); process.exit(0); });
}

function sampleVoices(dir, [line, out, voices], opts) {
  if (!voices) throw new Error("film sample <folder> <line> <out folder> <voice,voice>");
  const env = { FILM_KIT: findKit(dir).dir, ...(opts.query ? { FILM_QUERY: opts.query } : {}) };
  return run("node", [path.join(ENGINE, "voice.mjs"), "--sample", line, path.resolve(out), voices], { cwd: dir, env });
}

async function listScenes(dir) {
  const { scenes } = await openKit(dir);
  for (const s of Object.values(scenes)) {
    console.log(`${s.name}${s.lines > 1 ? ` (spans ${s.lines} lines)` : ""}  needs: ${(s.required ?? []).join(", ") || "nothing"}`);
    for (const w of new Set((s.wording ?? []).map(x => x.in))) console.log(`  words from ${w}`);
  }
}

// In-process, so the kit's tests import the engine and the kit by the paths the page uses.
async function testKit(dir) {
  const kit = loadKit(dir);
  registerMounts(mountTable(kit));
  const files = readdirSync(kit.dir).filter(f => f.endsWith(".test.mjs")).map(f => path.join(kit.dir, f));
  if (!files.length) throw new Error(`${kit.dir} has no *.test.mjs`);
  const stream = runTests({ files, isolation: "none" });
  stream.on("test:fail", () => { process.exitCode = 1; });
  stream.compose(specReporter).pipe(process.stdout);
}

const { verb, dir, opts, args } = parseArgs(process.argv.slice(2));
try {
  if (args.length && verb !== "sample") throw new Error(`unknown argument ${args[0]}`);
  if (verb === "setup") setup();
  else if (verb === "new" && dir) console.log(scaffold(dir, opts.title, findKit(path.dirname(path.resolve(dir)))));
  else if (verb === "run" && dir) await runFilm(dir, opts);
  else if (verb === "serve" && dir) await serveFilm(dir, opts);
  else if (verb === "scenes" && dir) await listScenes(dir);
  else if (verb === "test" && dir) await testKit(dir);
  else if (verb === "sample" && dir) await sampleVoices(dir, args, opts);
  else { console.log(USAGE); process.exitCode = 2; }
} catch (e) {
  console.error(e.message);
  process.exitCode = 1;
}
