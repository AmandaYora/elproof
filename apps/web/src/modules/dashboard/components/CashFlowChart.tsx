import { Link } from "react-router-dom";
import { Wallet } from "lucide-react";
import { Card, CardContent, CardHeader } from "@/shared/components/ui/Card";
import { formatCurrency, formatCurrencyCompact, formatDate } from "@/shared/lib/formatters";
import { cn } from "@/shared/lib/cn";
import type { CashFlowWeek } from "@/modules/dashboard/types";

const PLOT_PX = 168;
// Room above and below the bars for the two peak labels.
const LABEL_PX = 20;

function shortDate(iso: string): string {
  return formatDate(iso).replace(/ \d{4}$/, "");
}

function weekEnd(weekStart: string): string {
  const [y, m, d] = weekStart.split("-").map(Number);
  const end = new Date(Date.UTC(y, m - 1, d + 6));
  return end.toISOString().slice(0, 10);
}

function weekRange(weekStart: string): string {
  return `${shortDate(weekStart)} – ${shortDate(weekEnd(weekStart))}`;
}

/**
 * Arus kas 8 minggu: per week, what clients still owe (above the line, navy) and what the WO still owes vendors
 * (below the line, grey), one scale for both. Only the largest week of each side carries its amount; hovering a week
 * shows both amounts, and a screen reader gets the same numbers as a table. With nothing scheduled the chart is
 * replaced by an empty state that says how to fill it, never a row of zero bars.
 */
export function CashFlowChart({ weeks, emptyAction }: { weeks: CashFlowWeek[]; emptyAction: { label: string; to: string } }) {
  const totalIn = weeks.reduce((n, w) => n + w.expectedIn, 0);
  const totalOut = weeks.reduce((n, w) => n + w.expectedOut, 0);
  const empty = totalIn === 0 && totalOut === 0;

  const maxIn = Math.max(0, ...weeks.map((w) => w.expectedIn));
  const maxOut = Math.max(0, ...weeks.map((w) => w.expectedOut));
  const scale = maxIn + maxOut || 1;
  const upPx = Math.round((maxIn / scale) * PLOT_PX);
  const downPx = PLOT_PX - upPx;
  const peakIn = weeks.findIndex((w) => w.expectedIn === maxIn && maxIn > 0);
  const peakOut = weeks.findIndex((w) => w.expectedOut === maxOut && maxOut > 0);
  const bar = (value: number) => (value > 0 ? Math.max(3, (value / scale) * PLOT_PX) : 0);

  return (
    <Card className="h-full">
      <CardHeader
        title="Arus kas 8 minggu ke depan"
        action={
          !empty && (
            <dl className="flex items-center gap-4 text-[13px]">
              <div className="flex items-center gap-1.5">
                <span aria-hidden className="size-2.5 rounded-[3px] bg-primary" />
                <dt className="text-muted-foreground">Masuk</dt>
                <dd className="font-semibold tabular-nums text-foreground">{formatCurrencyCompact(totalIn)}</dd>
              </div>
              <div className="flex items-center gap-1.5">
                <span aria-hidden className="size-2.5 rounded-[3px] bg-input" />
                <dt className="text-muted-foreground">Keluar</dt>
                <dd className="font-semibold tabular-nums text-foreground">{formatCurrencyCompact(totalOut)}</dd>
              </div>
            </dl>
          )
        }
      />
      <CardContent>
        {empty ? (
          <div className="flex flex-col items-center justify-center gap-2 py-10 text-center" style={{ minHeight: PLOT_PX + 2 * LABEL_PX + 24 }}>
            <Wallet aria-hidden className="size-7 text-muted-foreground" />
            <p className="text-[14px] font-semibold text-foreground">Belum ada uang yang dijadwalkan dalam 8 minggu ke depan</p>
            <p className="max-w-sm text-[13px] text-muted-foreground">
              Grafik ini terisi dari termin klien dan tagihan vendor yang belum lunas. Atur jadwal termin di tab Keuangan setiap proyek.
            </p>
            <Link
              to={emptyAction.to}
              className="mt-2 inline-flex h-8 items-center rounded-md border border-border bg-card px-3 text-[13px] font-semibold text-foreground hover:bg-muted focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-solid focus-visible:outline-ring"
            >
              {emptyAction.label}
            </Link>
          </div>
        ) : (
          <figure className="m-0">
            <div aria-hidden className="flex gap-1 sm:gap-2">
              {weeks.map((w, i) => (
                <div key={w.weekStart} className="group relative flex min-w-0 flex-1 flex-col items-center">
                  {/* Hover band + tooltip: both amounts for this week. */}
                  <div className="pointer-events-none absolute inset-x-0 top-0 rounded-md bg-muted opacity-0 transition-opacity group-hover:opacity-100" style={{ height: PLOT_PX + 2 * LABEL_PX + 2 }} />
                  <div
                    className={cn(
                      "pointer-events-none absolute -top-2 z-10 w-max -translate-y-full rounded-md border border-border bg-popover px-2.5 py-1.5 text-[12px] opacity-0 shadow-md transition-opacity group-hover:opacity-100",
                      i < 2 ? "left-0" : i > weeks.length - 3 ? "right-0" : "left-1/2 -translate-x-1/2",
                    )}
                  >
                    <p className="font-semibold text-foreground">{weekRange(w.weekStart)}</p>
                    <p className="tabular-nums text-muted-foreground">Masuk <span className="text-foreground">{formatCurrency(w.expectedIn)}</span></p>
                    <p className="tabular-nums text-muted-foreground">Keluar <span className="text-foreground">{formatCurrency(w.expectedOut)}</span></p>
                  </div>

                  <div className="relative flex w-full flex-col items-center justify-end" style={{ height: upPx + LABEL_PX }}>
                    {i === peakIn && <span className="mb-1 text-[12px] font-semibold tabular-nums whitespace-nowrap text-foreground">{formatCurrencyCompact(w.expectedIn)}</span>}
                    <div className="w-full max-w-6 rounded-t-[4px] bg-primary" style={{ height: bar(w.expectedIn) }} />
                  </div>
                  <div className="relative h-0.5 w-full bg-border" />
                  <div className="relative flex w-full flex-col items-center justify-start" style={{ height: downPx + LABEL_PX }}>
                    <div className="w-full max-w-6 rounded-b-[4px] bg-input" style={{ height: bar(w.expectedOut) }} />
                    {i === peakOut && <span className="mt-1 text-[12px] font-semibold tabular-nums whitespace-nowrap text-foreground">{formatCurrencyCompact(w.expectedOut)}</span>}
                  </div>
                  <span className={cn("relative mt-2 truncate text-[12px] text-muted-foreground", i % 2 === 1 && "max-sm:invisible")}>
                    {i === 0 ? "Minggu ini" : shortDate(w.weekStart)}
                  </span>
                </div>
              ))}
            </div>
            <figcaption className="mt-3 text-[12px] text-muted-foreground">
              Termin klien dan tagihan vendor yang belum lunas, menurut minggu jatuh tempo. Yang sudah lewat jatuh tempo dihitung di minggu ini.
            </figcaption>
            <div className="sr-only">
              <table>
                <caption>Arus kas 8 minggu ke depan</caption>
                <thead>
                  <tr>
                    <th scope="col">Minggu</th>
                    <th scope="col">Masuk</th>
                    <th scope="col">Keluar</th>
                  </tr>
                </thead>
                <tbody>
                  {weeks.map((w) => (
                    <tr key={w.weekStart}>
                      <th scope="row">{weekRange(w.weekStart)}</th>
                      <td>{formatCurrency(w.expectedIn)}</td>
                      <td>{formatCurrency(w.expectedOut)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </figure>
        )}
      </CardContent>
    </Card>
  );
}
