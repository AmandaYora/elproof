import { useRef, useState, type ReactNode } from "react";
import { useCan } from "@/shared/stores/usePermissionStore";
import { Link, useNavigate } from "react-router-dom";
import { Pencil, Archive, ArchiveRestore, Copy, Check, CircleAlert, Trash2, Ban, CalendarDays, CalendarRange, Package, UserRound, MapPin, FileCheck2, type LucideIcon } from "lucide-react";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/shared/components/kit/tooltip";
import { Breadcrumbs } from "@/shared/components/ui/Breadcrumbs";
import { Button } from "@/shared/components/ui/Button";
import { Badge } from "@/shared/components/ui/Badge";
import { ActionMenu } from "@/shared/components/ui/ActionMenu";
import { Modal } from "@/shared/components/ui/Modal";
import { ConfirmDialog } from "@/shared/components/ui/ConfirmDialog";
import { ProjectStatusBadge } from "@/modules/projects/components/StatusBadges";
import { ProjectFormModal } from "@/modules/projects/components/ProjectFormModal";
import type { ProjectFormValues } from "@/modules/projects/schemas/project.schema";
import type { ClosingChecklist, ProjectStatus, ProjectStatusAction } from "@/modules/projects/types";
import { useProjectStore } from "@/modules/projects/stores/useProjectStore";
import { daysUntil } from "@/modules/projects/lib/dates";
import { eventsLine, locationLine } from "@/modules/rincian";
import { formatDate, formatDayCountdown } from "@/shared/lib/formatters";
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
  const canViewQuotation = useCan("quotation.view");
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
  // The events as the Acara tab last loaded them: the header follows an event write without another request.
  const events = useProjectStore((s) => (s.eventsFor === projectId ? s.events : null));

  if (!project) {
    return <div className="text-sm text-text-secondary">Proyek tidak ditemukan.</div>;
  }

  const d = daysUntil(project.eventDate);
  const isOpenProject = project.status !== "Completed" && project.status !== "Cancelled";
  const lifecycle = LIFECYCLE_ACTION[project.status];
  const picName = overview?.team.picName || project.picName;
  const location = events ? locationLine(events) : project.location;
  const lineOfEvents = events ? eventsLine(events) : project.eventsLine;

  // The two form handlers throw on failure so the modal can show the
  // message next to the form, with what the user typed still in place.
  async function handleEdit(values: ProjectFormValues) {
    // The event date is not part of this form: it is the main event's, changed on the Acara & rincian tab. The
    // contract value is not either: it comes from the rincian, which says itself when the schedule no longer fits.
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
    <header className="flex flex-col gap-3">
      <Breadcrumbs items={[{ label: "Proyek", to: ROUTE_PATHS.projects }, { label: project.name }]} />
      <div className="flex flex-wrap items-center justify-between gap-x-4 gap-y-3">
        <div className="flex min-w-0 flex-wrap items-center gap-x-3 gap-y-1.5">
          <h1 className="min-w-0 truncate text-[24px] leading-tight font-semibold tracking-[-0.02em] text-foreground">{project.name}</h1>
          <ProjectStatusBadge status={project.status} />
          {project.isArchived && <Badge tone="neutral">Diarsipkan</Badge>}
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
            <Button
              variant="secondary"
              size="sm"
              aria-label="Ubah proyek"
              icon={<Pencil className="h-3.5 w-3.5" />}
              onClick={() => {
                setEditVersion(project?.updatedAt);
                setEditOpen(true);
              }}
            >
              Ubah
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

      {/* One quiet row of properties (Linear's record header): when, what, who, where. The facts in full are on Ringkasan. */}
      <ul aria-label="Data utama proyek" className="flex flex-wrap items-center gap-x-5 gap-y-1.5 text-[13px] text-muted-foreground">
        <Property icon={CalendarDays} label="Tanggal acara">
          <span className="text-foreground">{formatDate(project.eventDate)}</span>
          {isOpenProject && <span className="ml-1.5 font-semibold tabular-nums text-highlight-text">{formatDayCountdown(d)}</span>}
        </Property>
        <Property icon={Package} label="Paket" full={project.packageNames.length > 1 ? project.packageNames.join(", ") : undefined}>
          {project.packageNames.length > 0 ? project.packageNames.join(", ") : "Tanpa paket"}
        </Property>
        <Property icon={UserRound} label="PIC">
          {picName || "PIC belum ditugaskan"}
        </Property>
        {lineOfEvents && (
          <Property icon={CalendarRange} label="Acara" full={lineOfEvents}>
            {lineOfEvents}
          </Property>
        )}
        <Property icon={MapPin} label="Lokasi" full={location || undefined}>
          {location || "Venue belum dipilih"}
        </Property>
        {project.quotation && (
          <li className="flex min-w-0 max-w-full flex-wrap items-center gap-x-1.5">
            <FileCheck2 aria-hidden className="h-3.5 w-3.5 shrink-0" />
            <span>
              Dari{" "}
              {canViewQuotation ? (
                <Link to={ROUTE_PATHS.quotationDetail(project.quotation.id)} className="text-foreground underline-offset-2 hover:underline">
                  {project.quotation.number} Revisi {project.quotation.revisionN}
                </Link>
              ) : (
                <span className="text-foreground">
                  {project.quotation.number} Revisi {project.quotation.revisionN}
                </span>
              )}
            </span>
            <span aria-hidden>·</span>
            <Link to={`${ROUTE_PATHS.projectDetail(projectId, "acara-rincian")}?kesepakatan=1`} className="font-medium text-primary underline-offset-2 hover:underline">
              Lihat kesepakatan awal
            </Link>
          </li>
        )}
      </ul>

      {actionError && (
        <p className="rounded-md border border-danger/30 bg-danger-soft px-3.5 py-2.5 text-[13px] font-medium text-danger-text">{actionError}</p>
      )}

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
    </header>
  );
}

/**
 * One property of the header: an icon, a name for assistive technology, the value on one line. A value that may be
 * long (`full`) is cut at the edge and shown whole in a tooltip, which keyboard users reach with Tab.
 */
function Property({ icon: Icon, label, full, children }: { icon: LucideIcon; label: string; full?: string; children: ReactNode }) {
  return (
    <li className="flex min-w-0 max-w-full items-center gap-1.5">
      <Icon aria-hidden className="h-3.5 w-3.5 shrink-0" />
      <span className="sr-only">{label}: </span>
      {full ? (
        <Tooltip>
          <TooltipTrigger
            render={
              <span
                tabIndex={0}
                className="min-w-0 truncate rounded-sm focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-solid focus-visible:outline-ring"
              />
            }
          >
            {children}
          </TooltipTrigger>
          <TooltipContent className="max-w-sm">{full}</TooltipContent>
        </Tooltip>
      ) : (
        <span className="min-w-0 truncate">{children}</span>
      )}
    </li>
  );
}
