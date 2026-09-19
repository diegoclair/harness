// A light section reads against the page's inherited dark ink, or the reverse: a screenshot glance and type-check both miss a barely-visible font/background pair.
// Effective background alpha-composites every translucent ancestor layer up to the nearest opaque one; WCAG ratio checked against 4.5, or 3 for >=24px or >=18.66px bold text.
import { parseArgs, launch, open, scrollTo, report } from "./lib.mjs";

const { url, viewports, cfg } = parseArgs({ viewports: "390x844,1024x768,1440x900,1920x1080" });
const c = { scope: "body", ignore: [], ignoreWhy: {}, ...(cfg.contrast || {}) };
const browser = await launch();
const results = [];

for (const vp of viewports) {
  const { ctx, page } = await open(browser, url, vp);
  const H = await page.evaluate(() => document.documentElement.scrollHeight);
  const found = new Map();
  for (let y = 0; y <= H; y += Math.round(vp[1] * 0.7)) {
    await scrollTo(page, y, 60);
    const nodes = await page.evaluate(({ scope, ignore }) => {
      const parse = (str) => {
        const m = str.match(/[\d.]+/g)?.map(Number);
        if (!m) return null;
        const [r, g, b, a = 1] = m;
        return { r, g, b, a };
      };
      const lin = (v) => { const s = v / 255; return s <= 0.03928 ? s / 12.92 : Math.pow((s + 0.055) / 1.055, 2.4); };
      const luminance = ({ r, g, b }) => 0.2126 * lin(r) + 0.7152 * lin(g) + 0.0722 * lin(b);
      const ratio = (c1, c2) => { const L1 = luminance(c1), L2 = luminance(c2); const hi = Math.max(L1, L2), lo = Math.min(L1, L2); return (hi + 0.05) / (lo + 0.05); };
      const effectiveBg = (el) => {
        const layers = [];
        for (let a = el; a; a = a.parentElement) {
          const cs = getComputedStyle(a);
          const bg = parse(cs.backgroundColor);
          if (bg && bg.a > 0) layers.unshift(bg);
          if (bg && bg.a > 0.98) break;
        }
        let out = { r: 255, g: 255, b: 255 };
        for (const l of layers) out = { r: l.r * l.a + out.r * (1 - l.a), g: l.g * l.a + out.g * (1 - l.a), b: l.b * l.a + out.b * (1 - l.a) };
        return out;
      };
      const out = [];
      for (const el of document.querySelectorAll(scope + " *")) {
        if (ignore.length && el.closest(ignore.join(","))) continue;
        const hasOwnText = [...el.childNodes].some((n) => n.nodeType === 3 && n.textContent.trim());
        if (!hasOwnText) continue;
        const cs = getComputedStyle(el);
        if (cs.visibility === "hidden" || cs.display === "none" || +cs.opacity === 0) continue;
        const rect = el.getBoundingClientRect();
        if (!rect.width || !rect.height || rect.bottom < 0 || rect.top > innerHeight) continue;
        const fg = parse(cs.color);
        if (!fg) continue;
        const bg = effectiveBg(el);
        const r = ratio(fg, bg);
        const fontSize = parseFloat(cs.fontSize);
        const bold = parseInt(cs.fontWeight) >= 700;
        const threshold = fontSize >= 24 || (fontSize >= 18.66 && bold) ? 3 : 4.5;
        const text = el.textContent.trim().slice(0, 30);
        const key = el.tagName + "." + (typeof el.className === "string" ? el.className.slice(0, 40) : "") + ":" + text;
        out.push({ key, pass: r >= threshold, ratio: Math.round(r * 100) / 100, threshold, text, fg: cs.color, bg: `rgb(${Math.round(bg.r)}, ${Math.round(bg.g)}, ${Math.round(bg.b)})` });
      }
      return out;
    }, { scope: c.scope, ignore: c.ignore });
    for (const n of nodes) if (!found.has(n.key) || (found.get(n.key).pass && !n.pass)) found.set(n.key, n);
  }
  const all = [...found.values()];
  const fails = all.filter((n) => !n.pass);
  results.push({
    label: `${vp.join("x")}`,
    pass: fails.length === 0,
    detail: `${all.length} measured, ${fails.length} failed` + (fails.length ? " · " + fails.slice(0, 5).map((f) => `"${f.text}" ${f.ratio}<${f.threshold} fg=${f.fg} bg=${f.bg}`).join(" | ") : ""),
  });
  await ctx.close();
}
await browser.close();
report("contrast", results);
