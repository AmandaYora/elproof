import { useEffect, useRef, useState } from "react";
import { useCan } from "@/shared/stores/usePermissionStore";
import { useNavigate } from "react-router-dom";
import { Pencil, AlertTriangle, Archive, ArchiveRestore, Copy, Check, CircleAlert, Trash2, Ban } from "lucide-react";
import { Card, CardContent } from "@/shared/components/ui/Card";
import { Button } from "@/shared/components/ui/Button";
import { Badge } from "@/shared/components/ui/Badge";
import { ActionMenu } from "@/shared/components/ui/ActionMenu";
import { Modal } from "@/shared/components/ui/Modal";
import { ProgressMeter } from "@/shared/components/ui/ProgressMeter";
import { ProjectStatusBadge, ConditionBadge } from "@/modules/projects/components/StatusBadges";
import { ProjectFormModal } from "@/modules/projects/components/ProjectFormModal";
import type { ProjectFormValues } from "@/modules/projects/schemas/project.schema";
import type { ClosingChecklist, ProjectStatus, ProjectStatusAction } from "@/modules/projects/types";
import { useProjectStore } from "@/modules/projects/stores/useProjectStore";
import { useStaffStore } from "@/modules/users/stores/useStaffStore";
import { daysUntil } from "@/modules/projects/lib/dates";
import { formatCurrency, formatDate } from "@/shared/lib/formatters";
import { getApiErrorMessage } from "@/shared/lib/api-error";
import { toast } from "@/shared/stores/useToastStore";
import { ROUTE_PATHS } from "@/app/routes/route-paths";

// The one step each status offers next (FLOW-IDEAL 2.4). Moving through the
// project's life uses these buttons, not a free choice of status.
const LIFECYCLE_ACTION: Record<ProjectStatus, { action: ProjectStatusAction; label: string; done: string; primary: boolean }> = {
  Draft: { action: "start_preparation", label: "Mulai persiapan", done: "Project masuk tahap persiapan", primary: true },
  Preparation: { action: "mark_ready", label: "Tandai siap", done: "Project ditandai siap", primary: true },
  Ready: { action: "complete", label: "Selesaikan project", done: "Project diselesaikan", primary: true },
  Completed: { action: "reopen", label: "Buka kembali", done: "Project dibuka kembali", primary: false },
  Cancelled: { action: "reopen", label: "Buka kembali", done: "Project dibuka kembali", primary: false },
};

export function ProjectHeaderCard({ projectId }: { projectId: string }) {
  const navigate = useNavigate();
  const project = useProjectStore((s) => s.currentProject);
  const milestones = useProjectStore((s) => s.milestones);
  const vendorMilestones = useProjectStore((s) => s.vendorMilestones);
  const vendorEngagements = useProjectStore((s) => s.vendorEngagements);
  const fetchMilestones = useProjectStore((s) => s.fetchMilestones);
  const fetchVendorSection = useProjectStore((s) => s.fetchVendorSection);
  const fetchClientPayments = useProjectStore((s) => s.fetchClientPayments);
  const updateProject = useProjectStore((s) => s.updateProject);
  const cancelProject = useProjectStore((s) => s.cancelProject);
  const transitionProject = useProjectStore((s) => s.transitionProject);
  const fetchClosingChecklist = useProjectStore((s) => s.fetchClosingChecklist);
  const toggleArchiveProject = useProjectStore((s) => s.toggleArchiveProject);
  const deleteProject = useProjectStore((s) => s.deleteProject);
  const duplicateProject = useProjectStore((s) => s.duplicateProject);
  const fetchStaff = useStaffStore((s) => s.fetchStaffSummaries);
  // Actions follow the workspace role's permissions (ADR-0026).
  const canDelete = useCan("project.delete");
  const canEdit = useCan("project.edit");
  const canArchive = useCan("project.archive");
  // Neither Wedding Planner nor Sales duplicates a project (duplicating
  // creates a new one — same restriction as creating from scratch, see
  // PLAN.md's RBAC section); the backend already rejects both roles' POST
  // .../duplicate.
  const canDuplicate = useCan("project.duplicate");
  // Margin/Keuntungan reveals vendor-cost/venue-cost business data — Wedding
  // Planner is not supposed to see this specific computed figure (confirmed
  // RBAC rule), even though the underlying vendor engagements and venue cost
  // snapshot it's derived from are already visible to every role via the
  // Vendor/Venue tabs themselves (no separate fetch to gate here anymore —
  // both now come from data already loaded for this project).
  const canSeeMargin = useCan("finance.margin.view");

  const [editOpen, setEditOpen] = useState(false);
  const [duplicateOpen, setDuplicateOpen] = useState(false);
  const [confirmingCancel, setConfirmingCancel] = useState(false);
  const [confirmingDelete, setConfirmingDelete] = useState(false);
  const [isDeleting, setIsDeleting] = useState(false);
  const [actionError, setActionError] = useState<string | null>(null);
  const [transitioning, setTransitioning] = useState(false);
  // Closing dialog: null = closed; the checklist loads when it opens.
  const [closing, setClosing] = useState<{ checklist: ClosingChecklist | null; error: string | null } | null>(null);
  const [completing, setCompleting] = useState(false);
  const closingToken = useRef(0);

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

  const progress = project.progress;
  const segments = [...milestones, ...vendorMilestones];
  const totalMilestones = (progress?.projectMilestoneStats.total ?? 0) + (progress?.vendorMilestoneStats.total ?? 0);
  const completedMilestones = (progress?.projectMilestoneStats.completed ?? 0) + (progress?.vendorMilestoneStats.completed ?? 0);
  const d = daysUntil(project.eventDate);
  const isOpenProject = project.status !== "Completed" && project.status !== "Cancelled";
  const lifecycle = LIFECYCLE_ACTION[project.status];

  // The two form handlers throw on failure so the modal can show the
  // message next to the form, with what the user typed still in place.
  async function handleEdit(values: ProjectFormValues) {
    await updateProject(projectId, values);
    setEditOpen(false);
    toast.success("Perubahan project disimpan");
  }

  async function handleTransition(action: ProjectStatusAction, successMessage: string) {
    setActionError(null);
    setTransitioning(true);
    try {
      await transitionProject(projectId, action);
      toast.success(successMessage);
    } catch (err) {
      setActionError(getApiErrorMessage(err, "Status project belum berubah. Coba lagi."));
    } finally {
      setTransitioning(false);
    }
  }

  // "Selesaikan project" first shows what is still open, so closing is a
  // decision made with the facts in front of the user.
  async function openClosing() {
    const token = ++closingToken.current;
    setClosing({ checklist: null, error: null });
    try {
      const checklist = await fetchClosingChecklist(projectId);
      // The user may have closed the dialog while this was loading.
      if (token === closingToken.current) setClosing({ checklist, error: null });
    } catch (err) {
      if (token === closingToken.current) {
        setClosing({ checklist: null, error: getApiErrorMessage(err, "Daftar periksa belum bisa dimuat. Coba lagi.") });
      }
    }
  }

  function closeClosing() {
    closingToken.current++;
    setClosing(null);
  }

  async function handleComplete() {
    setCompleting(true);
    try {
      await transitionProject(projectId, "complete");
      setClosing(null);
      toast.success("Project diselesaikan");
    } catch (err) {
      setClosing((c) => (c ? { ...c, error: getApiErrorMessage(err, "Project belum bisa diselesaikan. Coba lagi.") } : c));
    } finally {
      setCompleting(false);
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
    const duplicated = await duplicateProject(projectId, values);
    setDuplicateOpen(false);
    toast.success("Project diduplikat");
    navigate(ROUTE_PATHS.projectDetail(duplicated.id));
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
            {canEdit && lifecycle && (
              <Button
                variant={lifecycle.primary ? "primary" : "secondary"}
                size="sm"
                disabled={transitioning}
                onClick={() => (lifecycle.action === "complete" ? void openClosing() : void handleTransition(lifecycle.action, lifecycle.done))}
              >
                {transitioning ? "Memproses..." : lifecycle.label}
              </Button>
            )}
            {canEdit && (
              <Button variant="secondary" size="sm" icon={<Pencil className="h-3.5 w-3.5" />} onClick={() => setEditOpen(true)}>
                Ubah Project
              </Button>
            )}
            <ActionMenu
              items={[
                { label: "Duplikat project", icon: <Copy className="h-4 w-4" />, onSelect: () => setDuplicateOpen(true), hidden: !canDuplicate },
                {
                  label: project.isArchived ? "Pulihkan dari arsip" : "Arsipkan",
                  icon: project.isArchived ? <ArchiveRestore className="h-4 w-4" /> : <Archive className="h-4 w-4" />,
                  onSelect: () => void handleToggleArchive(),
                  hidden: !canArchive,
                },
                { label: "Batalkan project", icon: <Ban className="h-4 w-4" />, tone: "danger", onSelect: () => setConfirmingCancel(true), hidden: !(canEdit && isOpenProject) },
                {
                  label: "Hapus permanen",
                  icon: <Trash2 className="h-4 w-4" />,
                  tone: "danger",
                  onSelect: () => setConfirmingDelete(true),
                  hidden: !(canDelete && (project.isArchived || project.status === "Cancelled")),
                },
              ]}
            />
          </div>
        </div>

        {actionError && (
          <p className="rounded-md border border-danger/30 bg-danger-soft px-3.5 py-2.5 text-[13px] font-medium text-danger-text">{actionError}</p>
        )}

        {confirmingCancel && (
          <div className="flex flex-wrap items-center justify-between gap-3 rounded-md border border-danger/30 bg-danger-soft px-4 py-3">
            <span className="flex items-center gap-2 text-[13px] font-medium text-danger-text">
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
            <span className="flex items-center gap-2 text-[13px] font-medium text-danger-text">
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

        <div className="grid grid-cols-2 gap-4 border-t border-border-light pt-4 sm:grid-cols-3 lg:grid-cols-5">
          <InfoField label="Tanggal Acara" value={formatDate(project.eventDate)} />
          <InfoField
            label="Countdown"
            value={isOpenProject ? (d >= 0 ? `H-${d}` : `H+${Math.abs(d)}`) : project.status === "Completed" ? "Selesai" : "Dibatalkan"}
            emphasize
          />
          <InfoField label="Tanggal Booking" value={formatDate(project.prepStartDate)} />
          <InfoField label="Paket / Layanan" value={project.packageName} />
          <InfoField label="Nilai Kontrak" value={formatCurrency(project.contractValue)} />
          {canSeeMargin && <InfoField label="Margin" value={formatCurrency(margin)} />}
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
                  <span className="font-semibold text-danger-text"> · {progress.overdueMilestoneCount} terlambat</span>
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

      {closing && (
        <Modal
          open
          onClose={closeClosing}
          title="Selesaikan project"
          description="Periksa dulu apakah semuanya sudah beres. Anda tetap bisa menyelesaikan project meski ada yang belum tuntas."
          busy={completing}
          error={closing.error}
          footer={
            <>
              <Button variant="secondary" onClick={closeClosing} disabled={completing}>Kembali</Button>
              <Button onClick={() => void handleComplete()} disabled={completing || closing.checklist === null}>
                {completing ? "Menyelesaikan..." : closing.checklist && !closing.checklist.ready ? "Tetap selesaikan" : "Selesaikan project"}
              </Button>
            </>
          }
        >
          {closing.checklist === null && !closing.error && <p className="text-[13px] text-text-secondary">Memeriksa...</p>}
          {closing.checklist && (
            <ul className="flex flex-col gap-3">
              {closing.checklist.items.map((item) => (
                <li key={item.key} className="flex items-start gap-3">
                  <span
                    aria-hidden
                    className={
                      item.done
                        ? "mt-0.5 flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-success-soft text-success-text"
                        : "mt-0.5 flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-warning-soft text-warning-text"
                    }
                  >
                    {item.done ? <Check className="h-3.5 w-3.5" /> : <CircleAlert className="h-3.5 w-3.5" />}
                  </span>
                  <div>
                    <p className="text-[13.5px] font-medium text-text-primary">
                      {item.label}
                      <span className="sr-only">{item.done ? " — sudah" : " — belum"}</span>
                    </p>
                    {!item.done && item.detail && <p className="text-[12.5px] text-text-secondary">{item.detail}</p>}
                  </div>
                </li>
              ))}
            </ul>
          )}
        </Modal>
      )}
      <ProjectFormModal open={editOpen} onClose={() => setEditOpen(false)} onSubmit={handleEdit} initialProject={project} />
      <ProjectFormModal
        open={duplicateOpen}
        onClose={() => setDuplicateOpen(false)}
        onSubmit={handleDuplicate}
        initialProject={project}
        mode="duplicate"
      />
    </Card>
  );
}

function InfoField({ label, value, emphasize }: { label: string; value: string; emphasize?: boolean }) {
  return (
    <div className="min-w-0">
      <p className="text-[12px] font-medium text-text-secondary">{label}</p>
      <p className={emphasize ? "mt-0.5 text-[15px] font-bold tabular-nums text-navy-900" : "mt-0.5 text-[13.5px] font-medium text-text-primary"}>
        {value}
      </p>
    </div>
  );
}
