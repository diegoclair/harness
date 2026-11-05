import path from "node:path";
import { requireDependency } from "./setup.mjs";

const { chromium } = requireDependency("playwright-core");

function filmUrl(fmt) {
  const { FILM_ORIGIN: origin, FILM_ROOT: root, FILM_QUERY: query } = process.env;
  if (!origin || !root) throw new Error("this step runs inside \"film run\", which serves the film");
  const rel = path.relative(root, process.cwd()).split(path.sep).join("/");
  return `${origin}/${rel}/film.html?capture=1${fmt === "v" ? "&fmt=v" : ""}${query ? `&${query}` : ""}`;
}

export const frameSize = fmt => (fmt === "v" ? { width: 1080, height: 1920 } : { width: 1920, height: 1080 });

// A frame drawn without its font, image or script is a wrong frame, so any lost request fails the step on close.
export async function openFilm({ fmt = "h", viewport, args } = {}) {
  const browser = await chromium.launch(args ? { args } : undefined);
  const page = await browser.newPage(viewport ? { viewport } : undefined);
  const lost = [];
  page.on("pageerror", e => lost.push(`page error: ${e.message}`));
  page.on("requestfailed", r => lost.push(`${r.failure()?.errorText ?? "failed"} ${r.url()}`));
  page.on("response", r => { if (r.status() >= 400) lost.push(`${r.status()} ${r.url()}`); });
  const close = async () => {
    await browser.close();
    if (lost.length) throw new Error(`the film page lost what it needed:\n  ${lost.join("\n  ")}`);
  };
  try {
    await page.goto(filmUrl(fmt));
    await page.evaluate(() => window.ready);
  } catch (e) {
    await close();
    throw e;
  }
  return { page, close };
}
