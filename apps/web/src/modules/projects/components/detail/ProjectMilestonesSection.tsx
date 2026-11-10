import { lazy, Suspense, useEffect, useState } from "react";
import { MILESTONE_STATUS_LABELS } from "@/shared/lib/labels";
import { Plus, Ban, CheckCircle2, Pencil, ArrowUp, ArrowDown, Paperclip, List as ListIcon, GanttChartSquare } from "lucide-react";
import { Card, CardHeader, CardContent } from "@/shared/components/ui/Card";
import { Button } from "@/shared/components/ui/Button";
import { Select } from "@/shared/components/ui/Input";
import { Table, THead, TBody, TR, TH, TD } from "@/shared/components/ui/Table";
import { CardList, CardListField } from "@/shared/components/ui/CardList";
import { Pagination } from "@/shared/components/ui/Pagination";
import { usePagination } from "@/shared/hooks/usePagination";
import { MilestoneRail, MilestoneRailLegend } from "@/shared/components/ui/MilestoneRail";
import { IconActionButton } from "@/shared/components/ui/IconActionButton";
import { EvidenceUploadModal } from "@/shared/components/ui/EvidenceUploadModal";
import { ProjectMilestoneFormModal } from "@/modules/projects/components/ProjectMilestoneFormModal";
import { ProjectMilestoneEditModal, type ProjectMilestoneEditFields } from "@/modules/projects/components/detail/ProjectMilestoneEditModal";
// The Gantt library is ~88 KB gzip and only the "Gantt" view needs it, so it loads when that view is chosen.
const ProjectMilestoneGanttView = lazy(() => import("@/modules/projects/components/detail/ProjectMilestoneGanttView").then((m) => ({ default: m.ProjectMilestoneGanttView })));
import type { ProjectMilestoneFormValues } from "@/modules/projects/schemas/project-milestone.schema";
import type { EvidenceUploadFormValues } from "@/modules/projects/schemas/evidence.schema";
import { useProjectStore } from "@/modules/projects/stores/useProjectStore";
import { useCan } from "@/shared/stores/usePermissionStore";
import { computeMilestoneStats, isMilestoneOverdue } from "@/modules/projects/lib/dates";
import { compressFileForUpload } from "@/shared/lib/image-compression";
import { getApiErrorMessage } from "@/shared/lib/api-error";
import type { ProjectMilestone, MilestoneStatus } from "@/modules/projects/types";
import { formatDate } from "@/shared/lib/formatters";
import { cn } from "@/shared/lib/cn";

const STATUS_OPTIONS: MilestoneStatus[] = ["Not Started", "In Progress", "Completed", "Blocked", "Cancelled"];

function sortMilestones(list: ProjectMilestone[]): ProjectMilestone[] {
  return [...list].sort((a, b) => {
    const aCancelled = a.status === "Cancelled";
    const bCancelled = b.status === "Cancelled";
    if (aCancelled !== bCancelled) return aCancelled ? 1 : -1;
    return a.order - b.order;
  });
}

export function ProjectMilestonesSection({ projectId }: { projectId: string }) {
  const milestones = useProjectStore((s) => s.milestones);
  const fetchMilestones = useProjectStore((s) => s.fetchMilestones);
  const createMilestone = useProjectStore((s) => s.createMilestone);
  const updateMilestoneStatus = useProjectStore((s) => s.updateMilestoneStatus);
  const updateMilestone = useProjectStore((s) => s.updateMilestone);
  const reorderMilestones = useProjectStore((s) => s.reorderMilestones);
  const uploadEvidence = useProjectStore((s) => s.uploadEvidence);
  // Changing the timeline needs project.edit; proofs may also be attached by
  // whoever records finance (ADR-0026).
  const canEdit = useCan("project.edit");
  const canRecordFinance = useCan("finance.record");
  const canAttach = canEdit || canRecordFinance;
  const [addOpen, setAddOpen] = useState(false);
  const [editingMilestone, setEditingMilestone] = useState<ProjectMilestone | null>(null);
  const [evidenceMilestone, setEvidenceMilestone] = useState<ProjectMilestone | null>(null);
  const [evidenceError, setEvidenceError] = useState<string | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);
  const [viewMode, setViewMode] = useState<"list" | "gantt">("list");
  const sortedMilestones = sortMilestones(milestones);
  const stats = computeMilestoneStats(milestones);
  const { page, setPage, totalPages, totalItems, pageSize, pageItems } = usePagination(sortedMilestones);
  // Up/down only makes sense within the active (non-cancelled) set — a
  // cancelled milestone's on-screen position is always forced to the bottom
  // (see sortMilestones above) regardless of its stored sort order, so
  // reordering buttons on it (or past it) would be confusing.
  const activeOrder = sortedMilestones.filter((m) => m.status !== "Cancelled");

  useEffect(() => {
    void fetchMilestones(projectId);
  }, [projectId, fetchMilestones]);

  async function updateStatus(id: string, status: MilestoneStatus) {
    setActionError(null);
    try {
      await updateMilestoneStatus(projectId, id, status);
    } catch (err) {
      setActionError(getApiErrorMessage(err, "Status tahapan belum bisa diperbarui. Periksa koneksi Anda, lalu coba lagi."));
    }
  }

  // Errors are not caught here: they propagate to the form, which shows a 422 on its fields.
  async function handleAddMilestone(values: ProjectMilestoneFormValues) {
    setActionError(null);
    await createMilestone(projectId, values);
    setAddOpen(false);
  }

  async function handleEditSave(fields: ProjectMilestoneEditFields) {
    if (!editingMilestone) return;
    setActionError(null);
    await updateMilestone(projectId, editingMilestone.id, fields, editingMilestone.updatedAt);
    setEditingMilestone(null);
  }

  async function handleAddEvidence(file: File, values: EvidenceUploadFormValues) {
    if (!evidenceMilestone) return;
    setEvidenceError(null);
    try {
      const compressed = await compressFileForUpload(file);
      await uploadEvidence(projectId, {
        ...compressed,
        name: values.name,
        type: values.type,
        documentDate: values.documentDate,
        description: values.description,
        relatedKind: "projectMilestone",
        relatedId: evidenceMilestone.id,
      });
      setEvidenceMilestone(null);
    } catch (err) {
      setEvidenceError(getApiErrorMessage(err, "Gagal mengunggah bukti"));
    }
  }

  async function handleMove(milestoneId: string, direction: "up" | "down") {
    setActionError(null);
    const idx = activeOrder.findIndex((m) => m.id === milestoneId);
    const swapIdx = direction === "up" ? idx - 1 : idx + 1;
    if (idx === -1 || swapIdx < 0 || swapIdx >= activeOrder.length) return;
    const reordered = [...activeOrder];
    [reordered[idx], reordered[swapIdx]] = [reordered[swapIdx], reordered[idx]];
    const cancelled = sortedMilestones.filter((m) => m.status === "Cancelled");
    try {
      await reorderMilestones(projectId, [...reordered, ...cancelled].map((m) => m.id));
    } catch (err) {
      setActionError(getApiErrorMessage(err, "Urutan tahapan belum bisa diubah. Periksa koneksi Anda, lalu coba lagi."));
    }
  }

  return (
    <div id="milestone">
      <Card>
        <CardHeader
          title="Jadwal persiapan acara"
          subtitle="Progres keseluruhan didasarkan pada tahapan yang benar-benar telah diselesaikan, bukan angka manual."
          action={
            <div className="flex items-center gap-2">
              <div className="flex items-center rounded-md border border-border p-0.5">
                <button
                  type="button"
                  onClick={() => setViewMode("list")}
                  className={cn(
                    "flex items-center gap-1.5 rounded px-2.5 py-1 text-[12.5px] font-semibold transition-colors",
                    viewMode === "list" ? "bg-navy-900 text-white" : "text-text-secondary hover:text-text-primary"
                  )}
                >
                  <ListIcon className="h-3.5 w-3.5" /> List
                </button>
                <button
                  type="button"
                  onClick={() => setViewMode("gantt")}
                  className={cn(
                    "flex items-center gap-1.5 rounded px-2.5 py-1 text-[12.5px] font-semibold transition-colors",
                    viewMode === "gantt" ? "bg-navy-900 text-white" : "text-text-secondary hover:text-text-primary"
                  )}
                >
                  <GanttChartSquare className="h-3.5 w-3.5" /> Gantt
                </button>
              </div>
              {canEdit && (
                <Button size="sm" icon={<Plus className="h-3.5 w-3.5" />} onClick={() => setAddOpen(true)}>
                  Tambah tahapan
                </Button>
              )}
            </div>
          }
        />
        <CardContent>
          {actionError && (
            <p className="mb-3 rounded-md border border-danger/30 bg-danger-soft px-3.5 py-2.5 text-[13px] font-medium text-danger-text">{actionError}</p>
          )}
          <div className="flex flex-col gap-2 border-b border-border-light pb-4">
            <MilestoneRail milestones={milestones} size="md" />
            <div className="flex flex-wrap items-center justify-between gap-2">
              <MilestoneRailLegend />
              {stats.overdue > 0 && (
                <span className="text-[12.5px] font-semibold text-danger-text">{stats.overdue} tahapan terlambat dari target</span>
              )}
            </div>
          </div>

          {viewMode === "gantt" ? (
            <div className="pt-4">
              <Suspense fallback={<div role="status" className="rounded-md border border-border px-4 py-10 text-center text-[13px] text-muted-foreground">Memuat garis waktu...</div>}>
                <ProjectMilestoneGanttView milestones={sortedMilestones} />
              </Suspense>
            </div>
          ) : (
          <>
          <CardList
            className="sm:hidden"
            items={pageItems}
            keyFor={(m) => m.id}
            renderItem={(m) => {
              const overdue = isMilestoneOverdue(m.status, m.targetDate);
              const cancelled = m.status === "Cancelled";
              const activeIdx = activeOrder.findIndex((a) => a.id === m.id);
              return (
                <>
                  <div className="flex items-start justify-between gap-3">
                    <span className={cn("font-medium text-text-primary", cancelled && "line-through")}>{m.order}. {m.name}</span>
                    <div className="flex shrink-0 items-center gap-1">
                      {canEdit && !cancelled && (
                        <>
                          <IconActionButton
                            icon={ArrowUp}
                            label="Naikkan urutan"
                            tone="neutral"
                            disabled={activeIdx <= 0}
                            onClick={() => void handleMove(m.id, "up")}
                          />
                          <IconActionButton
                            icon={ArrowDown}
                            label="Turunkan urutan"
                            tone="neutral"
                            disabled={activeIdx === -1 || activeIdx >= activeOrder.length - 1}
                            onClick={() => void handleMove(m.id, "down")}
                          />
                        </>
                      )}
                      {canEdit && <IconActionButton icon={Pencil} label="Ubah tahapan" tone="neutral" onClick={() => setEditingMilestone(m)} />}
                      {canAttach && <IconActionButton icon={Paperclip} label="Lampirkan bukti" tone="info" onClick={() => setEvidenceMilestone(m)} />}
                      {canEdit && (cancelled ? (
                        <IconActionButton
                          icon={CheckCircle2}
                          label="Aktifkan kembali"
                          tone="success"
                          onClick={() => void updateStatus(m.id, "Not Started")}
                        />
                      ) : (
                        <IconActionButton
                          icon={Ban}
                          label="Batalkan tahapan"
                          tone="danger"
                          onClick={() => void updateStatus(m.id, "Cancelled")}
                        />
                      ))}
                    </div>
                  </div>
                  <div className="flex flex-col gap-1.5">
                    <CardListField
                      label="Target tanggal"
                      value={
                        <span className={overdue ? "font-semibold text-danger-text" : undefined}>
                          {formatDate(m.targetDate)}
                          {overdue && " · terlambat"}
                        </span>
                      }
                    />
                    <CardListField label="Tanggal selesai" value={formatDate(m.completedDate)} />
                  </div>
                  <Select
                    value={m.status}
                    disabled={!canEdit}
                    onChange={(e) => void updateStatus(m.id, e.target.value as MilestoneStatus)}
                    className="h-9 w-full text-[13px]"
                  >
                    {STATUS_OPTIONS.map((s) => (
                      <option key={s} value={s}>{MILESTONE_STATUS_LABELS.staff[s]}</option>
                    ))}
                  </Select>
                </>
              );
            }}
          />
          <div className="hidden sm:block">
          <Table className="mt-2">
            <THead>
              <TR>
                <TH>Tahapan</TH>
                <TH>Status</TH>
                <TH>Target tanggal</TH>
                <TH>Tanggal selesai</TH>
                <TH>Aksi</TH>
              </TR>
            </THead>
            <TBody>
              {pageItems.map((m) => {
                const overdue = isMilestoneOverdue(m.status, m.targetDate);
                const cancelled = m.status === "Cancelled";
                const activeIdx = activeOrder.findIndex((a) => a.id === m.id);
                return (
                  <TR key={m.id} className={cancelled ? "opacity-50" : undefined}>
                    <TD className={cn("font-medium", cancelled && "line-through")}>{m.order}. {m.name}</TD>
                    <TD>
                      <Select
                        value={m.status}
                        disabled={!canEdit}
                        onChange={(e) => void updateStatus(m.id, e.target.value as MilestoneStatus)}
                        className="h-8 w-40 text-[13px]"
                      >
                        {STATUS_OPTIONS.map((s) => (
                          <option key={s} value={s}>{MILESTONE_STATUS_LABELS.staff[s]}</option>
                        ))}
                      </Select>
                    </TD>
                    <TD className={overdue ? "font-semibold text-danger-text" : undefined}>
                      {formatDate(m.targetDate)}
                      {overdue && " · terlambat"}
                    </TD>
                    <TD>{formatDate(m.completedDate)}</TD>
                    <TD>
                      <div className="flex items-center gap-1">
                        {canEdit && !cancelled && (
                          <>
                            <IconActionButton
                              icon={ArrowUp}
                              label="Naikkan urutan"
                              tone="neutral"
                              disabled={activeIdx <= 0}
                              onClick={() => void handleMove(m.id, "up")}
                            />
                            <IconActionButton
                              icon={ArrowDown}
                              label="Turunkan urutan"
                              tone="neutral"
                              disabled={activeIdx === -1 || activeIdx >= activeOrder.length - 1}
                              onClick={() => void handleMove(m.id, "down")}
                            />
                          </>
                        )}
                        {canEdit && <IconActionButton icon={Pencil} label="Ubah tahapan" tone="neutral" onClick={() => setEditingMilestone(m)} />}
                      {canAttach && <IconActionButton icon={Paperclip} label="Lampirkan bukti" tone="info" onClick={() => setEvidenceMilestone(m)} />}
                        {canEdit && (cancelled ? (
                          <IconActionButton
                            icon={CheckCircle2}
                            label="Aktifkan kembali"
                            tone="success"
                            onClick={() => void updateStatus(m.id, "Not Started")}
                          />
                        ) : (
                          <IconActionButton
                            icon={Ban}
                            label="Batalkan tahapan"
                            tone="danger"
                            onClick={() => void updateStatus(m.id, "Cancelled")}
                          />
                        ))}
                      </div>
                    </TD>
                  </TR>
                );
              })}
            </TBody>
          </Table>
          </div>
          <Pagination
            page={page}
            totalPages={totalPages}
            totalItems={totalItems}
            pageSize={pageSize}
            onPageChange={setPage}
            className="-mx-5 -mb-4 mt-1"
          />
          </>
          )}
        </CardContent>
      </Card>

      <ProjectMilestoneFormModal open={addOpen} onClose={() => setAddOpen(false)} onSubmit={handleAddMilestone} />

      {editingMilestone && (
        <ProjectMilestoneEditModal
          key={editingMilestone.id}
          open
          onClose={() => setEditingMilestone(null)}
          milestone={editingMilestone}
          onSave={handleEditSave}
        />
      )}

      {evidenceMilestone && (
        <EvidenceUploadModal
          open
          onClose={() => setEvidenceMilestone(null)}
          onSubmit={handleAddEvidence}
          error={evidenceError}
          lockedRelatedKind="projectMilestone"
          lockedRelatedId={evidenceMilestone.id}
          title="Lampirkan bukti"
          description={`Unggah bukti pendukung untuk tahapan "${evidenceMilestone.name}".`}
        />
      )}
    </div>
  );
}
