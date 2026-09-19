// A scroll-driven effect that is smooth on desktop can be an instant swap on a phone: its progress must move in small steps over enough scroll, and read the same going down and up.
// Progress is a number the page writes, read from `var` on `selector` (a custom property or a data attribute).
import { parseArgs, launch, open, scrollTo, report } from "./lib.mjs";

const { url, viewports, cfg } = parseArgs({ viewports: "390x844,768x1024,1440x900" });
const c = { selector: null, var: "--p", step: 20, maxJump: 0.1, minRunway: 300, ...(cfg.fold || {}) };
if (!c.selector) { console.error("fold.selector is missing in the config"); process.exit(2); }
const browser = await launch();
const results = [];
for (const vp of viewports) {
  const { ctx, page } = await open(browser, url, vp, { settleMs: 800 });
  const read = async y => { await scrollTo(page, y, 30); return page.evaluate(({ sel, v }) => { const e = document.querySelector(sel); if (!e) return NaN; const raw = v.startsWith("--") ? getComputedStyle(e).getPropertyValue(v) || e.style.getPropertyValue(v) : e.getAttribute(v); return parseFloat(raw); }, { sel: c.selector, v: c.var }); };
  const H = await page.evaluate(() => document.documentElement.scrollHeight);
  let start = null, end = null;
  for (let y = 0; y < H && end === null; y += start === null ? c.step : 4) { const p = await read(y); if (start === null && p > 0) { start = Math.max(0, y - c.step); y = start - 4; continue; } if (start !== null && p >= 0.999) end = y; }
  if (start === null || end === null) { results.push({ label: vp.join("x"), pass: false, detail: "progress never ran from 0 to 1" }); await ctx.close(); continue; }
  const ys = []; for (let y = start; y <= end + c.step; y += c.step) ys.push(y);
  const down = []; for (const y of ys) down.push(await read(y));
  const up = []; for (const y of [...ys].reverse()) up.unshift(await read(y));
  let jump = 0; for (let i = 1; i < down.length; i++) jump = Math.max(jump, Math.abs(down[i] - down[i - 1]));
  const same = down.every((v, i) => Math.abs(v - up[i]) < 0.005), runway = end - start;
  results.push({ label: vp.join("x"), pass: jump <= c.maxJump && same && runway >= c.minRunway, detail: `runway ${runway}px, max jump ${jump.toFixed(3)} per ${c.step}px, down = up ${same}` });
  await ctx.close();
}
await browser.close();
report("fold-sampler", results);
