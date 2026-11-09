import { formatCurrencyCompact, formatDate } from "@/shared/lib/formatters";
import type { CashFlowWeek } from "@/modules/dashboard/types";

function weekLabel(weekStart: string): string {
  const [y, m, d] = weekStart.split("-").map(Number);
  const end = new Date(y, m - 1, d + 6);
  const endISO = `${end.getFullYear()}-${String(end.getMonth() + 1).padStart(2, "0")}-${String(end.getDate()).padStart(2, "0")}`;
  return `${formatDate(weekStart).replace(` ${y}`, "")} – ${formatDate(endISO).replace(` ${end.getFullYear()}`, "")}`;
}

/**
 * Uang yang diharapkan masuk dan keluar dalam 8 minggu ke depan: termin klien yang belum lunas dan
 * kewajiban ke vendor. Setiap baris menyebut angkanya dengan teks, batangnya hanya penguat.
 */
export function CashFlowCard({ weeks }: { weeks: CashFlowWeek[] }) {
  const max = Math.max(1, ...weeks.flatMap((w) => [w.expectedIn, w.expectedOut]));
  const totalIn = weeks.reduce((n, w) => n + w.expectedIn, 0);
  const totalOut = weeks.reduce((n, w) => n + w.expectedOut, 0);
  return (
    <section aria-labelledby="cashflow-heading" className="rounded-lg border border-border bg-surface px-5 py-4">
      <div className="flex flex-wrap items-baseline justify-between gap-2">
        <div>
          <h2 id="cashflow-heading" className="text-[15px] font-semibold text-text-primary">Arus kas 8 minggu ke depan</h2>
          <p className="text-[12.5px] text-text-secondary">Termin klien yang belum lunas dan tagihan vendor, per minggu jatuh tempo.</p>
        </div>
        <p className="text-[12.5px] tabular-nums text-text-secondary">
          Masuk <strong className="text-success-text">{formatCurrencyCompact(totalIn)}</strong> · Keluar <strong className="text-text-primary">{formatCurrencyCompact(totalOut)}</strong>
        </p>
      </div>
      <ul className="mt-3 flex flex-col gap-3">
        {weeks.map((w) => (
          <li key={w.weekStart} className="grid grid-cols-1 gap-1 text-[12.5px] sm:grid-cols-[8rem_minmax(0,1fr)] sm:items-center sm:gap-3">
            <span className="font-medium text-text-secondary">{weekLabel(w.weekStart)}</span>
            <span className="flex flex-col gap-1">
              <Bar label="Masuk" value={w.expectedIn} max={max} tone="bg-success" />
              <Bar label="Keluar" value={w.expectedOut} max={max} tone="bg-navy-900" />
            </span>
          </li>
        ))}
      </ul>
    </section>
  );
}

/** One labelled bar; the amount is always written out, the bar only helps comparing weeks. */
function Bar({ label, value, max, tone }: { label: string; value: number; max: number; tone: string }) {
  return (
    <span className="flex items-center gap-2">
      <span className="w-11 shrink-0 text-text-secondary">{label}</span>
      <span aria-hidden className="relative h-2 flex-1 rounded-full bg-surface-muted">
        <span className={`absolute inset-y-0 left-0 rounded-full ${tone}`} style={{ width: `${(value / max) * 100}%`, minWidth: value > 0 ? 4 : 0 }} />
      </span>
      <span className="w-16 shrink-0 text-right tabular-nums text-text-primary">{formatCurrencyCompact(value)}</span>
    </span>
  );
}
