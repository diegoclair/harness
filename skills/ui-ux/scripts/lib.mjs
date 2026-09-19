import { createRequire } from "node:module";
import { readFileSync } from "node:fs";

// Playwright is resolved from the folder you run in, so each project keeps its own pinned version.
async function loadPlaywright() {
  const candidates = [process.env.PLAYWRIGHT, (() => { try { return createRequire(process.cwd() + "/").resolve("playwright"); } catch { return null; } })()].filter(Boolean);
  for (const path of candidates) {
    try { const pw = await import(path); return pw.chromium ? pw : pw.default; } catch { /* next candidate */ }
  }
  console.error("playwright not found: run from a folder where `npm i -D playwright` was done, or set PLAYWRIGHT=/abs/path/to/playwright/index.js");
  process.exit(2);
}

export function parseArgs(defaults = {}) {
  const a = process.argv.slice(2), out = { url: null, viewports: null, config: null };
  for (let i = 0; i < a.length; i++) {
    if (a[i] === "--viewports") out.viewports = a[++i];
    else if (a[i] === "--config") out.config = a[++i];
    else if (!out.url) out.url = a[i];
  }
  if (!out.url) { console.error("usage: node <script> <url> [--viewports 390x844,1440x900] [--config file.json]"); process.exit(2); }
  out.cfg = out.config ? JSON.parse(readFileSync(out.config, "utf8")) : {};
  out.viewports = (out.viewports || out.cfg.viewports || defaults.viewports || "390x844,1440x900").split(",").map(v => v.split("x").map(Number));
  return out;
}

export async function launch() {
  const pw = await loadPlaywright();
  return pw.chromium.launch({ executablePath: process.env.CHROME || undefined });
}

export async function open(browser, url, [w, h], opts = {}) {
  const ctx = await browser.newContext({ viewport: { width: w, height: h }, ...opts });
  const page = await ctx.newPage();
  await page.goto(url);
  await page.waitForTimeout(opts.settleMs ?? 600);
  return { ctx, page };
}

export async function scrollTo(page, y, settleMs = 60) {
  await page.evaluate(y => scrollTo({ top: y, behavior: "instant" }), y);
  await page.waitForTimeout(settleMs);
}

export async function scrollToElement(page, selector, block = "center", settleMs = 300) {
  await page.evaluate(({ selector, block }) => document.querySelector(selector)?.scrollIntoView({ block, behavior: "instant" }), { selector, block });
  await page.waitForTimeout(settleMs);
}

// One line per check and a final verdict; the exit code lets a gate chain the scripts.
export function report(name, results) {
  for (const r of results) console.log(`${r.pass ? "PASS" : "FAIL"} ${name} ${r.label}${r.detail ? " · " + r.detail : ""}`);
  const ok = results.every(r => r.pass);
  console.log(`${ok ? "PASS" : "FAIL"} ${name}: ${results.filter(r => r.pass).length}/${results.length} checks`);
  process.exitCode = ok ? 0 : 1;
}
