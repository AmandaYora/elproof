#!/usr/bin/env node
// Writes a Brotli (.br) and a gzip (.gz) copy next to every text file of the build, after `vite build` and the prerender
// (run by `npm run prerender`, which `npm run build` runs). The Go server sends the copy the browser accepts, with
// Content-Encoding, so the JS and CSS travel compressed without compressing on every request (apps/api/cmd/server/spa.go).
// A copy is only kept when it saves at least a tenth; the server then sends the file itself.
import fs from "node:fs";
import path from "node:path";
import zlib from "node:zlib";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const dist = path.join(root, "dist");
if (!fs.existsSync(dist)) {
  console.error("dist/ not found: run `npm run build`");
  process.exit(2);
}

const TEXT = new Set([".js", ".mjs", ".css", ".html", ".svg", ".json", ".xml", ".txt", ".webmanifest"]);
const MIN_BYTES = 1024;
// The build's own bookkeeping, never sent to a browser (spa.go refuses dot-folders and routes.json).
const SKIP = (rel) => rel === "routes.json" || rel.split(path.sep).some((s) => s.startsWith("."));

function* walk(dir) {
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const full = path.join(dir, entry.name);
    if (entry.isDirectory()) yield* walk(full);
    else yield full;
  }
}

let files = 0;
let before = 0;
let afterBr = 0;
for (const full of walk(dist)) {
  const rel = path.relative(dist, full);
  if (full.endsWith(".br") || full.endsWith(".gz")) {
    // A copy from an earlier run: written again below from the current file, or dropped.
    fs.rmSync(full);
    continue;
  }
  if (SKIP(rel) || !TEXT.has(path.extname(full))) continue;
  const raw = fs.readFileSync(full);
  if (raw.length < MIN_BYTES) continue;

  const br = zlib.brotliCompressSync(raw, {
    params: {
      [zlib.constants.BROTLI_PARAM_MODE]: zlib.constants.BROTLI_MODE_TEXT,
      [zlib.constants.BROTLI_PARAM_QUALITY]: zlib.constants.BROTLI_MAX_QUALITY,
      [zlib.constants.BROTLI_PARAM_SIZE_HINT]: raw.length,
    },
  });
  const gz = zlib.gzipSync(raw, { level: 9 });
  const worth = (c) => c.length <= raw.length * 0.9;
  if (worth(br)) fs.writeFileSync(`${full}.br`, br);
  if (worth(gz)) fs.writeFileSync(`${full}.gz`, gz);
  if (worth(br) || worth(gz)) {
    files += 1;
    before += raw.length;
    afterBr += worth(br) ? br.length : raw.length;
  }
}

const kb = (n) => (n / 1024).toFixed(0);
console.log(`precompressed ${files} files: ${kb(before)} KB → ${kb(afterBr)} KB brotli (gzip copies beside them)`);
