import { Card, CardHeader, CardContent } from "@/shared/components/ui/Card";
import type { RevenueTrendPoint } from "@/modules/dashboard/types";
import { formatCurrency, formatCurrencyCompact } from "@/shared/lib/formatters";
import { cn } from "@/shared/lib/cn";

function formatBarLabel(amount: number): string {
  return formatCurrencyCompact(amount).replace(/^Rp\s*/, "");
}

export function RevenueTrendChart({ points }: { points: RevenueTrendPoint[] }) {
  const total = points.reduce((sum, p) => sum + p.total, 0);
  const max = Math.max(1, ...points.map((p) => p.total));

  return (
    <Card>
      <CardHeader title="Omzet 12 Bulan Terakhir" subtitle="Total nilai kontrak project berdasarkan bulan pelaksanaan acara." />
      <CardContent>
        <div className="mb-5 flex items-baseline gap-2">
          <span className="text-[28px] font-bold leading-none tabular-nums text-text-primary">{formatCurrencyCompact(total)}</span>
          <span className="text-[13px] text-text-secondary">total omzet dalam 12 bulan terakhir</span>
        </div>

        <div className="flex h-36 items-end gap-1.5 sm:gap-2.5">
          {points.map((p, i) => {
            const isCurrent = i === points.length - 1;
            const barHeightPercent = p.total === 0 ? 0 : Math.max(10, (p.total / max) * 100);

            return (
              <div key={p.key} className="flex min-w-0 flex-1 flex-col items-center gap-1.5">
                <span
                  className={cn(
                    "truncate text-[10.5px] font-semibold tabular-nums",
                    p.total > 0 ? "text-text-primary" : "text-text-secondary/40"
                  )}
                >
                  {p.total > 0 ? formatBarLabel(p.total) : "0"}
                </span>
                <div className="flex h-28 w-full items-end justify-center">
                  {p.total === 0 ? (
                    <div className="h-[3px] w-full rounded-full bg-border" title={`${p.label}: tidak ada omzet`} />
                  ) : (
                    <div
                      className={cn("w-full rounded-t-sm transition-all", isCurrent ? "bg-navy-900" : "bg-navy-900/30")}
                      style={{ height: `${barHeightPercent}%` }}
                      title={`${p.label}: ${formatCurrency(p.total)}`}
                    />
                  )}
                </div>
                <span className="truncate text-[10.5px] text-text-secondary">{p.label}</span>
              </div>
            );
          })}
        </div>
      </CardContent>
    </Card>
  );
}
