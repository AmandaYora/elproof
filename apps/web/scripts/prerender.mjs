#!/usr/bin/env node
// Writes the public pages to HTML after `vite build` (ADR-0043). Run by `npm run build`; needs dist/ (the browser build) and
// dist-ssr/entry-server.js (`vite build --ssr src/entry-server.tsx --outDir dist-ssr`). Produces:
//   dist/prerendered/<page>.html  each public page: its own <title>, description, canonical and share tags, and its markup
//   dist/routes.json              which file answers which public path, and every route pattern of the app (the Go server
//                                 answers 404 for a path that matches none, see apps/api/cmd/server/spa.go)
//   dist/sitemap.xml              the indexable public pages
//   dist/robots.txt               public/robots.txt plus the Sitemap line
// dist/index.html stays the empty app shell: the browser build, the budget check and every non-public route use it.
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const dist = path.join(root, "dist");
const outDir = path.join(dist, "prerendered");
const entry = path.join(root, "dist-ssr", "entry-server.js");

for (const needed of [path.join(dist, "index.html"), entry]) {
  if (!fs.existsSync(needed)) {
    console.error(`${path.relative(root, needed)} not found: run \`npm run build\``);
    process.exit(2);
  }
}

const { render, pages, routePatterns, SITE_URL, APP_NAME } = await import(pathToFileURL(entry).href);

const escapeHtml = (s) => String(s).replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;").replace(/"/g, "&quot;");
const url = (p) => `${SITE_URL}${p}`;
const fileFor = (p) => `prerendered/${p === "/" ? "index" : p.slice(1).replace(/\//g, "__")}.html`;

function headFor(page) {
  const title = `${page.title} — ${APP_NAME}`;
  const tags = [
    `<title>${escapeHtml(title)}</title>`,
    `<meta name="description" content="${escapeHtml(page.description)}" />`,
  ];
  if (page.noindex) tags.push(`<meta name="robots" content="noindex" />`);
  else tags.push(`<link rel="canonical" href="${escapeHtml(url(page.path))}" />`, `<meta property="og:url" content="${escapeHtml(url(page.path))}" />`);
  tags.push(
    `<meta property="og:type" content="website" />`,
    `<meta property="og:site_name" content="${escapeHtml(APP_NAME)}" />`,
    `<meta property="og:locale" content="id_ID" />`,
    `<meta property="og:title" content="${escapeHtml(title)}" />`,
    `<meta property="og:description" content="${escapeHtml(page.description)}" />`,
    `<meta property="og:image" content="${escapeHtml(url("/icons/icon-512.png"))}" />`,
    `<meta name="twitter:image" content="${escapeHtml(url("/icons/icon-512.png"))}" />`,
    `<meta name="twitter:card" content="summary" />`,
    `<meta name="twitter:title" content="${escapeHtml(title)}" />`,
    `<meta name="twitter:description" content="${escapeHtml(page.description)}" />`,
  );
  return tags.join("\n    ");
}

const shell = fs.readFileSync(path.join(dist, "index.html"), "utf8");
const META_BLOCK = /<!-- page-meta[\s\S]*?<!-- \/page-meta -->/;
const ROOT_DIV = '<div id="root"></div>';
if (!META_BLOCK.test(shell) || !shell.includes(ROOT_DIV) || !shell.includes('<html lang="id">')) {
  console.error("dist/index.html lacks the page-meta block, the empty #root or <html lang=\"id\">: see index.html");
  process.exit(1);
}

fs.rmSync(outDir, { recursive: true, force: true });
fs.mkdirSync(outDir, { recursive: true });

const manifestPages = {};
for (const page of pages) {
  const markup = await render(page.path);
  if (!markup.trim()) throw new Error(`${page.path} rendered nothing`);
  // In-order HTML only: no inline script (a strict CSP blocks it) and no part still waiting for one.
  if (/<script|<template|<!--\$\?-->/.test(markup)) throw new Error(`${page.path}: the markup still has streamed parts or inline scripts`);
  if (markup.includes("<!--$!-->")) console.warn(`${page.path}: a part could not render at build time and renders in the browser`);
  if (/\sstyle="/.test(markup)) console.warn(`${page.path}: a style attribute in the markup is ignored under a strict CSP (style-src without 'unsafe-inline'); prefer a class`);
  const html = shell
    .replace(META_BLOCK, () => headFor(page))
    // The public surface's tokens (useDocumentSurface) apply from the first paint instead of after the script runs.
    .replace('<html lang="id">', '<html lang="id" data-surface="marketing">')
    .replace(ROOT_DIV, () => `<div id="root" data-prerendered="${escapeHtml(page.path)}">${markup}</div>`);
  const file = fileFor(page.path);
  fs.writeFileSync(path.join(dist, file), html);
  manifestPages[page.path] = file;
  console.log(`prerendered ${page.path.padEnd(30)} → dist/${file} (${(Buffer.byteLength(html) / 1024).toFixed(1)} KB)`);
}

// The shell for a path no route answers (QA RT-06): the same empty app, but its tab already says what happened and search
// engines drop it. The browser draws the not-found page itself (the way back depends on who is signed in). The Go server
// sends it with status 404 when routes.json names it under "notFound", and index.html otherwise.
const notFound = "prerendered/404.html";
fs.writeFileSync(
  path.join(dist, notFound),
  shell.replace(META_BLOCK, () =>
    [`<title>${escapeHtml(`Halaman tidak ditemukan — ${APP_NAME}`)}</title>`, `<meta name="robots" content="noindex" />`].join("\n    "),
  ),
);
console.log(`prerendered 404 shell${" ".repeat(19)} → dist/${notFound}`);

const routes = routePatterns();
fs.writeFileSync(path.join(dist, "routes.json"), `${JSON.stringify({ pages: manifestPages, routes, notFound }, null, 2)}\n`);
console.log(`routes.json: ${Object.keys(manifestPages).length} pages, ${routes.length} route patterns`);

const indexable = pages.filter((p) => !p.noindex);
const sitemap = [
  '<?xml version="1.0" encoding="UTF-8"?>',
  '<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">',
  ...indexable.map((p) => `  <url><loc>${escapeHtml(url(p.path))}</loc></url>`),
  "</urlset>",
  "",
].join("\n");
fs.writeFileSync(path.join(dist, "sitemap.xml"), sitemap);
console.log(`sitemap.xml: ${indexable.length} pages`);

const robotsPath = path.join(dist, "robots.txt");
const robots = fs.existsSync(robotsPath) ? fs.readFileSync(robotsPath, "utf8").replace(/^Sitemap:.*\n?/gim, "") : "";
fs.writeFileSync(robotsPath, `${robots.replace(/\n*$/, "\n")}\nSitemap: ${url("/sitemap.xml")}\n`);
