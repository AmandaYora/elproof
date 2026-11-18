#!/usr/bin/env node
// Finds wording in UI text that is outside the glossary (PLAN.md Tahap 7, DESIGN.md §7): "Project", "Client",
// "Evidence", "Timeline"... in text a person reads. It looks only at what is shown (JSX text, labels/titles/placeholders,
// toast and error messages, display properties), never at identifiers, enum values, API paths or comments.
// Put `// ui-terms-ok` on a line to allow a deliberate exception. Exit code 1 when anything is found.
import fs from "node:fs";
import path from "node:path";
import { createRequire } from "node:module";
const require = createRequire(import.meta.url);
const ts = require("typescript");

const ROOT = path.resolve(path.dirname(new URL(import.meta.url).pathname), "../src");
const TERMS = [
  [/\bProjects?\b/i, "Proyek"],
  [/\bClients?\b/i, "Klien"],
  [/\bInvoices?\b/i, "Tagihan"],
  [/\bEvidence\b/i, "Bukti / Dokumen"],
  [/\bIssues?\b/i, "Kendala"],
  [/\bTimelines?\b/i, "Jadwal"],
  [/\bMilestones?\b/i, "Tahapan"],
  [/\bSubmit\b/i, 'kata kerja + objek, mis. "Simpan pembayaran"'],
  [/\bStaf{1,2}\b/i, "Tim / anggota tim"],
  // English verbs that crept into buttons and titles (Tahap 9). "Reset filter" is the term DESIGN.md §8 names for the empty-filter state.
  [/\bExport\b/i, "Ekspor"],
  [/\bImport\b/i, "Impor"],
  [/\bDownload\b/i, "Unduh"],
  [/\bUpload\b/i, "Unggah"],
  [/\bPassword\b/i, "Kata sandi"],
  [/\bReset\b(?! filter)/i, "Atur ulang"],
  [/\bread-only\b/i, "baca-saja"],
  // A vendor's price package (QA EP-15): one word per package, the stored field is priceAkad.
  [/\bPemberkatan\b/i, "Akad (paket harga vendor)"],
  [/\bResepsi Only\b/i, "Resepsi saja"],
  // QA FQ-08 / RT-06 (16 Nov 2026): signing in is "Masuk", the WO's start page is "Beranda".
  [/\bLog ?in\b/i, "Masuk (mis. \"Masuk dengan Google\")"],
  [/\bLog ?out\b/i, "Keluar"],
  [/\bSign ?(in|up)\b/i, "Masuk / Daftar"],
  [/\bDashboard\b/i, "Beranda (WO) / Ringkasan (Platform Console)"],
  [/\bProgress\b/i, "Progres"],
  // The shared sign-in page reads the same for everyone (18 Nov 2026): Indonesian words only.
  [/\bUsername\b/i, "Nama pengguna"],
  [/(?<!(Google|Microsoft) )\bAuthenticator\b/i, "autentikator (nama produk seperti Google Authenticator boleh)"],
];
// The platform team's own console is not part of the product people outside ElProof see (18 Nov 2026): no text outside
// modules/platform-admin names it or its admins. Public pages speak of "personel ElProof yang berwenang" instead.
const INTERNAL_TERMS = [
  [/\badmin (platform|ElProof)\b|\bplatform admin\b|\bPlatform Console\b|\bsuper ?admin\b/i, "jangan sebut tim/konsol internal di luar modules/platform-admin"],
];
const INTERNAL_OK = /^modules\/platform-admin\//;
const DISPLAY_ATTRS = new Set(["aria-label", "title", "placeholder", "alt", "label", "description", "subtitle", "hint", "error", "message", "header", "confirmLabel", "emptyText", "caption", "tooltip", "text", "heading"]);
const DISPLAY_KEYS = new Set(["title", "label", "description", "subtitle", "message", "placeholder", "header", "hint", "text", "caption", "confirmLabel", "empty", "heading", "tooltip", "body", "detail", "fallback"]);
const SKIP_FILES = [/shared\/lib\/labels\.ts$/, /\.d\.ts$/, /\.test\.tsx?$/, /constants\/cities/];

// No capitals-only labels (RISET WO Console §7.3, DESIGN.md): a label is 12–13 px, sentence case, muted. A class string with
// `uppercase` is flagged unless it also sets `font-mono` (a code people type in capitals, e.g. a voucher code) or its line
// carries `// ui-terms-ok`.
const UPPERCASE_CLASS = /(^|\s)([\w-]+:)*uppercase(?=\s|$)/;
// Screens being reworked on other branches of the WO Console revamp (2026-11-16): they still carry uppercase labels and are
// checked once those branches are merged. Remove an entry when its module is clean.
// Every console module follows the no-uppercase rule since the WO Console revamp merged (16 Nov 2026).
const UPPERCASE_PENDING = [];

function* walk(dir) {
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const full = path.join(dir, entry.name);
    if (entry.isDirectory()) yield* walk(full);
    else if (/\.tsx?$/.test(entry.name)) yield full;
  }
}

function isDisplayCall(node) {
  // toast.success("..."), toast.error("..."), getApiErrorMessage(err, "fallback")
  let call = node.parent;
  while (call && !ts.isCallExpression(call)) call = call.parent;
  if (!call) return false;
  const text = call.expression.getText();
  // setError("..."), setFileError("..."), form.setFormError("..."), setLoginNotice("..."): state that is drawn as text.
  // z.string().min(3, "…"), .refine(fn, "…"): a schema message is shown under the field (QA FQ-08 found "Isi nama timeline" there).
  if (/\.(min|max|length|regex|email|url|nonempty|refine|superRefine|positive|nonnegative|int|gte|lte|gt|lt|startsWith|endsWith)$/.test(text)) return true;
  return /^toast\.(success|error|info)$/.test(text) || text === "getApiErrorMessage" || text === "deleteWithUndo" || /(^|\.)set\w*(Error|Message|Notice)$/.test(text);
}

function isDisplayLiteral(node) {
  const p = node.parent;
  if (!p) return false;
  if (ts.isImportDeclaration(p) || ts.isExportDeclaration(p) || ts.isLiteralTypeNode(p)) return false;
  if (ts.isJsxAttribute(p)) return DISPLAY_ATTRS.has(p.name.getText());
  if (ts.isJsxExpression(p)) return true; // {"text"} inside JSX
  if (ts.isPropertyAssignment(p) && p.initializer === node) return DISPLAY_KEYS.has(p.name.getText().replace(/["']/g, ""));
  // body: ["…", "…"]: the paragraphs of a legal page (QA FQ-09 found "project", "Client", "timeline" there).
  if (ts.isArrayLiteralExpression(p) && ts.isPropertyAssignment(p.parent) && p.parent.initializer === p) {
    return DISPLAY_KEYS.has(p.parent.name.getText().replace(/["']/g, ""));
  }
  if (ts.isCallExpression(p) || ts.isBinaryExpression(p) || ts.isConditionalExpression(p) || ts.isTemplateSpan(p)) return isDisplayCall(node);
  return false;
}

const found = [];
for (const file of walk(ROOT)) {
  const rel = path.relative(ROOT, file).split(path.sep).join("/");
  if (SKIP_FILES.some((re) => re.test(rel))) continue;
  const src = fs.readFileSync(file, "utf8");
  const lines = src.split("\n");
  const sf = ts.createSourceFile(file, src, ts.ScriptTarget.Latest, true, file.endsWith("x") ? ts.ScriptKind.TSX : ts.ScriptKind.TS);
  const report = (node, text) => {
    for (const [re, use] of INTERNAL_OK.test(rel) ? TERMS : [...TERMS, ...INTERNAL_TERMS]) {
      const m = re.exec(text);
      if (!m) continue;
      const line = sf.getLineAndCharacterOfPosition(node.getStart()).line;
      if (/ui-terms-ok/.test(lines[line])) continue;
      found.push({ file: rel, line: line + 1, term: m[0], use, text: text.trim().slice(0, 90) });
    }
  };
  const checkCase = file.endsWith(".tsx") && !UPPERCASE_PENDING.some((re) => re.test(rel));
  const visit = (node) => {
    if (checkCase && (ts.isStringLiteral(node) || ts.isNoSubstitutionTemplateLiteral(node) || ts.isTemplateHead(node) || ts.isTemplateMiddle(node) || ts.isTemplateTail(node))) {
      const t = node.text;
      if (UPPERCASE_CLASS.test(t) && !/(^|\s)font-mono(\s|$)/.test(t)) {
        const line = sf.getLineAndCharacterOfPosition(node.getStart()).line;
        if (!/ui-terms-ok/.test(lines[line])) {
          found.push({ file: rel, line: line + 1, term: "uppercase", use: "12–13 px sentence-case label, text-muted-foreground", text: t.trim().slice(0, 90) });
        }
      }
    }
    if (ts.isJsxText(node)) {
      const t = node.getText();
      if (t.trim()) report(node, t);
    } else if ((ts.isStringLiteral(node) || ts.isNoSubstitutionTemplateLiteral(node)) && isDisplayLiteral(node)) {
      report(node, node.text);
    } else if (ts.isTemplateExpression(node) && (isDisplayLiteral(node) || isDisplayCall(node))) {
      report(node, [node.head.text, ...node.templateSpans.map((s) => s.literal.text)].join(" "));
    }
    // A countdown ("H-5", "Hari H", "H+3") is built only by formatDayCountdown in shared/lib/formatters.ts: a template
    // with "H-" or "H+" right before a value prints "H--3" once the day has passed (QA EP-15).
    if (ts.isTemplateExpression(node) && !rel.endsWith("shared/lib/formatters.ts")) {
      const parts = [node.head.text, ...node.templateSpans.map((s) => s.literal.text)];
      if (parts.slice(0, -1).some((p) => /(^|[^A-Za-z])H[-+]$/.test(p))) {
        const line = sf.getLineAndCharacterOfPosition(node.getStart()).line;
        if (!/ui-terms-ok/.test(lines[line])) {
          found.push({ file: rel, line: line + 1, term: "H-${…}", use: "formatDayCountdown(days)", text: node.getText().slice(0, 90) });
        }
      }
    }
    ts.forEachChild(node, visit);
  };
  visit(sf);
}

if (found.length === 0) {
  console.log("check-ui-terms: no wording outside the glossary.");
  process.exit(0);
}
const byFile = new Map();
for (const f of found) byFile.set(f.file, [...(byFile.get(f.file) ?? []), f]);
for (const [file, items] of byFile) {
  console.log(file);
  for (const i of items) console.log(`  ${i.line}: "${i.term}" -> ${i.use}   | ${i.text}`);
}
console.log(`\ncheck-ui-terms: ${found.length} stray term(s) in ${byFile.size} file(s).`);
process.exit(1);
