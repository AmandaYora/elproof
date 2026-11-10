import { Card, CardHeader, CardContent } from "@/shared/components/ui/Card";
import { cn } from "@/shared/lib/cn";

export interface TrendPoint {
  key: string;
  label: string;
  value: number;
}

interface TrendBarChartProps {
  title: string;
  subtitle: string;
  points: TrendPoint[];
  /** The big number: the sum over the period. */
  formatTotal: (total: number) => string;
  totalLabel: string;
  /** Short value printed above a bar ("40 jt"). */
  formatBarLabel: (value: number) => string;
  /** Full value, for the table a screen reader reads and the hover title ("Rp40.000.000"). */
  formatValue: (value: number) => string;
  /** Name of the value column in the table, e.g. "Omzet". */
  valueHeader: string;
  /** What the table's first column holds, e.g. "Bulan". */
  periodHeader?: string;
}

/**
 * One bar per period with its value written above it. The bars are decoration for people who see them; what a screen
 * reader gets is the same data as a table, so no value lives only in a colour or a tooltip. Bars grow with a CSS
 * transition, which the global reduced-motion rule removes.
 */
export function TrendBarChart({ title, subtitle, points, formatTotal, totalLabel, formatBarLabel, formatValue, valueHeader, periodHeader = "Periode" }: TrendBarChartProps) {
  const total = points.reduce((sum, p) => sum + p.value, 0);
  const max = Math.max(1, ...points.map((p) => p.value));

  return (
    <Card>
      <CardHeader title={title} subtitle={subtitle} />
      <CardContent>
        <figure className="m-0">
          <div className="mb-5 flex items-baseline gap-2">
            <span className="text-[28px] leading-none font-bold tabular-nums text-foreground">{formatTotal(total)}</span>
            <span className="text-[13px] text-muted-foreground">{totalLabel}</span>
          </div>

          <div aria-hidden className="flex h-36 items-end gap-1 sm:gap-1.5">
            {points.map((p, i) => {
              const current = i === points.length - 1;
              const height = p.value === 0 ? 0 : Math.max(10, (p.value / max) * 100);
              return (
                <div key={p.key} className="flex min-w-0 flex-1 flex-col items-center gap-1.5">
                  <span className="truncate text-[11px] font-semibold tabular-nums text-foreground">{p.value > 0 ? formatBarLabel(p.value) : <span className="text-muted-foreground">0</span>}</span>
                  <div className="flex h-28 w-full items-end justify-center" title={`${p.label}: ${formatValue(p.value)}`}>
                    {p.value === 0 ? (
                      <div className="h-[3px] w-full rounded-full bg-border" />
                    ) : (
                      <div className={cn("w-full rounded-t-sm transition-[height]", current ? "bg-primary" : "bg-primary/35")} style={{ height: `${height}%` }} />
                    )}
                  </div>
                  <span className="truncate text-[11px] text-muted-foreground">{p.label}</span>
                </div>
              );
            })}
          </div>

          <figcaption className="sr-only">{title}</figcaption>
          {/* sr-only does not clip a <table> by itself, so the wrapper carries it. */}
          <div className="sr-only">
          <table>
            <caption>{`${title}: ${subtitle}`}</caption>
            <thead>
              <tr>
                <th scope="col">{periodHeader}</th>
                <th scope="col">{valueHeader}</th>
              </tr>
            </thead>
            <tbody>
              {points.map((p) => (
                <tr key={p.key}>
                  <th scope="row">{p.label}</th>
                  <td>{formatValue(p.value)}</td>
                </tr>
              ))}
            </tbody>
          </table>
          </div>
        </figure>
      </CardContent>
    </Card>
  );
}
