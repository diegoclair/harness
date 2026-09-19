// A scene that plays once is missed by whoever was reading the other column: each listed scene must replay while in view, and freeze on hover when asked.
// State is read from the scene's classes matching `stateClass`; a loop passes when its first state comes back after other states.
import { parseArgs, launch, open, scrollToElement, report } from "./lib.mjs";

const { url, viewports, cfg } = parseArgs({ viewports: "1440x900" });
const c = { stateClass: "^s-", items: [], ...(cfg.loops || {}) };
if (!c.items.length) { console.error("loops.items is empty in the config"); process.exit(2); }
const browser = await launch();
const results = [];
for (const vp of viewports) {
  const { ctx, page } = await open(browser, url, vp, { settleMs: 800 });
  const state = sel => page.evaluate(({ sel, re }) => { const e = document.querySelector(sel); return e ? [...e.classList].filter(k => new RegExp(re).test(k)).sort().join("+") || "-" : null; }, { sel, re: c.stateClass });
  for (const it of c.items) {
    await scrollToElement(page, it.scrollTo || it.selector, it.block || "center", 400);
    const seq = [], t0 = Date.now();
    while (Date.now() - t0 < it.cycleMs * 2.2) { const s = await state(it.selector); if (seq[seq.length - 1] !== s) seq.push(s); await page.waitForTimeout(150); }
    const first = seq.indexOf(seq.find(s => s !== "-") ?? "-"), replayed = seq.slice(first + 2).includes(seq[first]);
    results.push({ label: `${vp.join("x")} ${it.name || it.selector} replays`, pass: seq.length > 2 && replayed, detail: seq.slice(0, 8).join(" → ") + (seq.length > 8 ? " …" : "") });
    if (it.hover) {
      await page.hover(it.hover); await page.waitForTimeout(300);
      const a = await state(it.selector); await page.waitForTimeout(Math.min(6000, it.cycleMs)); const b = await state(it.selector);
      results.push({ label: `${vp.join("x")} ${it.name || it.selector} freezes on hover`, pass: a === b, detail: `${a} → ${b}` });
      await page.mouse.move(0, 0);
    }
  }
  await ctx.close();
}
await browser.close();
report("loops", results);
