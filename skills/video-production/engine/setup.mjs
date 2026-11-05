import { spawnSync } from "node:child_process";
import { copyFileSync, existsSync, mkdirSync, readFileSync } from "node:fs";
import { createRequire } from "node:module";
import os from "node:os";
import path from "node:path";
import { ENGINE } from "./mounts.mjs";

// A reinstall wipes the skill folder, so everything downloaded lives here instead.
export const CACHE = path.join(os.homedir(), ".cache", "video-production");
export const PYTHON = path.join(CACHE, "venv", "bin", "python");
export const VOICE_CACHE = path.join(CACHE, "voice");
const NODE_DIR = path.join(CACHE, "node");

export function requireDependency(name) {
  try {
    return createRequire(path.join(NODE_DIR, "package.json"))(name);
  } catch (e) {
    if (e.code === "MODULE_NOT_FOUND") throw new Error(`${name} is missing: run "film setup" once on this machine`);
    throw e;
  }
}

function run(cmd, args, opts = {}) {
  const r = spawnSync(cmd, args, { stdio: "inherit", ...opts });
  if (r.error) throw r.error;
  if (r.status) throw new Error(`${path.basename(cmd)} ${args.slice(0, 2).join(" ")} exited ${r.status}`);
}

const same = (a, b) => existsSync(a) && existsSync(b) && readFileSync(a).equals(readFileSync(b));

function nodeModules() {
  mkdirSync(NODE_DIR, { recursive: true });
  const lock = path.join(NODE_DIR, "package-lock.json");
  if (same(path.join(ENGINE, "package-lock.json"), lock) && existsSync(path.join(NODE_DIR, "node_modules", "playwright-core"))) return;
  for (const f of ["package.json", "package-lock.json"]) copyFileSync(path.join(ENGINE, f), path.join(NODE_DIR, f));
  run("npm", ["ci", "--no-audit", "--no-fund"], { cwd: NODE_DIR });
}

function pythonEnv() {
  const pinned = path.join(CACHE, "venv", "requirements.txt");
  if (same(path.join(ENGINE, "requirements.txt"), pinned)) return;
  if (!existsSync(PYTHON)) run("python3", ["-m", "venv", path.join(CACHE, "venv")]);
  run(PYTHON, ["-m", "pip", "install", "-q", "-r", path.join(ENGINE, "requirements.txt")]);
  copyFileSync(path.join(ENGINE, "requirements.txt"), pinned);
}

export function setup() {
  nodeModules();
  // Playwright deletes browsers it thinks unused, and those may belong to another project on this machine.
  run(process.execPath, [path.join(NODE_DIR, "node_modules", "playwright-core", "cli.js"), "install", "chromium"], { env: { ...process.env, PLAYWRIGHT_SKIP_BROWSER_GC: "1" } });
  pythonEnv();
  run(PYTHON, [path.join(ENGINE, "whisper_model.py")]);
  mkdirSync(VOICE_CACHE, { recursive: true });
}
