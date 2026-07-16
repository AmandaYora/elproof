import { Card, CardHeader, CardContent } from "@/shared/components/ui/Card";
import { cn } from "@/shared/lib/cn";
import type { TrendPoint } from "@/modules/platform-admin/lib/trend";

interface TrendBarChartProps {
  title: string;
  subtitle: string;
  points: TrendPoint[];
  totalLabel: string;
  formatTotal: (value: number) => string;
  formatBarLabel: (value: number) => string;
  formatTooltip: (label: string, value: number) => string;
  emptyTooltip: (label: string) => string;
}

export function TrendBarChart({
  title,
  subtitle,
  points,
  totalLabel,
  formatTotal,
  formatBarLabel,
  formatTooltip,
  emptyTooltip,
}: TrendBarChartProps) {
  const total = points.reduce((sum, p) => sum + p.value, 0);
  const max = Math.max(1, ...points.map((p) => p.value));

  return (
    <Card>
      <CardHeader title={title} subtitle={subtitle} />
      <CardContent>
        <div className="mb-5 flex items-baseline gap-2">
          <span className="text-[28px] font-bold leading-none tabular-nums text-text-primary">{formatTotal(total)}</span>
          <span className="text-[13px] text-text-secondary">{totalLabel}</span>
        </div>

        <div className="flex h-36 items-end gap-1 sm:gap-1.5">
          {points.map((p, i) => {
            const isCurrent = i === points.length - 1;
            const barHeightPercent = p.value === 0 ? 0 : Math.max(10, (p.value / max) * 100);

            return (
              <div key={p.key} className="flex min-w-0 flex-1 flex-col items-center gap-1.5">
                <span
                  className={cn(
                    "truncate text-[10px] font-semibold tabular-nums",
                    p.value > 0 ? "text-text-primary" : "text-text-secondary/40"
                  )}
                >
                  {p.value > 0 ? formatBarLabel(p.value) : ""}
                </span>
                <div className="flex h-28 w-full items-end justify-center">
                  {p.value === 0 ? (
                    <div className="h-[3px] w-full rounded-full bg-border" title={emptyTooltip(p.label)} />
                  ) : (
                    <div
                      className={cn("w-full rounded-t-sm transition-all", isCurrent ? "bg-navy-900" : "bg-navy-900/30")}
                      style={{ height: `${barHeightPercent}%` }}
                      title={formatTooltip(p.label, p.value)}
                    />
                  )}
                </div>
                <span className="truncate text-[10px] text-text-secondary">{p.label}</span>
              </div>
            );
          })}
        </div>
      </CardContent>
    </Card>
  );
}
