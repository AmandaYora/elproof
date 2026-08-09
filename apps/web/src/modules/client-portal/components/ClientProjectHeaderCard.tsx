import { useEffect } from "react";
import { Link } from "react-router-dom";
import { HeartHandshake } from "lucide-react";
import { Badge } from "@/shared/components/ui/Badge";
import { ProgressMeter } from "@/shared/components/ui/ProgressMeter";
import { ProjectStatusBadge } from "@/modules/projects/components/StatusBadges";
import { useProjectStore } from "@/modules/projects/stores/useProjectStore";
import { CLIENT_CONDITION_COPY } from "@/modules/client-portal/lib/condition";
import { formatCurrency, formatDate } from "@/shared/lib/formatters";
import { ROUTE_PATHS } from "@/app/routes/route-paths";

function InfoField({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <p className="text-[11.5px] font-medium uppercase tracking-wide text-text-secondary">{label}</p>
      <p className="mt-1 text-[14px] font-semibold text-navy-950">{value}</p>
    </div>
  );
}

// Read-only sibling of the WO Console's ProjectHeaderCard (PLAN.md's Client
// Portal restructure) — same info grid + progress-bar structure, minus every
// internal-only field (Margin/Keuntungan, Mulai Persiapan/Tanggal Booking)
// and every action button (Ubah/Duplikat/Arsipkan/Batalkan/Hapus). Rendered
// once, persistently, above the tab strip in ClientPortalLayout.tsx —
// absorbs what used to be the standalone "Ringkasan" tab's summary section;
// the richer milestone stepper that tab also had now lives in its own
// "Timeline" tab instead (TimelineTabPage.tsx).
export function ClientProjectHeaderCard({ projectId }: { projectId: string }) {
  const project = useProjectStore((s) => s.currentProject);
  const milestones = useProjectStore((s) => s.milestones);
  const vendorMilestones = useProjectStore((s) => s.vendorMilestones);
  const clientPayments = useProjectStore((s) => s.clientPayments);
  const issues = useProjectStore((s) => s.issues);
  const fetchMilestones = useProjectStore((s) => s.fetchMilestones);
  const fetchVendorSection = useProjectStore((s) => s.fetchVendorSection);
  const fetchClientPayments = useProjectStore((s) => s.fetchClientPayments);
  const fetchIssues = useProjectStore((s) => s.fetchIssues);

  useEffect(() => {
    void fetchMilestones(projectId);
    void fetchVendorSection(projectId);
    void fetchClientPayments(projectId);
    void fetchIssues(projectId);
  }, [projectId, fetchMilestones, fetchVendorSection, fetchClientPayments, fetchIssues]);

  if (!project) return null;

  // Sisa Tagihan — identical formula to WO Console's own "Sisa Tagihan
  // Client" (ProjectHeaderCard.tsx), independently re-derived here since no
  // shared helper exists yet for it.
  const totalReceived = clientPayments.reduce((sum, p) => (p.type === "Refund" ? sum - p.amount : sum + p.amount), 0);
  const outstanding = project.contractValue - totalReceived;
  const openIssueCount = issues.filter((i) => i.status !== "Resolved" && i.status !== "Closed").length;

  const progress = project.progress;
  const segments = [...milestones, ...vendorMilestones];
  const totalMilestones = (progress?.projectMilestoneStats.total ?? 0) + (progress?.vendorMilestoneStats.total ?? 0);
  const completedMilestones = (progress?.projectMilestoneStats.completed ?? 0) + (progress?.vendorMilestoneStats.completed ?? 0);
  const conditionCopy = progress ? CLIENT_CONDITION_COPY[progress.condition] : null;

  return (
    <div className="flex flex-col gap-5 rounded-3xl border border-border bg-white p-6 shadow-sm sm:p-8">
      <div className="flex flex-wrap items-center gap-2">
        <h1 className="text-xl font-bold text-navy-950">{project.name}</h1>
        <ProjectStatusBadge status={project.status} />
      </div>

      <div className="grid grid-cols-2 gap-4 border-t border-border-light pt-4 sm:grid-cols-3 lg:grid-cols-5">
        <InfoField label="Tanggal Acara" value={formatDate(project.eventDate)} />
        <InfoField label="Paket / Layanan" value={project.packageName} />
        <InfoField label="Nilai Kontrak" value={formatCurrency(project.contractValue)} />
        <InfoField label="Sisa Tagihan" value={formatCurrency(outstanding)} />
        <InfoField label="Penanggung Jawab" value={project.picName || "-"} />
      </div>

      {project.description && (
        <p className="rounded-md bg-surface-muted px-4 py-3 text-[13px] text-text-secondary">{project.description}</p>
      )}

      {progress && conditionCopy && (
        <div className="rounded-md bg-surface-muted/60 p-4">
          <div className="mb-2 flex flex-wrap items-center justify-between gap-2">
            <div className="flex items-center gap-3">
              <span className="text-2xl font-bold leading-none tabular-nums text-navy-900">{progress.overallPercent}%</span>
              <Badge tone={conditionCopy.tone}>{conditionCopy.label}</Badge>
            </div>
            <span className="text-[12.5px] text-text-secondary">
              Timeline {completedMilestones}/{totalMilestones} selesai
            </span>
          </div>
          <ProgressMeter
            percent={progress.overallPercent}
            segments={segments}
            caption={`${completedMilestones}/${totalMilestones} timeline selesai berdasarkan pencapaian nyata — bukan estimasi manual.`}
          />
        </div>
      )}

      {openIssueCount > 0 && (
        <Link
          to={ROUTE_PATHS.portal("kendala")}
          className="group flex flex-col items-start gap-3 rounded-2xl border border-warning/20 bg-warning-soft/50 p-4 transition-all hover:bg-warning-soft hover:shadow-md sm:flex-row sm:items-center sm:justify-between sm:px-5"
        >
          <div className="flex items-center gap-3">
            <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-warning/20 text-warning-strong">
              <span className="text-[16px] font-bold">{openIssueCount}</span>
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
  );
}
