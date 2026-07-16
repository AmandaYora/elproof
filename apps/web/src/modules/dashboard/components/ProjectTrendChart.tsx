import { Card, CardHeader, CardContent } from "@/shared/components/ui/Card";
import type { ProjectTrendPoint } from "@/modules/dashboard/types";
import { cn } from "@/shared/lib/cn";

export function ProjectTrendChart({ points }: { points: ProjectTrendPoint[] }) {
  const total = points.reduce((sum, p) => sum + p.count, 0);
  const max = Math.max(1, ...points.map((p) => p.count));

  return (
    <Card>
      <CardHeader title="Trend Project Baru" subtitle="Jumlah project baru dibuat per bulan, berdasarkan tanggal project dibuat." />
      <CardContent>
        <div className="mb-5 flex items-baseline gap-2">
          <span className="text-[28px] font-bold leading-none tabular-nums text-text-primary">{total}</span>
          <span className="text-[13px] text-text-secondary">project baru dalam 12 bulan terakhir</span>
        </div>

        <div className="flex h-36 items-end gap-1.5 sm:gap-2.5">
          {points.map((p, i) => {
            const isCurrent = i === points.length - 1;
            const barHeightPercent = p.count === 0 ? 0 : Math.max(10, (p.count / max) * 100);

            return (
              <div key={p.key} className="flex min-w-0 flex-1 flex-col items-center gap-1.5">
                <span className={cn("text-[11px] font-semibold tabular-nums", p.count > 0 ? "text-text-primary" : "text-text-secondary/40")}>
                  {p.count}
                </span>
                <div className="flex h-28 w-full items-end justify-center">
                  {p.count === 0 ? (
                    <div className="h-[3px] w-full rounded-full bg-border" title={`${p.label}: tidak ada project baru`} />
                  ) : (
                    <div
                      className={cn("w-full rounded-t-sm transition-all", isCurrent ? "bg-navy-900" : "bg-navy-900/30")}
                      style={{ height: `${barHeightPercent}%` }}
                      title={`${p.label}: ${p.count} project baru`}
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
