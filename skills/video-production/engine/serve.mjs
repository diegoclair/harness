import { createServer } from "node:http";
import { createReadStream, existsSync, realpathSync, statSync } from "node:fs";
import path from "node:path";
import { mountedFile } from "./mounts.mjs";

const TYPES = {
  ".html": "text/html; charset=utf-8", ".js": "text/javascript; charset=utf-8", ".mjs": "text/javascript; charset=utf-8",
  ".css": "text/css; charset=utf-8", ".json": "application/json; charset=utf-8", ".svg": "image/svg+xml",
  ".png": "image/png", ".jpg": "image/jpeg", ".woff2": "font/woff2", ".wav": "audio/wav", ".mp4": "video/mp4",
};

// A secret is refused by the name of the file actually read, so neither a symlink nor an encoding gets one out.
const SECRET = /(^|\.)env($|\.)/;

function contained(base, file) {
  const real = realpathSync(file), rel = path.relative(realpathSync(base), real);
  return rel.startsWith("..") || path.isAbsolute(rel) || SECRET.test(path.basename(real)) ? null : real;
}

// A folder holding film.json has no page of its own, so its film.html is answered by the shared player.
export function resolveRequest(root, mounts, player, urlPath) {
  let clean;
  try { clean = path.posix.normalize(decodeURIComponent(urlPath)); } catch { return null; }
  if (clean.includes("\0")) return null;
  const hit = mountedFile(mounts, clean), base = hit?.base ?? root, file = hit?.file ?? path.join(root, clean);
  if (!existsSync(file) && path.basename(file) === "film.html" && existsSync(path.join(path.dirname(file), "film.json"))) return player;
  return existsSync(file) && statSync(file).isFile() ? contained(base, file) : null;
}

// Port 0 lets the system pick, which is what allows several films to run side by side.
export function startServer(root, mounts, player) {
  const server = createServer((req, res) => {
    const file = resolveRequest(root, mounts, player, new URL(req.url, "http://x").pathname);
    if (!file) { res.writeHead(404).end(); return; }
    res.writeHead(200, { "Content-Type": TYPES[path.extname(file)] ?? "application/octet-stream", "Cache-Control": "no-store" });
    createReadStream(file).pipe(res);
  });
  return new Promise(resolve => server.listen(0, "127.0.0.1", () => resolve({
    origin: `http://127.0.0.1:${server.address().port}`,
    stop: () => { server.closeAllConnections(); server.close(); },
  })));
}
