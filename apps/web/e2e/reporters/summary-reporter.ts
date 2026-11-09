import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
import type { FullResult, Reporter, TestCase, TestResult } from "@playwright/test/reporter";

const REPORTS = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../reports");

interface AxeViolation {
  rule: string;
  impact: string;
  help: string;
  nodes: number;
  sample: string;
}
interface AxePage {
  page: string;
  viewport: string;
  violations: AxeViolation[];
}

/** Collects the `axe` and `perf` attachments of the run and writes the JSON and Markdown baselines. */
export default class SummaryReporter implements Reporter {
  private axe: AxePage[] = [];
  private perf: any[] = [];

  onTestEnd(_test: TestCase, result: TestResult) {
    for (const a of result.attachments) {
      if (!a.body) continue;
      if (a.name === "axe") this.axe.push(JSON.parse(a.body.toString()));
      if (a.name === "perf") this.perf.push(JSON.parse(a.body.toString()));
    }
  }

  onEnd(_result: FullResult) {
    fs.mkdirSync(REPORTS, { recursive: true });
    if (this.axe.length) this.writeAxe();
    if (this.perf.length) this.writePerf();
  }

  private writeAxe() {
    const pages = [...this.axe].sort((a, b) => (a.page + a.viewport).localeCompare(b.page + b.viewport));
    const byImpact: Record<string, number> = { critical: 0, serious: 0, moderate: 0, minor: 0 };
    const byRule: Record<string, { impact: string; help: string; pages: number; nodes: number }> = {};
    for (const p of pages) {
      for (const v of p.violations) {
        byImpact[v.impact] = (byImpact[v.impact] ?? 0) + 1;
        const r = (byRule[v.rule] ??= { impact: v.impact, help: v.help, pages: 0, nodes: 0 });
        r.pages += 1;
        r.nodes += v.nodes;
      }
    }
    const report = {
      generatedBy: "e2e/tests/axe.spec.ts (tags wcag2a, wcag2aa, wcag21a, wcag21aa, wcag22aa)",
      pagesScanned: pages.length,
      violationsByImpact: byImpact,
      byRule: Object.fromEntries(Object.entries(byRule).sort((a, b) => b[1].nodes - a[1].nodes)),
      pages,
    };
    fs.writeFileSync(path.join(REPORTS, "axe-baseline.json"), JSON.stringify(report, null, 2));
  }

  private writePerf() {
    const rows = [...this.perf].sort((a, b) => a.id.localeCompare(b.id));
    const bundle = rows.find((r) => r.bundle)?.bundle ?? null;
    const json = {
      profile: "mobile 375x812, Android Chrome UA, CPU 4x, 1.6 Mbps / 150 ms RTT, cold cache, production build via vite preview (gzip), fonts served from the local cache",
      bundle: bundle && { totalJsRaw: bundle.totalJsRaw, totalJsGzip: bundle.totalJsGzip, chunks: bundle.chunks },
      routes: rows.map(({ bundle: _b, ...r }) => r),
    };
    fs.writeFileSync(path.join(REPORTS, "baseline.json"), JSON.stringify(json, null, 2));

    const kb = (n: number) => (n / 1024).toFixed(1);
    const lines = [
      "# Baseline performa dan jumlah panggilan API",
      "",
      "Diukur dengan `npm run e2e:perf` pada UI lama (sebelum revamp). Profil: HP 375×812, UA Android Chrome, CPU 4×,",
      "1,6 Mbps / 150 ms RTT, cache dingin, build produksi lewat `vite preview` (yang mengirim gzip; kolom JS gzip dihitung dari berkas",
      "build untuk chunk yang benar-benar dimuat). Jam peramban memakai waktu nyata (jam palsu Playwright ikut menggantikan Resource Timing). INP tidak dapat diukur dari skrip; kolom **INP-like** adalah durasi acara terbesar pada satu klik navigasi",
      "(indikatif, bukan INP lapangan). Angka API dihitung sampai jaringan idle dan **termasuk panggilan kerangka** (branding, izin, langganan,",
      "pengumuman), bukan hanya data halaman.",
      "",
      "| Layar | Panggilan API | JSON API (KB) | Chunk JS | JS gzip (KB) | LCP (ms) | CLS | TBT-like (ms) | INP-like (ms) |",
      "|---|---:|---:|---:|---:|---:|---:|---:|---:|",
      ...rows.map(
        (r) =>
          `| ${r.label} | ${r.apiCalls} | ${kb(r.apiBytes)} | ${r.jsChunks} | ${kb(r.jsGzip)} | ${r.lcpMs} | ${r.cls} | ${r.tbtMs} | ${r.inpLikeMs ?? "-"} |`
      ),
      "",
    ];
    if (bundle) {
      lines.push(
        `**Total JS seluruh aplikasi:** ${kb(bundle.totalJsRaw)} KB mentah, **${kb(bundle.totalJsGzip)} KB gzip** (${bundle.chunks.length} chunk).`,
        "",
        "Chunk terbesar (gzip):",
        "",
        ...bundle.chunks.slice(0, 10).map((c: { file: string; raw: number; gzip: number }) => `- \`${c.file}\`: ${kb(c.gzip)} KB gzip (${kb(c.raw)} KB mentah)`),
        ""
      );
    }
    lines.push("## Panggilan API per layar", "");
    for (const r of rows) {
      lines.push(`**${r.label}** (${r.apiCalls}):`, "", ...r.apiRequests.map((q: string) => `- ${q}`), "");
    }
    fs.writeFileSync(path.join(REPORTS, "BASELINE.md"), lines.join("\n"));
  }
}
