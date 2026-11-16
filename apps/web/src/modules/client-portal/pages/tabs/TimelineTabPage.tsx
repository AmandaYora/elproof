import { useState } from "react";
import { useOutletContext } from "react-router-dom";
import { ChevronDown } from "lucide-react";
import { LoadError } from "@/shared/components/feedback/LoadError";
import { SkeletonRows } from "@/shared/components/ui/Skeleton";
import { cn } from "@/shared/lib/cn";
import { useProjectStore } from "@/modules/projects/stores/useProjectStore";
import { useClientHomeStore } from "@/modules/client-portal/stores/useClientHomeStore";
import { MilestoneRail } from "@/modules/client-portal/components/MilestoneRail";
import { PageIntro } from "@/modules/client-portal/components/PageIntro";
import { useProofViewer } from "@/modules/client-portal/components/useProofViewer";
import { useLoad } from "@/modules/client-portal/lib/useLoad";
import { phaseSteps } from "@/modules/client-portal/lib/phases";
import type { ClientPortalContext } from "@/modules/client-portal/layouts/ClientPortalLayout";

/**
 * Jadwal persiapan grouped the way couples plan a wedding (a Zola-style checklist, RISET #5): this month, 1–3 months
 * away, further out, and the last two weeks before the day. Each step says how many days before the day it is
 * ("H-30") and its date; finished steps fold into "Sudah selesai (n)".
 */
export default function TimelineTabPage() {
  const { projectId } = useOutletContext<ClientPortalContext>();
  const eventDate = useClientHomeStore((s) => s.home?.project.eventDate);
  const milestones = useProjectStore((s) => s.milestones);
  const fetchMilestones = useProjectStore((s) => s.fetchMilestones);
  const viewer = useProofViewer(projectId);
  const [showDone, setShowDone] = useState(false);
  const { status, retry } = useLoad(() => fetchMilestones(projectId), [projectId, fetchMilestones]);

  const live = milestones.filter((m) => m.status !== "Cancelled");
  const phases = phaseSteps(live.filter((m) => m.status !== "Completed"), eventDate);
  const done = live.filter((m) => m.status === "Completed").sort((a, b) => (b.completedDate ?? "").localeCompare(a.completedDate ?? ""));

  return (
    <div className="mx-auto flex w-full max-w-3xl flex-col gap-6 sm:gap-8">
      <PageIntro title="Jadwal persiapan">Tahapan persiapan pernikahan Anda, diperbarui begitu tim menyelesaikan satu tahapan.</PageIntro>

      {status === "loading" && <div className="rounded-2xl border border-border bg-card p-5"><SkeletonRows rows={4} label="Memuat jadwal" /></div>}
      {status === "error" && <LoadError message="Jadwal belum bisa dimuat. Periksa koneksi, lalu coba lagi." onRetry={retry} />}

      {status === "ready" && live.length === 0 && (
        <p className="rounded-2xl border border-dashed border-border bg-card p-6 text-center text-[14px] text-muted-foreground">Jadwal persiapan Anda belum disusun. Wedding organizer Anda akan mengabari begitu ada.</p>
      )}

      {status === "ready" &&
        phases.map((p) => (
          <section key={p.id} aria-labelledby={`phase-${p.id}`}>
            <div className="mb-3 flex items-baseline justify-between gap-3">
              <h2 id={`phase-${p.id}`} className="text-base font-semibold text-foreground">{p.label}</h2>
              <span className="text-[13px] text-muted-foreground">{p.steps.length} tahapan</span>
            </div>
            <div className="rounded-2xl border border-border bg-card p-5 sm:p-6">
              <MilestoneRail steps={p.steps} onOpenProof={viewer.open} eventDate={eventDate} />
            </div>
          </section>
        ))}

      {status === "ready" && done.length > 0 && (
        <section aria-labelledby="done-heading" className="rounded-2xl border border-border bg-card">
          <h2 id="done-heading">
            <button
              type="button"
              aria-expanded={showDone}
              aria-controls={showDone ? "done-steps" : undefined}
              onClick={() => setShowDone((v) => !v)}
              className="flex min-h-14 w-full items-center justify-between gap-3 rounded-2xl px-5 py-3 text-left text-base font-semibold text-foreground focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-solid focus-visible:outline-ring sm:px-6"
            >
              Sudah selesai ({done.length})
              <ChevronDown className={cn("size-5 text-muted-foreground motion-safe:transition-transform", showDone && "rotate-180")} aria-hidden />
            </button>
          </h2>
          {showDone && (
            <div id="done-steps" className="border-t border-border-light px-5 pt-4 pb-5 sm:px-6">
              <MilestoneRail steps={done} onOpenProof={viewer.open} />
            </div>
          )}
        </section>
      )}
      {viewer.element}
    </div>
  );
}
