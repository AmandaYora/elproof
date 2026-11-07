import { Check } from "lucide-react";
import { cn } from "@/shared/lib/cn";
import { STAGE_LABELS, stageIndex } from "@/modules/client-portal/lib/stage";
import type { ProjectStatus } from "@/modules/projects/types";

/** The four steps as a vertical order tracker: done, current, still ahead. */
export function ProgressStages({ status, daysToEvent, percent }: { status: ProjectStatus; daysToEvent: number; percent?: number }) {
  const current = stageIndex(status, daysToEvent);
  if (current === null) return null;
  return (
    <section aria-labelledby="stages-heading" className="rounded-2xl border border-border bg-white p-5 sm:p-6">
      <div className="flex items-baseline justify-between gap-3">
        <h2 id="stages-heading" className="text-[15px] font-semibold text-text-primary">Progres persiapan</h2>
        {percent !== undefined && <span className="text-[13px] font-semibold tabular-nums text-navy-900">{percent}% selesai</span>}
      </div>
      <ol className="mt-4 flex flex-col">
        {STAGE_LABELS.map((label, i) => {
          const done = i < current;
          const active = i === current;
          return (
            <li key={label} className="relative flex gap-3 pb-5 last:pb-0" aria-current={active ? "step" : undefined}>
              {i < STAGE_LABELS.length - 1 && (
                <span aria-hidden className={cn("absolute left-[11px] top-6 h-[calc(100%-1.5rem)] w-0.5", done ? "bg-navy-900" : "bg-border")} />
              )}
              <span
                aria-hidden
                className={cn(
                  "z-10 flex h-6 w-6 shrink-0 items-center justify-center rounded-full border-2",
                  done && "border-navy-900 bg-navy-900 text-white",
                  active && "border-navy-900 bg-white",
                  !done && !active && "border-border bg-white",
                )}
              >
                {done ? <Check className="h-3.5 w-3.5" /> : active ? <span className="h-2 w-2 rounded-full bg-navy-900" /> : null}
              </span>
              <span className={cn("pt-0.5 text-[14px]", active ? "font-semibold text-text-primary" : done ? "text-text-primary" : "text-text-secondary")}>
                {label}
                <span className="sr-only">{done ? " — sudah" : active ? " — sedang berjalan" : " — belum"}</span>
              </span>
            </li>
          );
        })}
      </ol>
    </section>
  );
}
