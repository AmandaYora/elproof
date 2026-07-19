import { useEffect } from "react";
import { useOutletContext } from "react-router-dom";
import { Link } from "react-router-dom";
import { Check, HeartHandshake } from "lucide-react";
import { useProjectStore } from "@/modules/projects/stores/useProjectStore";
import { formatDate } from "@/shared/lib/formatters";
import { CLIENT_CONDITION_COPY } from "@/modules/client-portal/lib/condition";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import type { ClientPortalContext } from "@/modules/client-portal/layouts/ClientPortalLayout";

export default function RingkasanTabPage() {
  const { projectId } = useOutletContext<ClientPortalContext>();
  const project = useProjectStore((s) => s.currentProject);
  const milestones = useProjectStore((s) => s.milestones);
  const vendorEngagements = useProjectStore((s) => s.vendorEngagements);
  const issues = useProjectStore((s) => s.issues);
  const fetchMilestones = useProjectStore((s) => s.fetchMilestones);
  const fetchVendorSection = useProjectStore((s) => s.fetchVendorSection);
  const fetchIssues = useProjectStore((s) => s.fetchIssues);

  useEffect(() => {
    void fetchMilestones(projectId);
    void fetchVendorSection(projectId);
    void fetchIssues(projectId);
  }, [projectId, fetchMilestones, fetchVendorSection, fetchIssues]);

  if (!project || !project.progress) return null;

  const relevantMilestones = milestones.filter((m) => m.status !== "Cancelled");
  const vendorCount = vendorEngagements.length;
  const openIssues = issues.filter((i) => i.status !== "Resolved" && i.status !== "Closed");
  const condition = CLIENT_CONDITION_COPY[project.progress.condition];
  const completedCount = relevantMilestones.filter((m) => m.status === "Completed").length;

  return (
    <div className="flex flex-col gap-8 sm:gap-10">
      <section className="relative overflow-hidden rounded-3xl border border-border bg-white shadow-xl shadow-navy-900/5">
        <div className="absolute inset-0 bg-gradient-to-br from-blue-50/50 to-transparent"></div>
        <div className="relative p-6 sm:p-8">
          <div className="flex flex-col gap-6 md:flex-row md:items-center md:justify-between">
            <div className="flex-1">
              <p className="text-[13px] font-bold uppercase tracking-wider text-navy-600 sm:text-[13.5px]">Status Persiapan Pernikahan</p>
              <h2 className="mt-2 text-[28px] font-bold tracking-tight text-navy-950 sm:text-[32px]">{condition.label}</h2>
              <p className="mt-3 text-[14px] leading-relaxed text-text-secondary sm:text-[15px] max-w-xl">
                {vendorCount} vendor sedang mempersiapkan hari bahagia Anda. Setiap progress yang tercatat di sini didasarkan pada tahapan yang benar-benar telah diselesaikan oleh tim kami — bukan perkiraan.
              </p>
            </div>
            
            <div className="flex shrink-0 flex-col items-center justify-center rounded-2xl bg-white p-5 shadow-sm border border-border/60 min-w-[160px]">
              <span className="text-[36px] font-bold text-navy-900 leading-none">{completedCount}</span>
              <span className="mt-1 text-[13px] font-medium text-text-secondary border-t border-border/50 pt-2 w-full text-center">dari {relevantMilestones.length} tahapan selesai</span>
            </div>
          </div>

          {openIssues.length > 0 && (
            <Link
              to={ROUTE_PATHS.portal("kendala")}
              className="group mt-6 flex flex-col items-start gap-3 rounded-2xl border border-warning/20 bg-warning-soft/50 p-4 transition-all hover:bg-warning-soft hover:shadow-md sm:flex-row sm:items-center sm:justify-between sm:px-5"
            >
              <div className="flex items-center gap-3">
                <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-warning/20 text-warning-strong">
                  <span className="text-[16px] font-bold">{openIssues.length}</span>
                </div>
                <p className="text-[13.5px] font-medium text-warning-strong sm:text-[14px]">
                  Terdapat kendala aktif yang sedang kami tangani.
                </p>
              </div>
              <span className="inline-flex items-center gap-1.5 text-[13px] font-bold text-warning-strong group-hover:underline underline-offset-4">
                Lihat detail <HeartHandshake className="h-4 w-4" />
              </span>
            </Link>
          )}
        </div>
      </section>

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
    </div>
  );
}
