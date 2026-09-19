// A void of scroll reads as a broken page: no stretch of a section may show nothing but ground.
// A band at a section's own edge is allowed up to that edge's padding; `allow` raises the limit for a section with a written reason.
import { parseArgs, launch, open, scrollTo, report } from "./lib.mjs";

const { url, viewports, cfg } = parseArgs({ viewports: "390x844,1440x900,1440x1211,1920x1080" });
const c = { sections: "main > section", surfaces: [], limit: 64, step: 100, allow: {}, ...(cfg.emptyBands || {}) };
const browser = await launch();
const results = [];
for (const vp of viewports) {
  const { ctx, page } = await open(browser, url, vp, { settleMs: c.settleMs ?? 2500 });
  const H = await page.evaluate(() => document.documentElement.scrollHeight);
  const worst = {};
  for (let y = 0; y <= H - vp[1]; y += c.step) {
    await scrollTo(page, y, 25);
    const r = await page.evaluate(({ sections, surfaces }) => {
      const vh = innerHeight, surf = surfaces.length ? new RegExp("\\b(" + surfaces.join("|") + ")\\b") : null, items = [];
      for (const e of document.querySelectorAll("main *")) {
        const cs = getComputedStyle(e); if (cs.visibility === "hidden" || cs.display === "none" || +cs.opacity === 0) continue;
        const text = [...e.childNodes].some(n => n.nodeType === 3 && n.textContent.trim());
        const ui = (surf && surf.test(typeof e.className === "string" ? e.className : "")) || ["svg", "INPUT", "IMG"].includes(e.tagName);
        if (!text && !ui) continue;
        const q = e.getBoundingClientRect(); if (q.bottom <= 0 || q.top >= vh || !q.height) continue;
        items.push([q.top, q.bottom, e]);
      }
      const out = [];
      for (const s of document.querySelectorAll(sections)) {
        const q = s.getBoundingClientRect(), top = Math.max(0, q.top), bot = Math.min(vh, q.bottom); if (bot - top < 2) continue;
        const cs = getComputedStyle(s), padT = parseFloat(cs.paddingTop), padB = parseFloat(cs.paddingBottom);
        const iv = items.filter(([a, b, e]) => s.contains(e) && b > top && a < bot).map(([a, b]) => [Math.max(a, top), Math.min(b, bot)]).sort((x, y) => x[0] - y[0]);
        let cur = top, gap = 0, first = true;
        for (const [a, b] of iv) {
          if (a > cur) { const g = a - cur, inPadding = first && a - q.top <= padT + 2; if (!inPadding) gap = Math.max(gap, g); }
          cur = Math.max(cur, b); first = false;
        }
        const tail = bot - cur, allPadding = !iv.length && (bot - q.top <= padT + 2 || q.bottom - top <= padB + 2);
        if (tail > 0 && !allPadding && (iv.length ? q.bottom - cur > padB + 2 : true)) gap = Math.max(gap, tail);
        const name = s.tagName.toLowerCase() + (s.className ? "." + s.className.trim().split(/\s+/).join(".") : "");
        out.push([name, Math.round(gap)]);
      }
      return out;
    }, { sections: c.sections, surfaces: c.surfaces });
    for (const [k, v] of r) if (!worst[k] || v > worst[k][0]) worst[k] = [v, y];
  }
  const over = Object.entries(worst).filter(([k, [v]]) => v > (c.allow[k] ?? c.limit));
  results.push({ label: vp.join("x"), pass: over.length === 0, detail: over.length ? over.map(([k, [v, y]]) => `${k} ${v}px at y=${y}`).join("; ") : `max ${Math.max(0, ...Object.values(worst).map(x => x[0]))}px within limits` });
  await ctx.close();
}
await browser.close();
report("empty-bands", results);
