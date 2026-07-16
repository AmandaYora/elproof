import { useEffect } from "react";
import { useOutletContext } from "react-router-dom";
import { Link } from "react-router-dom";
import { Check, HeartHandshake } from "lucide-react";
import { Badge } from "@/shared/components/ui/Badge";
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
    <div className="flex flex-col gap-6 sm:gap-8">
      <section className="rounded-xl border border-border bg-surface p-4 sm:p-7">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div>
            <p className="text-[12.5px] font-medium text-text-secondary sm:text-[13px]">Kondisi persiapan pernikahan Anda saat ini</p>
            <p className="mt-1 text-base font-bold text-text-primary sm:text-lg">{condition.label}</p>
          </div>
          <Badge tone={condition.tone}>{completedCount}/{relevantMilestones.length} tahapan selesai</Badge>
        </div>
        <p className="mt-4 text-[13px] leading-relaxed text-text-secondary sm:text-[13.5px]">
          {vendorCount} vendor sedang mempersiapkan hari bahagia Anda. Setiap progress yang tercatat di sini
          didasarkan pada tahapan yang benar-benar telah diselesaikan oleh tim kami — bukan perkiraan.
        </p>
        {openIssues.length > 0 && (
          <Link
            to={ROUTE_PATHS.portal("kendala")}
            className="mt-4 flex flex-col items-start gap-1.5 rounded-lg border border-warning/30 bg-warning-soft px-4 py-3 text-[13px] font-medium text-warning transition-colors hover:bg-warning-soft/70 sm:flex-row sm:items-center sm:justify-between sm:gap-3 sm:text-[13.5px]"
          >
            <span>
              Ada {openIssues.length} kendala yang sedang kami tangani — kami ingin Anda mengetahuinya juga.
            </span>
            <span className="whitespace-nowrap underline underline-offset-2">Lihat detail</span>
          </Link>
        )}
      </section>

      <section>
        <div className="mb-4 flex items-center gap-2 sm:mb-5">
          <HeartHandshake className="h-5 w-5 shrink-0 text-navy-900" />
          <h2 className="text-base font-bold text-text-primary sm:text-lg">Perjalanan Persiapan Pernikahan Anda</h2>
        </div>
        <p className="mb-5 max-w-2xl text-[13px] leading-relaxed text-text-secondary sm:mb-6 sm:text-[13.5px]">
          Setiap langkah di bawah ini tercatat oleh tim WO kami saat benar-benar selesai dikerjakan — bukan
          jadwal atau target yang belum tentu terjadi.
        </p>

        <ol className="flex flex-col">
          {relevantMilestones.map((m, idx) => {
            const isLast = idx === relevantMilestones.length - 1;
            const isCompleted = m.status === "Completed";
            const isBlocked = m.status === "Blocked";
            const isInProgress = m.status === "In Progress";

            return (
              <li key={m.id} className="flex gap-3 sm:gap-4">
                <div className="flex flex-col items-center">
                  <span
                    className={
                      isCompleted
                        ? "flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-navy-900 text-white sm:h-7 sm:w-7"
                        : isBlocked
                        ? "flex h-6 w-6 shrink-0 items-center justify-center rounded-full border-2 border-danger bg-danger-soft sm:h-7 sm:w-7"
                        : isInProgress
                        ? "flex h-6 w-6 shrink-0 items-center justify-center rounded-full border-2 border-navy-900 bg-white sm:h-7 sm:w-7"
                        : "flex h-6 w-6 shrink-0 items-center justify-center rounded-full border-2 border-border bg-white sm:h-7 sm:w-7"
                    }
                  >
                    {isCompleted && <Check className="h-3 w-3 sm:h-3.5 sm:w-3.5" />}
                  </span>
                  {!isLast && <span className="w-px flex-1 bg-border" />}
                </div>
                <div className={isLast ? "pb-0" : "pb-6 sm:pb-7"}>
                  <p className={isCompleted || isInProgress ? "text-[13.5px] font-semibold text-text-primary sm:text-[14.5px]" : "text-[13.5px] font-medium text-text-secondary sm:text-[14.5px]"}>
                    {m.name}
                  </p>
                  <p className="mt-0.5 text-[12px] text-text-secondary sm:text-[12.5px]">
                    {isCompleted
                      ? `Selesai pada ${formatDate(m.completedDate)}`
                      : isBlocked
                      ? "Sedang terhambat — tim kami sedang menindaklanjuti"
                      : isInProgress
                      ? "Sedang berjalan"
                      : `Menuju target ${formatDate(m.targetDate)}`}
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
