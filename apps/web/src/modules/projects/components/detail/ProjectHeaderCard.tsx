import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { Pencil, AlertTriangle, Archive, ArchiveRestore, Copy } from "lucide-react";
import { Card, CardContent } from "@/shared/components/ui/Card";
import { Button } from "@/shared/components/ui/Button";
import { Badge } from "@/shared/components/ui/Badge";
import { ProgressMeter } from "@/shared/components/ui/ProgressMeter";
import { ProjectStatusBadge, ConditionBadge } from "@/modules/projects/components/StatusBadges";
import { ProjectFormModal } from "@/modules/projects/components/ProjectFormModal";
import type { ProjectFormValues } from "@/modules/projects/schemas/project.schema";
import { useProjectStore } from "@/modules/projects/stores/useProjectStore";
import { useStaffStore } from "@/modules/users/stores/useStaffStore";
import { useAuthStore } from "@/shared/stores/useAuthStore";
import { daysUntil } from "@/modules/projects/lib/dates";
import { formatCurrency, formatDate } from "@/shared/lib/formatters";
import { getApiErrorMessage } from "@/shared/lib/api-error";
import { ROUTE_PATHS } from "@/app/routes/route-paths";

export function ProjectHeaderCard({ projectId }: { projectId: string }) {
  const navigate = useNavigate();
  const project = useProjectStore((s) => s.currentProject);
  const milestones = useProjectStore((s) => s.milestones);
  const vendorMilestones = useProjectStore((s) => s.vendorMilestones);
  const vendorEngagements = useProjectStore((s) => s.vendorEngagements);
  const clientPayments = useProjectStore((s) => s.clientPayments);
  const fetchMilestones = useProjectStore((s) => s.fetchMilestones);
  const fetchVendorSection = useProjectStore((s) => s.fetchVendorSection);
  const fetchClientPayments = useProjectStore((s) => s.fetchClientPayments);
  const updateProject = useProjectStore((s) => s.updateProject);
  const cancelProject = useProjectStore((s) => s.cancelProject);
  const toggleArchiveProject = useProjectStore((s) => s.toggleArchiveProject);
  const deleteProject = useProjectStore((s) => s.deleteProject);
  const duplicateProject = useProjectStore((s) => s.duplicateProject);
  const staff = useStaffStore((s) => s.staffSummaries);
  const fetchStaff = useStaffStore((s) => s.fetchStaffSummaries);
  const isOwner = useAuthStore((s) => s.session?.role === "Owner");
  // Wedding Planner never duplicates a project (duplicating creates a new
  // one — same restriction as creating from scratch, see PLAN.md's RBAC
  // section); the backend already rejects this role's POST .../duplicate.
  const canDuplicate = useAuthStore((s) => s.session?.role) !== "Staff";
  // Margin/Keuntungan reveals vendor-cost/venue-cost business data — Wedding
  // Planner is not supposed to see this specific computed figure (confirmed
  // RBAC rule), even though the underlying vendor engagements and venue cost
  // snapshot it's derived from are already visible to every role via the
  // Vendor/Venue tabs themselves (no separate fetch to gate here anymore —
  // both now come from data already loaded for this project).
  const canSeeMargin = useAuthStore((s) => s.session?.role) !== "Staff";

  const [editOpen, setEditOpen] = useState(false);
  const [duplicateOpen, setDuplicateOpen] = useState(false);
  const [confirmingCancel, setConfirmingCancel] = useState(false);
  const [confirmingDelete, setConfirmingDelete] = useState(false);
  const [isDeleting, setIsDeleting] = useState(false);
  const [actionError, setActionError] = useState<string | null>(null);

  useEffect(() => {
    void fetchMilestones(projectId);
    void fetchVendorSection(projectId);
    void fetchClientPayments(projectId);
    void fetchStaff();
  }, [projectId, fetchMilestones, fetchVendorSection, fetchClientPayments, fetchStaff]);

  if (!project) {
    return <div className="text-sm text-text-secondary">Project tidak ditemukan.</div>;
  }

  // Margin/Keuntungan = Nilai Kontrak − biaya vendor aktif − biaya venue
  // (PLAN.md). Cancelled engagements are excluded, same as milestone stats
  // already exclude Cancelled from every "relevant" computation.
  const vendorCost = vendorEngagements
    .filter((v) => v.engagementStatus !== "Cancelled")
    .reduce((sum, v) => sum + v.contractValue, 0);
  // Read directly off the project's own cost snapshot — never a live venue
  // fetch (see PLAN.md "Financial Calculation Correctness"): a completed
  // project's recorded Margin must never drift just because venue master
  // data changed later.
  const venueCost = (project.venueRentalPrice ?? 0) + (project.venueCharge ?? 0);
  const margin = project.contractValue - vendorCost - venueCost;

  // Sisa Tagihan Client (PLAN.md §1.7/§3.9) — day-to-day operational status
  // ("has the client paid this installment"), visible to every role that
  // already sees Nilai Kontrak, unlike Margin/Keuntungan above.
  const totalReceived = clientPayments.reduce(
    (sum, p) => (p.type === "Refund" ? sum - p.amount : sum + p.amount),
    0
  );
  const outstanding = project.contractValue - totalReceived;

  const progress = project.progress;
  const segments = [...milestones, ...vendorMilestones];
  const totalMilestones = (progress?.projectMilestoneStats.total ?? 0) + (progress?.vendorMilestoneStats.total ?? 0);
  const completedMilestones = (progress?.projectMilestoneStats.completed ?? 0) + (progress?.vendorMilestoneStats.completed ?? 0);
  const pic = staff.find((s) => s.id === project.picStaffId);
  const d = daysUntil(project.eventDate);
  const isOpenProject = project.status !== "Completed" && project.status !== "Cancelled";

  async function handleEdit(values: ProjectFormValues) {
    setActionError(null);
    try {
      await updateProject(projectId, values);
      setEditOpen(false);
    } catch (err) {
      setActionError(getApiErrorMessage(err, "Gagal menyimpan perubahan project"));
    }
  }

  async function handleCancelProject() {
    setActionError(null);
    try {
      await cancelProject(projectId);
      setConfirmingCancel(false);
    } catch (err) {
      setActionError(getApiErrorMessage(err, "Gagal membatalkan project"));
    }
  }

  async function handleToggleArchive() {
    setActionError(null);
    try {
      await toggleArchiveProject(projectId);
    } catch (err) {
      setActionError(getApiErrorMessage(err, "Gagal mengubah status arsip project"));
    }
  }

  async function handleDuplicate(values: ProjectFormValues) {
    setActionError(null);
    try {
      const duplicated = await duplicateProject(projectId, values);
      setDuplicateOpen(false);
      navigate(ROUTE_PATHS.projectDetail(duplicated.id));
    } catch (err) {
      setActionError(getApiErrorMessage(err, "Gagal menduplikasi project"));
    }
  }

  async function handleDeleteProject() {
    setActionError(null);
    setIsDeleting(true);
    try {
      await deleteProject(projectId);
      navigate(ROUTE_PATHS.projects);
    } catch (err) {
      setActionError(getApiErrorMessage(err, "Gagal menghapus project secara permanen"));
      setIsDeleting(false);
    }
  }

  return (
    <Card>
      <CardContent className="flex flex-col gap-5 py-5">
        <div className="flex flex-wrap items-start justify-between gap-4">
          <div>
            <div className="flex flex-wrap items-center gap-2">
              <h1 className="text-xl font-bold text-text-primary">{project.name}</h1>
              <ProjectStatusBadge status={project.status} />
              {project.isArchived && <Badge tone="neutral">Diarsipkan</Badge>}
            </div>
            <p className="mt-1 text-[13px] text-text-secondary">
              {project.brideName} &amp; {project.groomName} · {project.venue}
            </p>
          </div>
          <div className="flex shrink-0 flex-wrap items-center gap-2">
            <Button variant="secondary" size="sm" icon={<Pencil className="h-3.5 w-3.5" />} onClick={() => setEditOpen(true)}>
              Ubah Project
            </Button>
            {canDuplicate && (
              <Button variant="secondary" size="sm" icon={<Copy className="h-3.5 w-3.5" />} onClick={() => setDuplicateOpen(true)}>
                Duplikat Project
              </Button>
            )}
            <Button
              variant="secondary"
              size="sm"
              icon={project.isArchived ? <ArchiveRestore className="h-3.5 w-3.5" /> : <Archive className="h-3.5 w-3.5" />}
              onClick={() => void handleToggleArchive()}
            >
              {project.isArchived ? "Pulihkan dari Arsip" : "Arsipkan"}
            </Button>
            {isOpenProject && (
              <Button variant="danger" size="sm" onClick={() => setConfirmingCancel(true)}>
                Batalkan Project
              </Button>
            )}
            {isOwner && (project.isArchived || project.status === "Cancelled") && (
              <Button variant="danger" size="sm" onClick={() => setConfirmingDelete(true)}>
                Hapus Permanen
              </Button>
            )}
          </div>
        </div>

        {actionError && (
          <p className="rounded-md border border-danger/30 bg-danger-soft px-3.5 py-2.5 text-[13px] font-medium text-danger">{actionError}</p>
        )}

        {confirmingCancel && (
          <div className="flex flex-wrap items-center justify-between gap-3 rounded-md border border-danger/30 bg-danger-soft px-4 py-3">
            <span className="flex items-center gap-2 text-[13px] font-medium text-danger">
              <AlertTriangle className="h-4 w-4" /> Yakin ingin membatalkan project ini? Data yang sudah ada tidak akan dihapus.
            </span>
            <span className="flex gap-2">
              <Button variant="secondary" size="sm" onClick={() => setConfirmingCancel(false)}>Batal</Button>
              <Button variant="danger" size="sm" onClick={() => void handleCancelProject()}>Ya, Batalkan</Button>
            </span>
          </div>
        )}

        {confirmingDelete && (
          <div className="flex flex-wrap items-center justify-between gap-3 rounded-md border border-danger/30 bg-danger-soft px-4 py-3">
            <span className="flex items-center gap-2 text-[13px] font-medium text-danger">
              <AlertTriangle className="h-4 w-4" /> Yakin ingin menghapus project ini secara permanen? Tindakan ini permanen — seluruh
              timeline, vendor, pembayaran, kendala, dan evidence akan ikut terhapus dan tidak dapat dipulihkan.
            </span>
            <span className="flex gap-2">
              <Button variant="secondary" size="sm" onClick={() => setConfirmingDelete(false)} disabled={isDeleting}>Batal</Button>
              <Button variant="danger" size="sm" onClick={() => void handleDeleteProject()} disabled={isDeleting}>
                {isDeleting ? "Menghapus..." : "Ya, Hapus Permanen"}
              </Button>
            </span>
          </div>
        )}

        <div className="grid grid-cols-2 gap-4 border-t border-border-light pt-4 sm:grid-cols-3 lg:grid-cols-6">
          <InfoField label="Tanggal Acara" value={formatDate(project.eventDate)} />
          <InfoField
            label="Countdown"
            value={isOpenProject ? (d >= 0 ? `H-${d}` : `H+${Math.abs(d)}`) : project.status === "Completed" ? "Selesai" : "Dibatalkan"}
            emphasize
          />
          <InfoField label="Mulai Persiapan" value={formatDate(project.prepStartDate)} />
          <InfoField label="Paket / Layanan" value={project.packageName} />
          <InfoField label="Nilai Kontrak" value={formatCurrency(project.contractValue)} />
          <InfoField label="Sisa Tagihan Client" value={formatCurrency(outstanding)} />
          {canSeeMargin && <InfoField label="Margin/Keuntungan" value={formatCurrency(margin)} />}
          <InfoField label="Penanggung Jawab" value={pic?.name ?? "-"} />
        </div>

        {project.description && (
          <p className="rounded-md bg-surface-muted px-4 py-3 text-[13px] text-text-secondary">{project.description}</p>
        )}

        {progress && (
          <div className="rounded-md bg-surface-muted/60 p-4">
            <div className="mb-2 flex flex-wrap items-center justify-between gap-2">
              <div className="flex items-center gap-3">
                <span className="text-2xl font-bold leading-none tabular-nums text-navy-900">{progress.overallPercent}%</span>
                <ConditionBadge condition={progress.condition} />
              </div>
              <span className="text-[12.5px] text-text-secondary">
                Timeline project {progress.projectMilestoneStats.completed}/{progress.projectMilestoneStats.total} · Timeline vendor{" "}
                {progress.vendorMilestoneStats.completed}/{progress.vendorMilestoneStats.total}
                {progress.overdueMilestoneCount > 0 && (
                  <span className="font-semibold text-danger"> · {progress.overdueMilestoneCount} terlambat</span>
                )}
              </span>
            </div>
            <ProgressMeter
              percent={progress.overallPercent}
              segments={segments}
              caption={`${completedMilestones}/${totalMilestones} timeline selesai berdasarkan pencapaian nyata — bukan estimasi manual.`}
            />
          </div>
        )}
      </CardContent>

      <ProjectFormModal open={editOpen} onClose={() => setEditOpen(false)} onSubmit={(values) => void handleEdit(values)} initialProject={project} />
      <ProjectFormModal
        open={duplicateOpen}
        onClose={() => setDuplicateOpen(false)}
        onSubmit={(values) => void handleDuplicate(values)}
        initialProject={project}
        mode="duplicate"
      />
    </Card>
  );
}

function InfoField({ label, value, emphasize }: { label: string; value: string; emphasize?: boolean }) {
  return (
    <div>
      <p className="text-[11.5px] font-medium uppercase tracking-wide text-text-secondary">{label}</p>
      <p className={emphasize ? "mt-0.5 text-[15px] font-bold tabular-nums text-navy-900" : "mt-0.5 text-[13.5px] font-medium text-text-primary"}>
        {value}
      </p>
    </div>
  );
}
