import { writeFileSync } from "node:fs";
import { openFilm } from "./lib.mjs";

const { page, close } = await openFilm();
writeFileSync("cues.json", JSON.stringify(await page.evaluate(() => window.CUES)));
await close();
