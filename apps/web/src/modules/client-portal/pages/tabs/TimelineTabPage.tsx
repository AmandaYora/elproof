import { useOutletContext } from "react-router-dom";
import { LoadError } from "@/shared/components/feedback/LoadError";
import { SkeletonRows } from "@/shared/components/ui/Skeleton";
import { useProjectStore } from "@/modules/projects/stores/useProjectStore";
import { MilestoneRail } from "@/modules/client-portal/components/MilestoneRail";
import { useProofViewer } from "@/modules/client-portal/components/useProofViewer";
import { useLoad } from "@/modules/client-portal/lib/useLoad";
import type { ClientPortalContext } from "@/modules/client-portal/layouts/ClientPortalLayout";

/**
 * Jadwal persiapan, in two short rails: what is happening and what comes next
 * (nearest first), then what is already done (latest first, with its proof).
 */
export default function TimelineTabPage() {
  const { projectId } = useOutletContext<ClientPortalContext>();
  const milestones = useProjectStore((s) => s.milestones);
  const fetchMilestones = useProjectStore((s) => s.fetchMilestones);
  const viewer = useProofViewer(projectId);
  const { status, retry } = useLoad(() => fetchMilestones(projectId), [projectId, fetchMilestones]);

  const live = milestones.filter((m) => m.status !== "Cancelled");
  const ahead = live
    .filter((m) => m.status !== "Completed")
    // In progress and blocked first, then by target date.
    .sort((a, b) => Number(b.status === "In Progress" || b.status === "Blocked") - Number(a.status === "In Progress" || a.status === "Blocked") || a.targetDate.localeCompare(b.targetDate));
  const done = live.filter((m) => m.status === "Completed").sort((a, b) => (b.completedDate ?? "").localeCompare(a.completedDate ?? ""));

  return (
    <div className="mx-auto flex w-full max-w-3xl flex-col gap-6 sm:gap-8">
      <header>
        <h1 className="text-xl font-semibold text-foreground">Jadwal persiapan</h1>
        <p className="mt-1.5 max-w-prose text-[14px] leading-relaxed text-muted-foreground">Tahapan persiapan pernikahan Anda. Diperbarui begitu tim menyelesaikan satu tahapan.</p>
      </header>

      {status === "loading" && <div className="rounded-2xl border border-border bg-card p-5"><SkeletonRows rows={4} label="Memuat jadwal" /></div>}
      {status === "error" && <LoadError message="Jadwal belum bisa dimuat. Periksa koneksi, lalu coba lagi." onRetry={retry} />}

      {status === "ready" && live.length === 0 && (
        <p className="rounded-2xl border border-dashed border-border bg-card p-6 text-center text-[13.5px] text-muted-foreground">Jadwal persiapan Anda belum disusun. Wedding organizer Anda akan mengabari begitu ada.</p>
      )}

      {status === "ready" && ahead.length > 0 && (
        <section aria-labelledby="ahead-heading" className="rounded-2xl border border-border bg-card p-5 sm:p-6">
          <h2 id="ahead-heading" className="mb-4 text-[15px] font-semibold text-foreground">Sedang dan akan berjalan</h2>
          <MilestoneRail steps={ahead} onOpenProof={viewer.open} />
        </section>
      )}

      {status === "ready" && done.length > 0 && (
        <section aria-labelledby="done-heading" className="rounded-2xl border border-border bg-card p-5 sm:p-6">
          <h2 id="done-heading" className="mb-4 text-[15px] font-semibold text-foreground">Sudah selesai</h2>
          <MilestoneRail steps={done} onOpenProof={viewer.open} />
        </section>
      )}
      {viewer.element}
    </div>
  );
}
