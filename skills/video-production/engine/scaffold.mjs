import { existsSync, mkdirSync, readFileSync, writeFileSync } from "node:fs";
import path from "node:path";
import { kitFile } from "./kit.mjs";

// A film's approved narration lives in its voice folder and is versioned; every other render output stays out of git.
const GITIGNORE = ["node_modules/", "out/", "shots/", "*.mp4", "*.wav", "!voice/*.wav", "!voice/*.json", "cues*.json", ".build/", "clips/", ""].join("\n");

export function scaffold(dir, title, kit) {
  if (existsSync(path.join(dir, "film.json"))) throw new Error(`${dir} already has a film.json`);
  const film = { ...JSON.parse(readFileSync(kitFile(kit, kit.template), "utf8")), title: title ?? path.basename(path.resolve(dir)) };
  mkdirSync(dir, { recursive: true });
  writeFileSync(path.join(dir, "film.json"), `${JSON.stringify(film, null, 2)}\n`);
  writeFileSync(path.join(dir, ".gitignore"), GITIGNORE);
  return `${path.join(dir, "film.json")} is ready: write the lines, "film scenes ${dir}" lists what each can use, then "film run ${dir}"`;
}
