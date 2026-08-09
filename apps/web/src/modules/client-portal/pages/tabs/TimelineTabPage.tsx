import { useEffect } from "react";
import { useOutletContext } from "react-router-dom";
import { Check, HeartHandshake } from "lucide-react";
import { useProjectStore } from "@/modules/projects/stores/useProjectStore";
import { formatDate } from "@/shared/lib/formatters";
import type { ClientPortalContext } from "@/modules/client-portal/layouts/ClientPortalLayout";

// The detailed vertical milestone stepper that used to be the main content
// of the standalone "Ringkasan" tab — split out into its own "Timeline" tab
// as part of PLAN.md's Client Portal restructure (the compact progress
// summary that also lived on that tab now lives in the persistent
// ClientProjectHeaderCard instead, matching WO Console's own header-card /
// separate-Timeline-tab split).
export default function TimelineTabPage() {
  const { projectId } = useOutletContext<ClientPortalContext>();
  const milestones = useProjectStore((s) => s.milestones);
  const fetchMilestones = useProjectStore((s) => s.fetchMilestones);

  useEffect(() => {
    void fetchMilestones(projectId);
  }, [projectId, fetchMilestones]);

  const relevantMilestones = milestones.filter((m) => m.status !== "Cancelled");

  return (
    <section className="rounded-3xl border border-border bg-white p-6 sm:p-8 shadow-sm">
      <div className="mb-8 flex items-center gap-3 border-b border-border pb-5">
        <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-navy-50 text-navy-600">
          <HeartHandshake className="h-5 w-5 shrink-0" />
        </div>
        <div>
          <h2 className="text-[18px] font-bold text-navy-950 sm:text-[20px]">Perjalanan Persiapan</h2>
          <p className="mt-0.5 text-[13px] text-text-secondary sm:text-[14px]">Tercatat secara real-time saat tim menyelesaikan tahapan.</p>
        </div>
      </div>

      <ol className="flex flex-col ml-4 sm:ml-6">
        {relevantMilestones.map((m, idx) => {
          const isLast = idx === relevantMilestones.length - 1;
          const isCompleted = m.status === "Completed";
          const isBlocked = m.status === "Blocked";
          const isInProgress = m.status === "In Progress";

          return (
            <li key={m.id} className="group flex gap-5 sm:gap-6 relative">
              <div className="flex flex-col items-center">
                <span
                  className={
                    isCompleted
                      ? "flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-navy-900 text-white shadow-md shadow-navy-900/20 z-10 transition-transform group-hover:scale-110"
                      : isBlocked
                      ? "flex h-8 w-8 shrink-0 items-center justify-center rounded-full border-[3px] border-danger bg-white text-danger z-10 transition-transform group-hover:scale-110"
                      : isInProgress
                      ? "flex h-8 w-8 shrink-0 items-center justify-center rounded-full border-[3px] border-navy-900 bg-white text-navy-900 z-10 transition-transform group-hover:scale-110"
                      : "flex h-8 w-8 shrink-0 items-center justify-center rounded-full border-2 border-border bg-surface-muted text-border z-10"
                  }
                >
                  {isCompleted && <Check className="h-4 w-4" />}
                  {!isCompleted && !isBlocked && !isInProgress && <div className="h-2 w-2 rounded-full bg-border" />}
                  {isInProgress && <div className="h-2.5 w-2.5 rounded-full bg-navy-900 animate-pulse" />}
                </span>
                {!isLast && <span className="w-[2px] flex-1 bg-gradient-to-b from-border to-border/50 my-1" />}
              </div>
              <div className={isLast ? "pb-4" : "pb-10 sm:pb-12"}>
                <div className="flex flex-col sm:flex-row sm:items-center gap-1.5 sm:gap-3">
                  <p className={isCompleted || isInProgress ? "text-[15px] font-bold text-navy-950 sm:text-[16px]" : "text-[15px] font-medium text-text-secondary sm:text-[16px]"}>
                    {m.name}
                  </p>
                  {isInProgress && (
                    <span className="inline-flex items-center rounded-full bg-blue-50 px-2 py-0.5 text-[11px] font-bold text-blue-600 ring-1 ring-blue-500/20">
                      IN PROGRESS
                    </span>
                  )}
                </div>
                <p className="mt-1.5 text-[13px] text-text-secondary sm:text-[13.5px] bg-surface-muted/50 inline-block px-3 py-1.5 rounded-lg border border-border/50">
                  {isCompleted
                    ? `Diselesaikan pada ${formatDate(m.completedDate)}`
                    : isBlocked
                    ? "Sedang terhambat — tim kami sedang menindaklanjuti"
                    : isInProgress
                    ? "Sedang dikerjakan oleh tim"
                    : `Target: ${formatDate(m.targetDate)}`}
                </p>
              </div>
            </li>
          );
        })}
      </ol>
    </section>
  );
}
