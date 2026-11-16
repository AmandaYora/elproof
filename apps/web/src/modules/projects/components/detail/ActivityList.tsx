import type { ActivityLogEntry } from "@/modules/projects/types";
import { formatDateTime } from "@/shared/lib/formatters";
import { activityText } from "@/shared/lib/labels";

/** Who did what, newest first, on a thin line: used by the Riwayat panel and by the full history. */
export function ActivityList({ entries }: { entries: ActivityLogEntry[] }) {
  return (
    <ol className="flex flex-col">
      {entries.map((entry, index) => {
        const isLast = index === entries.length - 1;
        return (
          <li key={entry.id} className="flex gap-3">
            <div aria-hidden className="flex flex-col items-center">
              <span className="mt-1.5 h-2 w-2 shrink-0 rounded-full bg-input" />
              {!isLast && <span className="w-px flex-1 bg-border-light" />}
            </div>
            <div className={isLast ? "min-w-0" : "min-w-0 pb-4"}>
              <p className="text-[13px] text-text-primary">{activityText(entry.description)}</p>
              <p className="mt-0.5 text-[12px] text-text-secondary">
                {entry.actorName || "Sistem"} · {formatDateTime(entry.timestamp)}
              </p>
            </div>
          </li>
        );
      })}
    </ol>
  );
}
