// A dark ground tinted by the accent reads brown, and a light layer in front of an object reads as glass: surfaces stay saturation 0, lights stay behind, objects stay opaque.
import { parseArgs, launch, open, scrollToElement, report } from "./lib.mjs";

const { url, viewports, cfg } = parseArgs({ viewports: "1440x900" });
const c = { surfaces: [], maxSaturation: 2, lights: [], opaque: [], scrollTo: null, waitMs: 0, ...(cfg.neutral || {}) };
const browser = await launch();
const results = [];
for (const vp of viewports) {
  const { ctx, page } = await open(browser, url, vp, { settleMs: 800 });
  if (c.scrollTo) await scrollToElement(page, c.scrollTo, "center", 300 + c.waitMs);
  const r = await page.evaluate(({ surfaces, lights, opaque }) => {
    const sat = col => { const m = col.match(/[\d.]+/g).map(Number); if (m.length > 3 && m[3] === 0) return null; const [r, g, b] = m.slice(0, 3).map(x => x / 255), mx = Math.max(r, g, b), mn = Math.min(r, g, b), l = (mx + mn) / 2, d = mx - mn; return d === 0 ? 0 : Math.round(d / (1 - Math.abs(2 * l - 1)) * 100); };
    const surf = surfaces.map(s => { const e = document.querySelector(s); return [s, e ? sat(getComputedStyle(e).backgroundColor) : "missing"]; });
    const z = s => { const e = document.querySelector(s); return e ? parseInt(getComputedStyle(e).zIndex) || 0 : null; };
    const light = lights.map(({ light, objects }) => [light, objects, z(light), z(objects)]);
    const op = opaque.map(s => { const e = document.querySelector(s); return [s, e ? +getComputedStyle(e).opacity : "missing"]; });
    return { surf, light, op };
  }, c);
  for (const [s, v] of r.surf) results.push({ label: `${vp.join("x")} surface ${s}`, pass: v === null || (typeof v === "number" && v <= c.maxSaturation), detail: v === null ? "transparent" : `saturation ${v}%` });
  for (const [l, o, zl, zo] of r.light) results.push({ label: `${vp.join("x")} light ${l} behind ${o}`, pass: zl !== null && zo !== null && zl < zo, detail: `z ${zl} < ${zo}` });
  for (const [s, v] of r.op) results.push({ label: `${vp.join("x")} opaque ${s}`, pass: v === 1, detail: `opacity ${v}` });
  await ctx.close();
}
await browser.close();
report("neutral-and-light", results);
