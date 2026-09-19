// Content cut by an ancestor's overflow: the page-level scrollWidth check cannot see it, because the clipping box hides the excess.
// Reports text or images partly inside and partly outside a hidden/clip ancestor; fully hidden ones are a state, not a cut; an ellipsis is an intended cut.
import { parseArgs, launch, open, scrollTo, report } from "./lib.mjs";

const { url, viewports, cfg } = parseArgs({ viewports: "390x844,1024x768,1440x900,1920x1080" });
const c = { tolerance: 1, ignore: [], ...(cfg.clipped || {}) };
const browser = await launch();
const results = [];
// Reduced motion serves every scene in its final state, so text a timeline hides at the sampled moment is still measured.
for (const vp of viewports) for (const mode of ["js", "reduced"]) {
  const { ctx, page } = await open(browser, url, vp, { reducedMotion: mode === "reduced" ? "reduce" : "no-preference" });
  const H = await page.evaluate(() => document.documentElement.scrollHeight);
  const found = new Map();
  for (let y = 0; y <= H; y += Math.round(vp[1] * .6)) {
    await scrollTo(page, y, 160);
    const cut = await page.evaluate(({ tol, ignore }) => {
      const out = [];
      const shown = e => { for (let a = e; a && a !== document.documentElement; a = a.parentElement) { const cs = getComputedStyle(a); if (cs.display === "none" || cs.visibility === "hidden" || +cs.opacity === 0) return false; } return true; };
      const name = e => e.tagName.toLowerCase() + (typeof e.className === "string" && e.className.trim() ? "." + e.className.trim().split(/\s+/).join(".") : "");
      for (const e of document.querySelectorAll("body *")) {
        if (ignore.length && e.closest(ignore.join(","))) continue;
        if (![...e.childNodes].some(n => n.nodeType === 3 && n.textContent.trim()) && e.tagName !== "IMG") continue;
        const q = e.getBoundingClientRect(); if (!q.width || !q.height || q.bottom < 0 || q.top > innerHeight || !shown(e)) continue;
        for (let a = e.parentElement; a && a !== document.body; a = a.parentElement) {
          const cs = getComputedStyle(a), clipX = cs.overflowX !== "visible", clipY = cs.overflowY !== "visible";
          if (!clipX && !clipY) continue;
          if (cs.textOverflow === "ellipsis") break;
          const r = a.getBoundingClientRect(), sx = a.offsetWidth ? r.width / a.offsetWidth : 1, sy = a.offsetHeight ? r.height / a.offsetHeight : 1;
          // a scaled or tilted ancestor draws its box at another size than its layout: measure the clip box in the same drawn space as the child
          const box = { l: r.left + a.clientLeft * sx, t: r.top + a.clientTop * sy, r: r.left + (a.clientLeft + a.clientWidth) * sx, b: r.top + (a.clientTop + a.clientHeight) * sy };
          const cx = clipX ? Math.max(0, box.l - q.left, q.right - box.r) : 0, cy = clipY ? Math.max(0, box.t - q.top, q.bottom - box.b) : 0;
          const inside = q.right > box.l + tol && q.left < box.r - tol && q.bottom > box.t + tol && q.top < box.b - tol;
          if (Math.max(cx, cy) > tol && inside) { out.push({ key: name(e) + " \"" + (e.textContent || e.alt || "").trim().slice(0, 28) + "\" by " + name(a), cut: Math.round(Math.max(cx, cy)) }); break; }
          if (!inside) break;
        }
      }
      return out;
    }, { tol: c.tolerance, ignore: c.ignore });
    for (const x of cut) if (!found.has(x.key) || found.get(x.key).cut < x.cut) found.set(x.key, x);
  }
  const list = [...found.values()];
  results.push({ label: `${vp.join("x")} ${mode}`, pass: list.length === 0, detail: list.length ? list.map(x => `${x.key} cut ${x.cut}px`).join("; ") : "0 clipped" });
  await ctx.close();
}
await browser.close();
report("clipped", results);
