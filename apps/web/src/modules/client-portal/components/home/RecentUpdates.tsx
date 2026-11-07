import { CheckCircle2 } from "lucide-react";
import { formatDate } from "@/shared/lib/formatters";
import type { ProjectMilestone } from "@/modules/projects/types";

/** The latest finished steps, so the client sees things moving. */
export function RecentUpdates({ milestones }: { milestones: ProjectMilestone[] }) {
  const done = milestones
    .filter((m) => m.status === "Completed" && m.completedDate)
    .sort((a, b) => (b.completedDate ?? "").localeCompare(a.completedDate ?? ""))
    .slice(0, 3);
  if (done.length === 0) return null;
  return (
    <section aria-labelledby="news-heading" className="rounded-2xl border border-border bg-white p-5 sm:p-6">
      <h2 id="news-heading" className="text-[15px] font-semibold text-text-primary">Kabar terbaru</h2>
      <ul className="mt-3 flex flex-col gap-3">
        {done.map((m) => (
          <li key={m.id} className="flex items-start gap-3">
            <CheckCircle2 className="mt-0.5 h-4 w-4 shrink-0 text-success" aria-hidden />
            <span className="text-[14px] text-text-primary">
              {m.name}
              <span className="block text-[12.5px] text-text-secondary">Selesai {formatDate(m.completedDate)}</span>
            </span>
          </li>
        ))}
      </ul>
    </section>
  );
}
