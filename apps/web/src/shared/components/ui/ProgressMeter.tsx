import { cn } from "@/shared/lib/cn";

export type ProgressMeterStatus = "Not Started" | "In Progress" | "Completed" | "Blocked" | "Cancelled";

interface ProgressMeterProps {
  percent: number;
  segments: { status: ProgressMeterStatus }[];
  caption?: string;
  className?: string;
}

export function ProgressMeter({ percent, segments, caption, className }: ProgressMeterProps) {
  const relevant = segments.filter((s) => s.status !== "Cancelled");
  const total = relevant.length;
  const showNotches = total > 1 && total <= 20;
  const blockedPositions = relevant
    .map((s, i) => ({ status: s.status, position: total === 0 ? 0 : ((i + 0.5) / total) * 100 }))
    .filter((s) => s.status === "Blocked");

  return (
    <div className={cn("flex flex-col gap-2", className)}>
      <div className="relative h-2.5 w-full overflow-visible rounded-full bg-surface-muted">
        <div
          className="h-full rounded-full bg-gradient-to-r from-navy-800 to-navy-900 transition-[width] duration-500 ease-out"
          style={{ width: `${Math.min(100, Math.max(0, percent))}%` }}
        />
        {showNotches &&
          Array.from({ length: total - 1 }, (_, i) => ((i + 1) / total) * 100).map((pos) => (
            <span
              key={pos}
              className="absolute top-0 h-full w-[1.5px] bg-background"
              style={{ left: `${pos}%` }}
            />
          ))}
        {blockedPositions.map((b, i) => (
          <span
            key={i}
            className="absolute -top-0.5 h-3.5 w-3.5 -translate-x-1/2 rounded-full bg-danger ring-2 ring-background"
            style={{ left: `${b.position}%` }}
            title="Milestone terhambat"
          />
        ))}
      </div>
      {caption && <p className="text-[12.5px] text-text-secondary">{caption}</p>}
    </div>
  );
}
