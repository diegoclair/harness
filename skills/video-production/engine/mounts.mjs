import { registerHooks } from "node:module";
import path from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

export const ENGINE = path.dirname(fileURLToPath(import.meta.url));

// One table answers where a URL path lives, for the page server and for scenes imported in Node alike.
export function mountTable(kit) {
  return [["/__engine/", ENGINE], ["/__kit/", kit.dir], ...Object.entries(kit.mounts ?? {}).map(([url, dir]) => [url, path.resolve(kit.dir, dir)])];
}

export function mountedFile(table, urlPath) {
  const hit = table.find(([url]) => urlPath.startsWith(url));
  return hit ? { base: hit[1], file: path.join(hit[1], urlPath.slice(hit[0].length)) } : null;
}

export function registerMounts(table) {
  registerHooks({
    resolve(specifier, context, next) {
      const hit = mountedFile(table, specifier);
      return next(hit ? pathToFileURL(hit.file).href : specifier, context);
    },
  });
}
