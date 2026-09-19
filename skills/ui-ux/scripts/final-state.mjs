// Content never depends on JS: without JS and with reduced motion every piece is in its final state, nothing overflows sideways, and the first screen carries the promise.
// `allowHidden` lists the "before" states that are meant to stay hidden in the final state (the old title a scene replaces, a key that disappears).
import { parseArgs, launch, open, scrollTo, report } from "./lib.mjs";

const { url, viewports, cfg } = parseArgs({ viewports: "390x844,1440x900" });
const c = { scope: "main", firstScreen: [], allowHidden: [], ...(cfg.finalState || {}) };
const browser = await launch();
const results = [];
for (const vp of viewports) for (const mode of ["js", "reduced", "nojs"]) {
  const opts = mode === "nojs" ? { javaScriptEnabled: false } : mode === "reduced" ? { reducedMotion: "reduce" } : {};
  const { ctx, page } = await open(browser, url, vp, { ...opts, settleMs: mode === "js" ? 1500 : 600 });
  const errors = []; page.on("pageerror", e => errors.push(e.message));
  const H = await page.evaluate(() => document.documentElement.scrollHeight);
  let maxW = 0;
  for (let y = 0; y <= H; y += 400) { await scrollTo(page, y, 40); maxW = Math.max(maxW, await page.evaluate(() => document.documentElement.scrollWidth)); }
  await scrollTo(page, 0, 200);
  const r = await page.evaluate(({ scope, firstScreen, allowHidden }) => {
    const off = firstScreen.filter(s => { const e = document.querySelector(s); return !e || e.getBoundingClientRect().bottom > innerHeight; });
    const allow = allowHidden.length ? allowHidden.join(",") : null;
    const hidden = [...document.querySelectorAll(scope + " *")].filter(e => !e.childElementCount && e.textContent.trim() && !(allow && e.closest(allow))).filter(e => {
      for (let a = e; a && a !== document.body; a = a.parentElement) { const cs = getComputedStyle(a); if (cs.opacity === "0" || cs.visibility === "hidden" || cs.display === "none") return true; }
      return false;
    }).map(e => e.textContent.trim().slice(0, 24));
    return { off, hidden: [...new Set(hidden)] };
  }, c);
  const final = mode === "js" || r.hidden.length === 0;
  const pass = maxW === vp[0] && r.off.length === 0 && final && errors.length === 0;
  results.push({ label: `${vp.join("x")} ${mode}`, pass, detail: `scrollWidth ${maxW}/${vp[0]}` + (r.off.length ? `; off first screen: ${r.off.join(", ")}` : "") + (mode !== "js" && r.hidden.length ? `; hidden: ${r.hidden.slice(0, 20).join(" | ")}` : "") + (errors.length ? `; errors ${errors.length}` : "") });
  await ctx.close();
}
await browser.close();
report("final-state", results);
