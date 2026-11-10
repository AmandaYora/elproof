import { useRef, useState } from "react";
import { useCan } from "@/shared/stores/usePermissionStore";
import { useNavigate } from "react-router-dom";
import { Pencil, Archive, ArchiveRestore, Copy, Check, CircleAlert, Trash2, Ban } from "lucide-react";
import { Card, CardContent } from "@/shared/components/ui/Card";
import { Button } from "@/shared/components/ui/Button";
import { Badge } from "@/shared/components/ui/Badge";
import { ActionMenu } from "@/shared/components/ui/ActionMenu";
import { Modal } from "@/shared/components/ui/Modal";
import { ProgressMeter, type ProgressMeterStatus } from "@/shared/components/ui/ProgressMeter";
import { ConfirmDialog } from "@/shared/components/ui/ConfirmDialog";
import { ProjectStatusBadge, ConditionBadge } from "@/modules/projects/components/StatusBadges";
import { ProjectFormModal } from "@/modules/projects/components/ProjectFormModal";
import type { ProjectFormValues } from "@/modules/projects/schemas/project.schema";
import type { ClosingChecklist, ProjectStatus, ProjectStatusAction } from "@/modules/projects/types";
import { useProjectStore } from "@/modules/projects/stores/useProjectStore";
import { daysUntil } from "@/modules/projects/lib/dates";
import { formatCurrency, formatDate } from "@/shared/lib/formatters";
import { getApiErrorMessage } from "@/shared/lib/api-error";
import { toast } from "@/shared/stores/useToastStore";
import { ROUTE_PATHS } from "@/app/routes/route-paths";

// The one step each status offers next (FLOW-IDEAL 2.4). Moving through the
// project's life uses these buttons, not a free choice of status.
const LIFECYCLE_ACTION: Record<ProjectStatus, { action: ProjectStatusAction; label: string; done: string; primary: boolean }> = {
  Draft: { action: "start_preparation", label: "Mulai persiapan", done: "Proyek masuk tahap persiapan", primary: true },
  Preparation: { action: "mark_ready", label: "Tandai siap", done: "Proyek ditandai siap", primary: true },
  Ready: { action: "complete", label: "Selesaikan proyek", done: "Proyek diselesaikan", primary: true },
  Completed: { action: "reopen", label: "Buka kembali", done: "Proyek dibuka kembali", primary: false },
  Cancelled: { action: "reopen", label: "Buka kembali", done: "Proyek dibuka kembali", primary: false },
};

export function ProjectHeaderCard({ projectId }: { projectId: string }) {
  const navigate = useNavigate();
  const project = useProjectStore((s) => s.currentProject);
  const overview = useProjectStore((s) => s.overview);
  const updateProject = useProjectStore((s) => s.updateProject);
  const cancelProject = useProjectStore((s) => s.cancelProject);
  const transitionProject = useProjectStore((s) => s.transitionProject);
  const fetchClosingChecklist = useProjectStore((s) => s.fetchClosingChecklist);
  const setProjectArchived = useProjectStore((s) => s.setProjectArchived);
  const deleteProject = useProjectStore((s) => s.deleteProject);
  const duplicateProject = useProjectStore((s) => s.duplicateProject);
  // Actions follow the workspace role's permissions (ADR-0026).
  const canDelete = useCan("project.delete");
  const canEdit = useCan("project.edit");
  const canArchive = useCan("project.archive");
  // Neither Wedding Planner nor Sales duplicates a project (duplicating
  // creates a new one — same restriction as creating from scratch, see
  // PLAN.md's RBAC section); the backend already rejects both roles' POST
  // .../duplicate.
  const canDuplicate = useCan("project.duplicate");
  // The server sends `margin` only to roles allowed to see it (finance.margin.view), so its presence is the permission.

  const [editOpen, setEditOpen] = useState(false);
  /** `updatedAt` of the project when the edit form was opened: the save is refused if someone saved first (ADR-0036). */
  const [editVersion, setEditVersion] = useState<string | undefined>();
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

  if (!project) {
    return <div className="text-sm text-text-secondary">Proyek tidak ditemukan.</div>;
  }

  const progress = project.progress;
  // One notch per timeline item, in the order done > in progress > blocked > not started: the counts come with the project.
  const segments: Array<{ status: ProgressMeterStatus }> = progress
    ? (["Completed", "In Progress", "Blocked", "Not Started"] as const).flatMap((status) => {
        const key = { Completed: "completed", "In Progress": "inProgress", Blocked: "blocked", "Not Started": "notStarted" }[status] as "completed" | "inProgress" | "blocked" | "notStarted";
        return Array.from({ length: progress.projectMilestoneStats[key] + progress.vendorMilestoneStats[key] }, () => ({ status }));
      })
    : [];
  const totalMilestones = (progress?.projectMilestoneStats.total ?? 0) + (progress?.vendorMilestoneStats.total ?? 0);
  const completedMilestones = (progress?.projectMilestoneStats.completed ?? 0) + (progress?.vendorMilestoneStats.completed ?? 0);
  const d = daysUntil(project.eventDate);
  const isOpenProject = project.status !== "Completed" && project.status !== "Cancelled";
  const lifecycle = LIFECYCLE_ACTION[project.status];

  // The two form handlers throw on failure so the modal can show the
  // message next to the form, with what the user typed still in place.
  async function handleEdit(values: ProjectFormValues) {
    await updateProject(projectId, values, editVersion);
    setEditOpen(false);
    toast.success("Perubahan proyek disimpan");
  }

  async function handleTransition(action: ProjectStatusAction, successMessage: string) {
    setActionError(null);
    setTransitioning(true);
    try {
      await transitionProject(projectId, action);
      toast.success(successMessage);
    } catch (err) {
      setActionError(getApiErrorMessage(err, "Status proyek belum berubah. Periksa koneksi Anda, lalu coba lagi."));
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
      toast.success("Proyek diselesaikan");
    } catch (err) {
      setClosing((c) => (c ? { ...c, error: getApiErrorMessage(err, "Proyek belum bisa diselesaikan. Periksa koneksi Anda, lalu coba lagi.") } : c));
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
      setActionError(getApiErrorMessage(err, "Proyek belum bisa dibatalkan. Periksa koneksi Anda, lalu coba lagi."));
    }
  }

  async function handleToggleArchive() {
    setActionError(null);
    try {
      await setProjectArchived(projectId, !project?.isArchived);
    } catch (err) {
      setActionError(getApiErrorMessage(err, "Status arsip proyek belum bisa diubah. Periksa koneksi Anda, lalu coba lagi."));
    }
  }

  async function handleDuplicate(values: ProjectFormValues) {
    const duplicated = await duplicateProject(projectId, values);
    setDuplicateOpen(false);
    toast.success("Proyek diduplikat");
    navigate(ROUTE_PATHS.projectDetail(duplicated.id));
  }

  async function handleDeleteProject() {
    setActionError(null);
    setIsDeleting(true);
    try {
      await deleteProject(projectId);
      navigate(ROUTE_PATHS.projects);
    } catch (err) {
      setActionError(getApiErrorMessage(err, "Proyek belum bisa dihapus. Periksa koneksi Anda, lalu coba lagi."));
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
              <Button variant="secondary" size="sm" icon={<Pencil className="h-3.5 w-3.5" />} onClick={() => {
                setEditVersion(project?.updatedAt);
                setEditOpen(true);
              }}>
                Ubah proyek
              </Button>
            )}
            <ActionMenu
              items={[
                { label: "Duplikat proyek", icon: <Copy className="h-4 w-4" />, onSelect: () => setDuplicateOpen(true), hidden: !canDuplicate },
                {
                  label: project.isArchived ? "Pulihkan dari arsip" : "Arsipkan",
                  icon: project.isArchived ? <ArchiveRestore className="h-4 w-4" /> : <Archive className="h-4 w-4" />,
                  onSelect: () => void handleToggleArchive(),
                  hidden: !canArchive,
                },
                { label: "Batalkan proyek", icon: <Ban className="h-4 w-4" />, tone: "danger", onSelect: () => setConfirmingCancel(true), hidden: !(canEdit && isOpenProject) },
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

        <div className="grid grid-cols-2 gap-4 border-t border-border-light pt-4 sm:grid-cols-3 lg:grid-cols-5">
          <InfoField label="Tanggal acara" value={formatDate(project.eventDate)} />
          <InfoField
            label="Countdown"
            value={isOpenProject ? (d >= 0 ? `H-${d}` : `H+${Math.abs(d)}`) : project.status === "Completed" ? "Selesai" : "Dibatalkan"}
            emphasize
          />
          <InfoField label="Tanggal booking" value={formatDate(project.prepStartDate)} />
          <InfoField label="Paket / Layanan" value={project.packageName} />
          <InfoField label="Nilai kontrak" value={formatCurrency(project.contractValue)} />
          {overview?.finance.margin !== undefined && <InfoField label="Margin" value={formatCurrency(overview.finance.margin)} />}
        </div>

        {project.description && (
          <p className="rounded-md bg-surface-muted px-4 py-3 text-[13px] text-text-secondary">{project.description}</p>
        )}

        {progress && (
          <div className="rounded-md bg-surface-muted/60 p-4">
            <div className="mb-2 flex flex-wrap items-center justify-between gap-2">
              <div className="flex items-center gap-3">
                <span className="text-2xl font-bold leading-none tabular-nums text-primary">{progress.overallPercent}%</span>
                <ConditionBadge condition={progress.condition} />
              </div>
              <span className="text-[12.5px] text-text-secondary">
                Tahapan proyek {progress.projectMilestoneStats.completed}/{progress.projectMilestoneStats.total} · Tahapan vendor{" "}
                {progress.vendorMilestoneStats.completed}/{progress.vendorMilestoneStats.total}
                {progress.overdueMilestoneCount > 0 && (
                  <span className="font-semibold text-danger-text"> · {progress.overdueMilestoneCount} terlambat</span>
                )}
              </span>
            </div>
            <ProgressMeter
              percent={progress.overallPercent}
              segments={segments}
              caption={`${completedMilestones}/${totalMilestones} tahapan selesai berdasarkan pencapaian nyata — bukan estimasi manual.`}
            />
          </div>
        )}
      </CardContent>

      {closing && (
        <Modal
          open
          onClose={closeClosing}
          title="Selesaikan proyek"
          description="Periksa dulu apakah semuanya sudah beres. Anda tetap bisa menyelesaikan proyek meski ada yang belum tuntas."
          busy={completing}
          error={closing.error}
          footer={
            <>
              <Button variant="secondary" onClick={closeClosing} disabled={completing}>Kembali</Button>
              <Button onClick={() => void handleComplete()} disabled={completing || closing.checklist === null}>
                {completing ? "Menyelesaikan..." : closing.checklist && !closing.checklist.ready ? "Tetap selesaikan" : "Selesaikan proyek"}
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
      <ConfirmDialog
        open={confirmingCancel}
        title="Batalkan proyek ini?"
        confirmLabel="Batalkan proyek"
        error={actionError}
        onConfirm={() => void handleCancelProject()}
        onClose={() => {
          setConfirmingCancel(false);
          setActionError(null);
        }}
      >
        <p>Proyek ditandai dibatalkan dan klien melihatnya. Data yang sudah ada tidak dihapus, dan proyek bisa dibuka kembali.</p>
      </ConfirmDialog>
      <ConfirmDialog
        open={confirmingDelete}
        title="Hapus proyek ini secara permanen?"
        confirmLabel="Hapus permanen"
        confirmText={project.name}
        isSubmitting={isDeleting}
        error={actionError}
        onConfirm={() => void handleDeleteProject()}
        onClose={() => {
          setConfirmingDelete(false);
          setActionError(null);
        }}
      >
        <p>
          Seluruh jadwal, vendor, pembayaran, kendala, dan bukti proyek ini ikut terhapus dan tidak bisa dipulihkan.
        </p>
      </ConfirmDialog>
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
      <p className={emphasize ? "mt-0.5 text-[15px] font-bold tabular-nums text-primary" : "mt-0.5 text-[13.5px] font-medium text-text-primary"}>
        {value}
      </p>
    </div>
  );
}
