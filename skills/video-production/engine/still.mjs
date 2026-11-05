import { mkdirSync } from "node:fs";
import { openFilm, frameSize } from "./lib.mjs";

const [fmt, ...ts] = process.argv.slice(2);
mkdirSync("shots", { recursive: true });
const { page, close } = await openFilm({ fmt, viewport: frameSize(fmt) });
for (const t of ts) { await page.evaluate(t => window.render(t), +t); await page.screenshot({ path: `shots/${fmt}-${t}.jpg`, type: "jpeg", quality: 80 }); }
await close();
