import { spawn, spawnSync } from "node:child_process";
import { readFileSync, writeFileSync } from "node:fs";
import { THRESHOLDS as T } from "./qa-thresholds.mjs";
import { kitFile, loadKit } from "./kit.mjs";
import { CAPTION_HOLD } from "./scenes/captions.mjs";
import { captionSpans, compareTranscript, longestStill, lumaStd, parseTranscript, reportMarkdown, runChecks, speechCuts } from "./qa-checks.mjs";

const [file] = process.argv.slice(2);
const SMALL = { width: 108, height: 192 };

function greyFrames(path) {
  return new Promise((resolve, reject) => {
    const ff = spawn("ffmpeg", ["-v", "error", "-i", path, "-vf", `fps=${T.stillSampleFps},scale=${SMALL.width}:${SMALL.height},format=gray`, "-f", "rawvideo", "-"], { stdio: ["ignore", "pipe", "inherit"] });
    const parts = [];
    ff.stdout.on("data", d => parts.push(d));
    ff.on("error", reject);
    ff.on("close", () => {
      const all = Buffer.concat(parts), size = SMALL.width * SMALL.height, out = [];
      for (let i = 0; i + size <= all.length; i += size) out.push(all.subarray(i, i + size));
      resolve(out);
    });
  });
}

function loudness(path) {
  const log = spawnSync("ffmpeg", ["-hide_banner", "-nostats", "-i", path, "-af", "ebur128=peak=true", "-f", "null", "-"], { encoding: "utf8", maxBuffer: 1 << 28 }).stderr;
  const tail = log.slice(log.lastIndexOf("Summary:"));
  return { lufs: +tail.match(/I:\s*(-?[\d.]+) LUFS/)[1], truePeak: +tail.match(/Peak:\s*(-?[\d.]+) dBFS/)[1] };
}

const duration = path => +spawnSync("ffprobe", ["-v", "error", "-show_entries", "format=duration", "-of", "csv=p=0", path], { encoding: "utf8" }).stdout.trim();
const readJson = p => JSON.parse(readFileSync(p, "utf8"));

const cues = readJson("cues.json"), facts = readJson(".build/facts.json");
const kit = loadKit(process.env.FILM_KIT);
const dictionary = readJson(kitFile(kit, kit.dictionary));
const frames = await greyFrames(file);
const compare = compareTranscript(facts.lines, parseTranscript(readFileSync(".build/transcript.md", "utf8")), dictionary);
const checks = runChecks({
  cues, facts, compare, ...loudness(file), fileDuration: duration(file),
  still: longestStill(frames, T.stillSampleFps), frame0Std: lumaStd(frames[0]),
  cuts: speechCuts(cues.voice, cues.scenes), spans: captionSpans(facts.lines, CAPTION_HOLD),
});
const report = reportMarkdown(file, checks, compare);
writeFileSync(".build/qa.md", report);
writeFileSync(".build/qa.json", JSON.stringify({ file, checks, compare }, null, 1));
console.log(report);
if (checks.some(c => !c.ok)) process.exitCode = 1;
