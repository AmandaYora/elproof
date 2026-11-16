import { useLayoutEffect, useRef, useState } from "react";
import { EmptyState } from "@/shared/components/feedback/EmptyState";
import { LoadError } from "@/shared/components/feedback/LoadError";
import { Skeleton } from "@/shared/components/ui/Skeleton";
import { formatCurrency, formatCurrencyCompact } from "@/shared/lib/formatters";
import type { CashFlowWeek } from "@/modules/finance/types";

const MONTHS = ["Jan", "Feb", "Mar", "Apr", "Mei", "Jun", "Jul", "Agu", "Sep", "Okt", "Nov", "Des"];

function parts(iso: string): Date {
  const [y, m, d] = iso.split("-").map(Number);
  return new Date(y, m - 1, d);
}

/** "9 Nov" — the Monday that starts the week. */
function shortDay(iso: string): string {
  const d = parts(iso);
  return `${d.getDate()} ${MONTHS[d.getMonth()]}`;
}

/** "9 – 15 Nov", or "30 Nov – 6 Des" across a month. */
function weekRange(iso: string): string {
  const start = parts(iso);
  const end = new Date(start.getFullYear(), start.getMonth(), start.getDate() + 6);
  const startText = start.getMonth() === end.getMonth() ? String(start.getDate()) : shortDay(iso);
  return `${startText} – ${end.getDate()} ${MONTHS[end.getMonth()]}`;
}

/** A round axis maximum and its step (1, 2 or 5 × 10^n), four steps high. */
function niceScale(max: number): { top: number; step: number } {
  const raw = Math.max(max, 1) / 4;
  const pow = 10 ** Math.floor(Math.log10(raw));
  const step = [1, 2, 2.5, 5, 10].map((m) => m * pow).find((s) => s >= raw) ?? 10 * pow;
  return { top: step * 4, step };
}

/** A bar from the baseline up, its top corners rounded 4 px (square at the baseline). */
function barPath(x: number, y: number, w: number, h: number): string {
  const r = Math.min(4, h, w / 2);
  return `M${x},${y + h}V${y + r}Q${x},${y} ${x + r},${y}H${x + w - r}Q${x + w},${y} ${x + w},${y + r}V${y + h}Z`;
}

/** The full amount from a tablet up; on a phone the short form, so the table fits without scrolling sideways. */
function Amount({ value }: { value: number }) {
  return (
    <>
      <span className="hidden sm:inline">{formatCurrency(value)}</span>
      <span className="sm:hidden" title={formatCurrency(value)}>
        {formatCurrencyCompact(value)}
      </span>
    </>
  );
}

function useWidth<T extends HTMLElement>() {
  const ref = useRef<T>(null);
  const [width, setWidth] = useState(0);
  useLayoutEffect(() => {
    const el = ref.current;
    if (!el) return;
    setWidth(el.clientWidth);
    const ro = new ResizeObserver(([entry]) => setWidth(entry.contentRect.width));
    ro.observe(el);
    return () => ro.disconnect();
  }, []);
  return { ref, width };
}

const SERIES = [
  { key: "expectedIn", label: "Masuk", detail: "termin klien yang belum lunas", fill: "var(--chart-in)", swatch: "bg-chart-in" },
  { key: "expectedOut", label: "Keluar", detail: "sisa bayar ke vendor", fill: "var(--chart-out)", swatch: "bg-chart-out" },
] as const;

/**
 * One grouped bar chart of the 8-week cash flow (money in vs out per week, RISET WO Console §7.3/§7.6) and the same
 * numbers as a table. The weeks are the ones /dashboard computes: unpaid installments in, vendor balances out, anything
 * already overdue counted in the first week.
 */
export function CashFlowTab({ weeks, status, onRetry }: { weeks: CashFlowWeek[] | null; status: "loading" | "error" | "ready"; onRetry: () => void }) {
  if (status === "error" && !weeks) return <LoadError message="Arus kas belum bisa dimuat. Periksa koneksi Anda, lalu coba lagi." onRetry={onRetry} />;
  if (!weeks) return <Skeleton className="h-80 rounded-lg" />;

  const totalIn = weeks.reduce((n, w) => n + w.expectedIn, 0);
  const totalOut = weeks.reduce((n, w) => n + w.expectedOut, 0);

  if (totalIn === 0 && totalOut === 0) {
    return (
      <div className="rounded-lg border border-border bg-card">
        <EmptyState
          title="Belum ada uang yang dijadwalkan 8 minggu ke depan"
          description="Arus kas dihitung dari termin klien yang belum lunas dan tanggal jatuh tempo vendor. Isi keduanya di tab Keuangan setiap proyek."
        />
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-4">
      <section aria-labelledby="cashflow-title" className="rounded-lg border border-border bg-card px-5 py-4">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div>
            <h2 id="cashflow-title" className="text-[16px] font-semibold text-foreground">Arus kas 8 minggu ke depan</h2>
            <p className="mt-0.5 max-w-[60ch] text-[13px] text-muted-foreground">
              Masuk: {SERIES[0].detail}. Keluar: {SERIES[1].detail}. Yang sudah lewat jatuh tempo dihitung di minggu ini.
            </p>
          </div>
          <ul className="flex flex-wrap gap-x-5 gap-y-1 text-[13px]" aria-label="Keterangan grafik">
            {SERIES.map((s) => (
              <li key={s.key} className="flex items-center gap-2">
                <span aria-hidden className={`size-2.5 rounded-sm ${s.swatch}`} />
                <span className="text-muted-foreground">{s.label}</span>
                <span className="font-semibold tabular-nums text-foreground">{formatCurrency(s.key === "expectedIn" ? totalIn : totalOut)}</span>
              </li>
            ))}
          </ul>
        </div>
        <Chart weeks={weeks} totalIn={totalIn} totalOut={totalOut} />
      </section>

      <div className="overflow-x-auto rounded-lg border border-border bg-card">
        <table className="w-full border-collapse text-left text-sm">
          <caption className="sr-only">Arus kas per minggu</caption>
          <thead>
            <tr className="h-10 text-[12.5px] text-muted-foreground shadow-[0_1px_0_var(--border)]">
              <th scope="col" className="px-4 font-semibold">Minggu</th>
              <th scope="col" className="px-4 text-right font-semibold">Masuk</th>
              <th scope="col" className="px-4 text-right font-semibold">Keluar</th>
              <th scope="col" className="hidden px-4 text-right font-semibold sm:table-cell">Selisih</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-border">
            {weeks.map((w, i) => (
              <tr key={w.weekStart} className="h-11">
                <th scope="row" className="px-4 font-medium whitespace-nowrap text-foreground">
                  {weekRange(w.weekStart)}
                  {i === 0 && <span className="ml-1.5 text-[12.5px] font-normal text-muted-foreground">(minggu ini)</span>}
                </th>
                <td className="px-4 text-right whitespace-nowrap tabular-nums"><Amount value={w.expectedIn} /></td>
                <td className="px-4 text-right whitespace-nowrap tabular-nums"><Amount value={w.expectedOut} /></td>
                <td className="hidden px-4 text-right font-medium whitespace-nowrap tabular-nums sm:table-cell"><Amount value={w.expectedIn - w.expectedOut} /></td>
              </tr>
            ))}
          </tbody>
          <tfoot>
            <tr className="h-11 border-t border-border font-semibold">
              <th scope="row" className="px-4 text-foreground">Total</th>
              <td className="px-4 text-right whitespace-nowrap tabular-nums"><Amount value={totalIn} /></td>
              <td className="px-4 text-right whitespace-nowrap tabular-nums"><Amount value={totalOut} /></td>
              <td className="hidden px-4 text-right whitespace-nowrap tabular-nums sm:table-cell"><Amount value={totalIn - totalOut} /></td>
            </tr>
          </tfoot>
        </table>
      </div>
    </div>
  );
}

function Chart({ weeks, totalIn, totalOut }: { weeks: CashFlowWeek[]; totalIn: number; totalOut: number }) {
  const { ref, width } = useWidth<HTMLDivElement>();
  const [hover, setHover] = useState<number | null>(null);
  const height = 240;
  const margin = { top: 12, right: 4, bottom: 28, left: 60 };
  const plotW = Math.max(0, width - margin.left - margin.right);
  const plotH = height - margin.top - margin.bottom;
  const { top, step } = niceScale(Math.max(...weeks.flatMap((w) => [w.expectedIn, w.expectedOut])));
  const y = (v: number) => margin.top + plotH - (v / top) * plotH;
  const band = plotW / weeks.length;
  const barW = Math.max(4, Math.min(24, (band * 0.62 - 2) / 2));
  const ticks = Array.from({ length: 5 }, (_, i) => i * step);
  const summary = `Arus kas 8 minggu: masuk ${formatCurrency(totalIn)}, keluar ${formatCurrency(totalOut)}. Angka per minggu ada di tabel di bawah grafik.`;

  return (
    <div ref={ref} className="relative mt-4">
      {width > 0 && (
        <svg width={width} height={height} role="img" aria-label={summary} className="block overflow-visible">
          {ticks.map((t) => (
            <g key={t}>
              <line x1={margin.left} x2={width - margin.right} y1={y(t)} y2={y(t)} stroke="var(--border)" strokeWidth={1} shapeRendering="crispEdges" />
              <text x={margin.left - 8} y={y(t)} dy="0.32em" textAnchor="end" className="fill-muted-foreground text-[11px] tabular-nums">
                {t === 0 ? "0" : formatCurrencyCompact(t)}
              </text>
            </g>
          ))}
          {weeks.map((w, i) => {
            const cx = margin.left + band * i + band / 2;
            const values = [w.expectedIn, w.expectedOut];
            return (
              <g key={w.weekStart} onMouseEnter={() => setHover(i)} onMouseLeave={() => setHover((h) => (h === i ? null : h))}>
                {/* A hit target as tall as the plot, wider than the bars. */}
                <rect x={margin.left + band * i} y={margin.top} width={band} height={plotH} fill={hover === i ? "var(--muted)" : "transparent"} opacity={0.6} />
                {values.map((v, j) => {
                  const h = Math.max(0, y(0) - y(v));
                  const x = cx - barW - 1 + j * (barW + 2);
                  return h > 0 ? <path key={j} d={barPath(x, y(v), barW, h)} fill={SERIES[j].fill} /> : null;
                })}
                {/* On a phone the weeks are too narrow for eight dates: every other one is written. */}
                {(band >= 48 || i % 2 === 0) && (
                  <text x={cx} y={height - 8} textAnchor="middle" className="fill-muted-foreground text-[11px]">
                    {shortDay(w.weekStart)}
                  </text>
                )}
              </g>
            );
          })}
        </svg>
      )}
      {hover !== null && width > 0 && (
        <div
          aria-hidden
          // Beside the hovered week (right of it in the left half, left of it in the right half), never over the axis.
          className={`pointer-events-none absolute top-6 z-10 w-max rounded-md border border-border bg-popover px-3 py-2 text-[12.5px] shadow-sm ${
            hover < weeks.length / 2 ? "" : "-translate-x-full"
          }`}
          style={{ left: margin.left + band * hover + (hover < weeks.length / 2 ? band - 4 : 4) }}
        >
          <p className="font-semibold text-foreground">Minggu {weekRange(weeks[hover].weekStart)}</p>
          {SERIES.map((s) => (
            <p key={s.key} className="mt-1 flex items-center gap-2 text-muted-foreground">
              <span className={`size-2 rounded-sm ${s.swatch}`} />
              {s.label}
              <span className="ml-auto pl-3 font-medium tabular-nums text-foreground">{formatCurrency(weeks[hover][s.key])}</span>
            </p>
          ))}
        </div>
      )}
    </div>
  );
}
