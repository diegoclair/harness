// A 3D layer left on at rest is rasterised and its text reads blurred: at rest every listed chain must be plain 2D.
import { parseArgs, launch, open, scrollToElement, report } from "./lib.mjs";

const { url, viewports, cfg } = parseArgs({ viewports: "390x844,1440x900,1440x1211,1920x1080" });
const c = { centre: null, chains: [], stopAt: "section", waitMs: 10000, ...(cfg.restFlat || {}) };
if (!c.chains.length) { console.error("restFlat.chains is empty in the config"); process.exit(2); }
const browser = await launch();
const results = [];
for (const vp of viewports) for (const js of [true, false]) {
  const { ctx, page } = await open(browser, url, vp, { javaScriptEnabled: js, settleMs: 800 });
  if (c.centre) await scrollToElement(page, c.centre, "center", js ? c.waitMs : 500);
  const bad = await page.evaluate(({ chains, stopAt }) => {
    const out = [];
    for (const start of chains) {
      for (let e = document.querySelector(start); e && !e.matches(stopAt); e = e.parentElement) {
        const cs = getComputedStyle(e), flat = cs.transform === "none" && cs.willChange === "auto" && cs.filter === "none" && ["none", "1"].includes(cs.scale) && cs.translate === "none" && cs.rotate === "none" && cs.perspective === "none";
        if (!flat) out.push(`${start} ← ${(e.className || e.tagName).toString().split(" ")[0]} {transform ${cs.transform}, will-change ${cs.willChange}, filter ${cs.filter}, scale ${cs.scale}, perspective ${cs.perspective}}`);
      }
    }
    return out;
  }, { chains: c.chains, stopAt: c.stopAt });
  results.push({ label: `${vp.join("x")} ${js ? "js" : "nojs"}`, pass: bad.length === 0, detail: bad.length ? bad.join("; ") : `${c.chains.length} chains flat` });
  await ctx.close();
}
await browser.close();
report("rest-flat", results);
