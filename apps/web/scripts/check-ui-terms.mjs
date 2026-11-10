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
];
const DISPLAY_ATTRS = new Set(["aria-label", "title", "placeholder", "alt", "label", "description", "subtitle", "hint", "error", "message", "header", "confirmLabel", "emptyText", "caption", "tooltip", "text", "heading"]);
const DISPLAY_KEYS = new Set(["title", "label", "description", "subtitle", "message", "placeholder", "header", "hint", "text", "caption", "confirmLabel", "empty", "heading", "tooltip", "body", "detail"]);
const SKIP_FILES = [/shared\/lib\/labels\.ts$/, /\.d\.ts$/, /\.test\.tsx?$/, /constants\/cities/];

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
  return /^toast\.(success|error|info)$/.test(text) || text === "getApiErrorMessage" || text === "deleteWithUndo" || /(^|\.)set\w*(Error|Message|Notice)$/.test(text);
}

function isDisplayLiteral(node) {
  const p = node.parent;
  if (!p) return false;
  if (ts.isImportDeclaration(p) || ts.isExportDeclaration(p) || ts.isLiteralTypeNode(p)) return false;
  if (ts.isJsxAttribute(p)) return DISPLAY_ATTRS.has(p.name.getText());
  if (ts.isJsxExpression(p)) return true; // {"text"} inside JSX
  if (ts.isPropertyAssignment(p) && p.initializer === node) return DISPLAY_KEYS.has(p.name.getText().replace(/["']/g, ""));
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
    for (const [re, use] of TERMS) {
      const m = re.exec(text);
      if (!m) continue;
      const line = sf.getLineAndCharacterOfPosition(node.getStart()).line;
      if (/ui-terms-ok/.test(lines[line])) continue;
      found.push({ file: rel, line: line + 1, term: m[0], use, text: text.trim().slice(0, 90) });
    }
  };
  const visit = (node) => {
    if (ts.isJsxText(node)) {
      const t = node.getText();
      if (t.trim()) report(node, t);
    } else if ((ts.isStringLiteral(node) || ts.isNoSubstitutionTemplateLiteral(node)) && isDisplayLiteral(node)) {
      report(node, node.text);
    } else if (ts.isTemplateExpression(node) && (isDisplayLiteral(node) || isDisplayCall(node))) {
      report(node, [node.head.text, ...node.templateSpans.map((s) => s.literal.text)].join(" "));
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
