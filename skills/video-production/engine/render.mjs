import { spawn } from "node:child_process";
import { writeFileSync } from "node:fs";
import { openFilm, frameSize } from "./lib.mjs";

const [fmt = "h", out = "picture.mp4", from = "0", to = ""] = process.argv.slice(2);
const FPS = 30;
const { page, close } = await openFilm({ fmt, viewport: frameSize(fmt), args: ["--force-color-profile=srgb", "--font-render-hinting=none"] });
// A soundtrack cut from estimated speech would drift off the picture, so an unmeasured line stops the render.
const unmeasured = await page.evaluate(() => window.voiceUnmeasured ?? []);
if (unmeasured.length) { await close(); throw new Error(`run the voice step first: unmeasured ${unmeasured.join(", ")}`); }
const cues = await page.evaluate(() => window.CUES);
writeFileSync("cues.json", JSON.stringify(cues));
const t0 = +from, t1 = to ? +to : cues.duration;
const ff = spawn("ffmpeg", ["-y", "-loglevel", "error", "-f", "image2pipe", "-framerate", String(FPS), "-c:v", "mjpeg", "-i", "-",
  "-c:v", "libx264", "-preset", "slow", "-crf", "16", "-pix_fmt", "yuv420p", "-movflags", "+faststart", out], { stdio: ["pipe", "inherit", "inherit"] });
// An edit list maps output time onto film time, so a short cut reuses the same scenes.
const EDL = process.env.EDL ? JSON.parse(process.env.EDL) : [[t0, t1]];
const frames = EDL.flatMap(([a, b]) => Array.from({ length: Math.round((b - a) * FPS) }, (_, k) => a + k / FPS));
const n = frames.length;
for (let i = 0; i < n; i++) {
  const t = frames[i];
  await page.evaluate(t => window.render(t), t);
  const buf = await page.screenshot({ type: "jpeg", quality: 95 });
  if (!ff.stdin.write(buf)) await new Promise(r => ff.stdin.once("drain", r));
  if (i % 150 === 0) console.log(`${fmt} ${t.toFixed(1)}s`);
}
ff.stdin.end();
await new Promise(r => ff.on("close", r));
await close();
