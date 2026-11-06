import { cn } from "@/shared/lib/cn";

/**
 * Placeholder shape while data loads (PLAN.md Fase 3b): keeps the layout
 * steady instead of a "Memuat..." line that jumps when content arrives.
 */
export function Skeleton({ className }: { className?: string }) {
  return <div aria-hidden className={cn("rounded-md bg-surface-muted motion-safe:animate-pulse", className)} />;
}

/** A stack of list rows; announces loading once for assistive tech. */
export function SkeletonRows({ rows = 4, label = "Memuat data" }: { rows?: number; label?: string }) {
  return (
    <div role="status" aria-label={label} className="flex flex-col gap-3">
      {Array.from({ length: rows }, (_, i) => (
        <div key={i} className="flex items-center gap-3">
          <Skeleton className="h-9 w-9 rounded-full" />
          <div className="flex flex-1 flex-col gap-1.5">
            <Skeleton className="h-3.5 w-2/5" />
            <Skeleton className="h-3 w-3/5" />
          </div>
        </div>
      ))}
    </div>
  );
}
