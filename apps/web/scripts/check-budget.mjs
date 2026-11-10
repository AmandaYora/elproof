#!/usr/bin/env node
// Performance budget gate (PLAN Tahap 9). Reads perf-budget.json and fails when the production build or, with --routes,
// the last `npm run e2e:perf` report goes over it. Prints every line so a failure says what grew.
//   node scripts/check-budget.mjs            bundle sizes of dist/ (run `npm run build` first)
//   node scripts/check-budget.mjs --routes   also per-route API calls and JS gzip from e2e/reports/baseline.json
import fs from "node:fs";
import path from "node:path";
import zlib from "node:zlib";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const budget = JSON.parse(fs.readFileSync(path.join(root, "perf-budget.json"), "utf8"));
const withRoutes = process.argv.includes("--routes");
const KB = 1024;
const failures = [];
const line = (ok, text) => console.log(`${ok ? "ok  " : "FAIL"} ${text}`);
function check(ok, text) {
  line(ok, text);
  if (!ok) failures.push(text);
}

// --- bundle (dist/)
const assets = path.join(root, "dist", "assets");
if (!fs.existsSync(assets)) {
  console.error("dist/assets not found: run `npm run build` first");
  process.exit(2);
}
const chunks = fs
  .readdirSync(assets)
  .filter((f) => f.endsWith(".js"))
  .map((file) => ({ file, gzip: zlib.gzipSync(fs.readFileSync(path.join(assets, file)), { level: 9 }).length }));
const html = fs.readFileSync(path.join(root, "dist", "index.html"), "utf8");
const entryFile = /src="\/assets\/([^"]+\.js)"/.exec(html)?.[1];
const entry = chunks.find((c) => c.file === entryFile);
const total = chunks.reduce((s, c) => s + c.gzip, 0);
const largest = [...chunks].filter((c) => c !== entry).sort((a, b) => b.gzip - a.gzip)[0];
const kb = (n) => (n / KB).toFixed(1);

check(Boolean(entry) && entry.gzip <= budget.bundle.entryGzipKB * KB, `entry ${entryFile}: ${kb(entry?.gzip ?? 0)} KB gzip (budget ${budget.bundle.entryGzipKB})`);
check(total <= budget.bundle.totalGzipKB * KB, `all JS: ${kb(total)} KB gzip in ${chunks.length} chunks (budget ${budget.bundle.totalGzipKB})`);
check(largest.gzip <= budget.bundle.largestChunkGzipKB * KB, `largest lazy chunk ${largest.file}: ${kb(largest.gzip)} KB gzip (budget ${budget.bundle.largestChunkGzipKB})`);

// --- routes (last e2e:perf report)
if (withRoutes) {
  const reportPath = path.join(root, "e2e", "reports", "baseline.json");
  if (!fs.existsSync(reportPath)) {
    console.error("e2e/reports/baseline.json not found: run `npm run e2e:perf` first");
    process.exit(2);
  }
  const report = JSON.parse(fs.readFileSync(reportPath, "utf8"));
  const seen = new Set();
  for (const r of report.routes) {
    seen.add(r.id);
    const allowed = budget.routes[r.id];
    if (!allowed) {
      check(false, `${r.id}: no budget in perf-budget.json (add one)`);
      continue;
    }
    check(r.apiCalls <= allowed.apiCalls, `${r.id}: ${r.apiCalls} API calls (budget ${allowed.apiCalls})`);
    check(r.jsGzip <= budget.routeJsGzipKB * KB, `${r.id}: ${kb(r.jsGzip)} KB JS gzip on first load (budget ${budget.routeJsGzipKB})`);
  }
  for (const id of Object.keys(budget.routes)) if (!seen.has(id)) check(false, `${id}: budgeted but not measured (the report is out of date or the route was removed)`);
}

if (failures.length) {
  console.error(`\n${failures.length} budget line(s) over. Fix the growth, or raise perf-budget.json with the reason in the same change.`);
  process.exit(1);
}
console.log("\nperformance budget: within limits.");
