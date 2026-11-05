import { execFileSync } from "node:child_process";
import { existsSync, readFileSync } from "node:fs";
import path from "node:path";

const KIT_FILE = path.join("video-kit", "kit.json");

export function loadKit(dir) {
  return { ...JSON.parse(readFileSync(path.join(dir, "kit.json"), "utf8")), dir: path.resolve(dir) };
}

// The kit is a film's anchor: it owns the brand, the voices and the server root of every film below it.
export function findKit(filmDir) {
  for (let dir = path.resolve(filmDir); ; dir = path.dirname(dir)) {
    if (existsSync(path.join(dir, KIT_FILE))) return loadKit(path.join(dir, "video-kit"));
    if (dir === path.dirname(dir)) throw new Error(`no ${KIT_FILE} in any folder above ${filmDir}`);
  }
}

export function serverRoot(kit, filmDir, flag) {
  if (flag) return path.resolve(flag);
  if (kit.root) return path.resolve(kit.dir, kit.root);
  try {
    return execFileSync("git", ["-C", filmDir, "rev-parse", "--show-toplevel"], { encoding: "utf8", stdio: ["ignore", "pipe", "ignore"] }).trim();
  } catch {
    throw new Error(`${filmDir} is in no git repository and its kit declares no root: pass --root`);
  }
}

export const kitFile = (kit, rel) => path.resolve(kit.dir, rel);
