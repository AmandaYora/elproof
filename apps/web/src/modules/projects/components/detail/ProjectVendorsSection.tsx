import { useEffect, useState } from "react";
import { ChevronDown, ChevronRight, Plus, Pencil, FileWarning, Ban, CheckCircle2, AlertTriangle } from "lucide-react";
import { Card, CardHeader, CardContent } from "@/shared/components/ui/Card";
import { Button } from "@/shared/components/ui/Button";
import { Badge } from "@/shared/components/ui/Badge";
import { Select } from "@/shared/components/ui/Input";
import { Pagination } from "@/shared/components/ui/Pagination";
import { usePagination } from "@/shared/hooks/usePagination";
import { MilestoneRail } from "@/shared/components/ui/MilestoneRail";
import { IconActionButton } from "@/shared/components/ui/IconActionButton";
import { EmptyState } from "@/shared/components/feedback/EmptyState";
import {
  EngagementStatusBadge,
  IssueImpactBadge,
  IssueStatusBadge,
} from "@/modules/projects/components/StatusBadges";
import { ProjectVendorFormModal } from "@/modules/projects/components/ProjectVendorFormModal";
import type { ProjectVendorFormValues } from "@/modules/projects/schemas/project-vendor.schema";
import { VendorMilestoneFormModal } from "@/modules/projects/components/VendorMilestoneFormModal";
import type { VendorMilestoneFormValues } from "@/modules/projects/schemas/vendor-milestone.schema";
import {
  VendorMilestoneEditModal,
  type MilestoneEditFields,
  type MilestoneHistoryEntry,
  type NewEvidenceMeta,
} from "@/modules/projects/components/detail/VendorMilestoneEditModal";
import { useProjectStore, type VendorMilestoneUpdateFields } from "@/modules/projects/stores/useProjectStore";
import { useVendorStore } from "@/modules/vendors/stores/useVendorStore";
import { useVendorCategoryStore } from "@/modules/vendor-categories/stores/useVendorCategoryStore";
import { useStaffStore } from "@/modules/users/stores/useStaffStore";
import { getMilestoneEvidenceCompleteness, todayISO } from "@/modules/projects/lib/dates";
import { compressFileForUpload } from "@/shared/lib/image-compression";
import type { ProjectVendor, VendorMilestone, VendorPayment, VendorIssue, MilestoneStatus } from "@/modules/projects/types";
import { getApiErrorMessage } from "@/shared/lib/api-error";
import { formatCurrency, formatDate } from "@/shared/lib/formatters";
import { cn } from "@/shared/lib/cn";

const MILESTONE_STATUS_OPTIONS: MilestoneStatus[] = ["Not Started", "In Progress", "Completed", "Blocked", "Cancelled"];

function sortVendorsForDisplay(list: ProjectVendor[]): ProjectVendor[] {
  return [...list].sort((a, b) => {
    const aCancelled = a.engagementStatus === "Cancelled";
    const bCancelled = b.engagementStatus === "Cancelled";
    if (aCancelled !== bCancelled) return aCancelled ? 1 : -1;
    return 0;
  });
}

function sortVendorMilestones(list: VendorMilestone[]): VendorMilestone[] {
  return [...list].sort((a, b) => {
    const aCancelled = a.status === "Cancelled";
    const bCancelled = b.status === "Cancelled";
    if (aCancelled !== bCancelled) return aCancelled ? 1 : -1;
    return a.order - b.order;
  });
}

export function ProjectVendorsSection({ projectId }: { projectId: string }) {
  const vendorEngagements = useProjectStore((s) => s.vendorEngagements);
  const vendorMilestones = useProjectStore((s) => s.vendorMilestones);
  const payments = useProjectStore((s) => s.payments);
  const issues = useProjectStore((s) => s.issues);
  const evidence = useProjectStore((s) => s.evidence);
  const activity = useProjectStore((s) => s.activity);
  const fetchVendorSection = useProjectStore((s) => s.fetchVendorSection);
  const fetchPayments = useProjectStore((s) => s.fetchPayments);
  const fetchIssues = useProjectStore((s) => s.fetchIssues);
  const fetchEvidence = useProjectStore((s) => s.fetchEvidence);
  const fetchActivity = useProjectStore((s) => s.fetchActivity);
  const createVendorEngagement = useProjectStore((s) => s.createVendorEngagement);
  const updateVendorEngagement = useProjectStore((s) => s.updateVendorEngagement);
  const cancelVendorEngagement = useProjectStore((s) => s.cancelVendorEngagement);
  const createVendorMilestone = useProjectStore((s) => s.createVendorMilestone);
  const updateVendorMilestone = useProjectStore((s) => s.updateVendorMilestone);
  const uploadEvidence = useProjectStore((s) => s.uploadEvidence);

  const vendors = useVendorStore((s) => s.vendors);
  const fetchVendors = useVendorStore((s) => s.fetchVendors);
  const categories = useVendorCategoryStore((s) => s.categories);
  const fetchCategories = useVendorCategoryStore((s) => s.fetchCategories);
  const staff = useStaffStore((s) => s.staff);
  const fetchStaff = useStaffStore((s) => s.fetchStaff);

  const [expanded, setExpanded] = useState<Set<string>>(new Set());
  const [addOpen, setAddOpen] = useState(false);
  const [editing, setEditing] = useState<ProjectVendor | null>(null);
  const [editingMilestone, setEditingMilestone] = useState<{ milestone: VendorMilestone; vendorName: string; total: number } | null>(null);
  const [addMilestoneFor, setAddMilestoneFor] = useState<{ projectVendorId: string; vendorName: string } | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);

  useEffect(() => {
    void fetchVendorSection(projectId);
    void fetchPayments(projectId);
    void fetchIssues(projectId);
    void fetchEvidence(projectId);
    void fetchActivity(projectId);
    void fetchVendors();
    void fetchCategories();
    void fetchStaff();
  }, [projectId, fetchVendorSection, fetchPayments, fetchIssues, fetchEvidence, fetchActivity, fetchVendors, fetchCategories, fetchStaff]);

  function toggle(id: string) {
    setExpanded((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  }

  async function handleAdd(values: ProjectVendorFormValues) {
    setActionError(null);
    try {
      await createVendorEngagement(projectId, values);
      setAddOpen(false);
    } catch (err) {
      setActionError(getApiErrorMessage(err, "Gagal menambahkan vendor"));
    }
  }

  async function handleEdit(values: ProjectVendorFormValues) {
    if (!editing) return;
    setActionError(null);
    try {
      await updateVendorEngagement(projectId, editing.id, values);
      setEditing(null);
    } catch (err) {
      setActionError(getApiErrorMessage(err, "Gagal menyimpan perubahan vendor"));
    }
  }

  async function handleCancelVendor(pvId: string) {
    setActionError(null);
    try {
      await cancelVendorEngagement(projectId, pvId);
    } catch (err) {
      setActionError(getApiErrorMessage(err, "Gagal membatalkan vendor"));
    }
  }

  async function handleAddVendorMilestone(values: VendorMilestoneFormValues) {
    if (!addMilestoneFor) return;
    setActionError(null);
    try {
      await createVendorMilestone(projectId, addMilestoneFor.projectVendorId, values);
      setAddMilestoneFor(null);
    } catch (err) {
      setActionError(getApiErrorMessage(err, "Gagal menambahkan milestone vendor"));
    }
  }

  function toMilestoneUpdateFields(m: VendorMilestone): VendorMilestoneUpdateFields {
    return {
      status: m.status,
      targetDate: m.targetDate,
      completedDate: m.completedDate ?? "",
      picStaffId: m.picStaffId,
      description: m.description,
      notes: m.notes,
    };
  }

  async function toggleVendorMilestoneCancelled(m: VendorMilestone) {
    setActionError(null);
    try {
      await updateVendorMilestone(projectId, m.projectVendorId, m.id, {
        ...toMilestoneUpdateFields(m),
        status: m.status === "Cancelled" ? "Not Started" : "Cancelled",
      });
    } catch (err) {
      setActionError(getApiErrorMessage(err, "Gagal memperbarui milestone"));
    }
  }

  async function quickStatusChange(m: VendorMilestone, status: MilestoneStatus) {
    setActionError(null);
    try {
      await updateVendorMilestone(projectId, m.projectVendorId, m.id, {
        ...toMilestoneUpdateFields(m),
        status,
        completedDate: status === "Completed" ? m.completedDate ?? todayISO() : m.completedDate ?? "",
      });
    } catch (err) {
      setActionError(getApiErrorMessage(err, "Gagal memperbarui status milestone"));
    }
  }

  async function handleSaveMilestone(fields: MilestoneEditFields) {
    if (!editingMilestone) return;
    setActionError(null);
    try {
      await updateVendorMilestone(projectId, editingMilestone.milestone.projectVendorId, editingMilestone.milestone.id, fields);
      setEditingMilestone(null);
    } catch (err) {
      setActionError(getApiErrorMessage(err, "Gagal menyimpan milestone"));
    }
  }

  async function handleAddMilestoneEvidence(file: File, meta: NewEvidenceMeta) {
    if (!editingMilestone) return;
    const compressed = await compressFileForUpload(file);
    await uploadEvidence(projectId, {
      ...compressed,
      name: meta.name,
      type: meta.type,
      documentDate: meta.documentDate,
      description: meta.description,
      relatedKind: "vendorMilestone",
      relatedId: editingMilestone.milestone.id,
    });
  }

  function historyFor(milestoneId: string): MilestoneHistoryEntry[] {
    return activity
      .filter((a) => a.entityType === "vendor_milestone" && a.entityId === milestoneId)
      .map((a) => ({ id: a.id, actorName: staff.find((s) => s.id === a.actorStaffId)?.name ?? "Sistem", timestamp: a.timestamp, text: a.description }));
  }

  function evidenceFor(milestone: VendorMilestone) {
    return evidence.filter((e) => e.relatedKind === "vendorMilestone" && e.relatedId === milestone.id);
  }

  return (
    <div id="vendor">
      <Card>
        <CardHeader
          title="Vendor Project"
          subtitle="Progress setiap vendor berdasarkan pencapaian milestone yang telah diselesaikan."
          action={
            <Button size="sm" icon={<Plus className="h-3.5 w-3.5" />} onClick={() => setAddOpen(true)}>
              Tambah Vendor
            </Button>
          }
        />
        <CardContent className="p-0">
          {actionError && (
            <p className="mx-5 mt-4 rounded-md border border-danger/30 bg-danger-soft px-3.5 py-2.5 text-[13px] font-medium text-danger">{actionError}</p>
          )}
          {vendorEngagements.length === 0 ? (
            <EmptyState title="Belum ada vendor" description="Tambahkan vendor yang terlibat pada project ini." />
          ) : (
            <ul className="divide-y divide-border-light">
              {sortVendorsForDisplay(vendorEngagements).map((pv) => {
                const vendor = vendors.find((v) => v.id === pv.vendorId) ?? null;
                return (
                  <VendorAccordionRow
                    key={pv.id}
                    pv={pv}
                    vendorName={vendor?.name ?? "Vendor tidak dikenal"}
                    categoryName={categories.find((c) => c.id === pv.categoryId)?.name ?? "-"}
                    picName={staff.find((s) => s.id === pv.picStaffId)?.name ?? "-"}
                    milestones={vendorMilestones.filter((m) => m.projectVendorId === pv.id)}
                    payments={payments.filter((p) => p.projectVendorId === pv.id)}
                    issues={issues.filter((i) => i.projectVendorId === pv.id)}
                    isOpen={expanded.has(pv.id)}
                    onToggle={() => toggle(pv.id)}
                    onEditVendor={() => setEditing(pv)}
                    onQuickStatusChange={(m, status) => void quickStatusChange(m, status)}
                    onEditMilestone={(m, vendorNameLabel, total) => setEditingMilestone({ milestone: m, vendorName: vendorNameLabel, total })}
                    onAddMilestone={() => setAddMilestoneFor({ projectVendorId: pv.id, vendorName: vendor?.name ?? "Vendor" })}
                    onToggleCancelMilestone={(m) => void toggleVendorMilestoneCancelled(m)}
                    onCancelVendor={() => void handleCancelVendor(pv.id)}
                    hasEvidence={(m) => evidenceFor(m).length > 0}
                    evidenceComplete={(m) => getMilestoneEvidenceCompleteness(m.name, evidenceFor(m))}
                  />
                );
              })}
            </ul>
          )}
        </CardContent>
      </Card>

      <ProjectVendorFormModal open={addOpen} onClose={() => setAddOpen(false)} onSubmit={(values) => void handleAdd(values)} />
      {editing && (
        <ProjectVendorFormModal
          open={Boolean(editing)}
          onClose={() => setEditing(null)}
          onSubmit={(values) => void handleEdit(values)}
          initialProjectVendor={editing}
        />
      )}
      {editingMilestone && (
        <VendorMilestoneEditModal
          open={Boolean(editingMilestone)}
          onClose={() => setEditingMilestone(null)}
          milestone={editingMilestone.milestone}
          vendorName={editingMilestone.vendorName}
          totalMilestones={editingMilestone.total}
          evidenceList={evidenceFor(editingMilestone.milestone)}
          historyEntries={historyFor(editingMilestone.milestone.id)}
          onSave={(fields) => void handleSaveMilestone(fields)}
          onAddEvidence={handleAddMilestoneEvidence}
        />
      )}
      {addMilestoneFor && (
        <VendorMilestoneFormModal
          open={Boolean(addMilestoneFor)}
          onClose={() => setAddMilestoneFor(null)}
          onSubmit={(values) => void handleAddVendorMilestone(values)}
          vendorName={addMilestoneFor.vendorName}
        />
      )}
    </div>
  );
}

interface VendorAccordionRowProps {
  pv: ProjectVendor;
  vendorName: string;
  categoryName: string;
  picName: string;
  milestones: VendorMilestone[];
  payments: VendorPayment[];
  issues: VendorIssue[];
  isOpen: boolean;
  onToggle: () => void;
  onEditVendor: () => void;
  onQuickStatusChange: (milestone: VendorMilestone, status: MilestoneStatus) => void;
  onEditMilestone: (milestone: VendorMilestone, vendorName: string, total: number) => void;
  onAddMilestone: () => void;
  onToggleCancelMilestone: (milestone: VendorMilestone) => void;
  onCancelVendor: () => void;
  hasEvidence: (milestone: VendorMilestone) => boolean;
  evidenceComplete: (milestone: VendorMilestone) => { requiresEvidence: boolean; complete: boolean };
}

function VendorAccordionRow({
  pv,
  vendorName,
  categoryName,
  picName,
  milestones,
  payments,
  issues,
  isOpen,
  onToggle,
  onEditVendor,
  onQuickStatusChange,
  onEditMilestone,
  onAddMilestone,
  onToggleCancelMilestone,
  onCancelVendor,
  hasEvidence,
  evidenceComplete,
}: VendorAccordionRowProps) {
  const [confirmingCancel, setConfirmingCancel] = useState(false);
  const remaining = pv.contractValue - pv.paidAmount;
  const sortedMilestones = sortVendorMilestones(milestones);
  const { page, setPage, totalPages, totalItems, pageSize, pageItems } = usePagination(sortedMilestones);
  const vendorCancelled = pv.engagementStatus === "Cancelled";

  const actionIcons = (
    <>
      <span
        onClick={(e) => {
          e.stopPropagation();
          onEditVendor();
        }}
        role="button"
        tabIndex={0}
        className="shrink-0 rounded-md p-1.5 text-text-secondary hover:bg-white hover:text-navy-900"
      >
        <Pencil className="h-4 w-4" />
      </span>
      {!vendorCancelled && (
        <span
          onClick={(e) => {
            e.stopPropagation();
            setConfirmingCancel(true);
          }}
          role="button"
          tabIndex={0}
          title="Batalkan vendor"
          className="shrink-0 rounded-md p-1.5 text-danger hover:bg-danger-soft"
        >
          <Ban className="h-4 w-4" />
        </span>
      )}
    </>
  );

  return (
    <li className={vendorCancelled ? "opacity-60" : undefined}>
      {/* Mobile: stacked card header — fixed-width columns don't fit one line on small screens */}
      <button
        onClick={onToggle}
        className="flex w-full flex-col gap-2.5 px-5 py-3.5 text-left transition-colors hover:bg-surface-muted sm:hidden"
      >
        <div className="flex items-start justify-between gap-3">
          <div className="flex min-w-0 items-center gap-2.5">
            {isOpen ? <ChevronDown className="h-4 w-4 shrink-0 text-text-secondary" /> : <ChevronRight className="h-4 w-4 shrink-0 text-text-secondary" />}
            <span className="min-w-0">
              <span className={cn("block truncate text-[13.5px] font-semibold text-text-primary", vendorCancelled && "line-through")}>
                {vendorName}
              </span>
              <span className="block truncate text-[12px] text-text-secondary">{categoryName} · {pv.scope}</span>
            </span>
          </div>
          <span className="flex shrink-0 items-center gap-1">{actionIcons}</span>
        </div>
        <MilestoneRail milestones={milestones} size="sm" />
        <div className="flex items-center justify-between gap-3">
          <EngagementStatusBadge status={pv.engagementStatus} />
          <span className="text-[13px] tabular-nums text-text-primary">{formatCurrency(pv.contractValue)}</span>
        </div>
      </button>

      {/* Desktop: single-line row */}
      <button
        onClick={onToggle}
        className="hidden w-full items-center gap-4 px-5 py-3.5 text-left transition-colors hover:bg-surface-muted sm:flex"
      >
        {isOpen ? <ChevronDown className="h-4 w-4 shrink-0 text-text-secondary" /> : <ChevronRight className="h-4 w-4 shrink-0 text-text-secondary" />}
        <span className="min-w-45 flex-1">
          <span className={cn("block text-[13.5px] font-semibold text-text-primary", vendorCancelled && "line-through")}>
            {vendorName}
          </span>
          <span className="block text-[12px] text-text-secondary">{categoryName} · {pv.scope}</span>
        </span>
        <span className="w-40 shrink-0">
          <MilestoneRail milestones={milestones} size="sm" />
        </span>
        <span className="w-32 shrink-0">
          <EngagementStatusBadge status={pv.engagementStatus} />
        </span>
        <span className="w-32 shrink-0 text-right text-[13px] tabular-nums text-text-primary">
          {formatCurrency(pv.contractValue)}
        </span>
        <span className="flex shrink-0 items-center gap-1">{actionIcons}</span>
      </button>

      {confirmingCancel && (
        <div className="mx-5 mb-3 flex flex-wrap items-center justify-between gap-3 rounded-md border border-danger/30 bg-danger-soft px-4 py-3">
          <span className="flex items-center gap-2 text-[13px] font-medium text-danger">
            <AlertTriangle className="h-4 w-4 shrink-0" />
            Yakin ingin membatalkan kerja sama dengan vendor ini? Milestone, pembayaran, dan kendala yang sudah tercatat tidak akan dihapus.
          </span>
          <span className="flex shrink-0 gap-2">
            <Button variant="secondary" size="sm" onClick={() => setConfirmingCancel(false)}>Batal</Button>
            <Button
              variant="danger"
              size="sm"
              onClick={() => {
                onCancelVendor();
                setConfirmingCancel(false);
              }}
            >
              Ya, Batalkan
            </Button>
          </span>
        </div>
      )}

      {isOpen && (
        <div className="bg-surface-muted/50 px-5 pb-5 pt-1">
          <div className="grid grid-cols-1 gap-3 pb-3 text-[12.5px] text-text-secondary sm:grid-cols-4">
            <span>PIC: <span className="font-medium text-text-primary">{picName}</span></span>
            <span>DP: <span className="font-medium text-text-primary">{formatCurrency(pv.dpAmount)}</span></span>
            <span>Sudah Dibayar: <span className="font-medium text-text-primary">{formatCurrency(pv.paidAmount)}</span></span>
            <span>Sisa: <span className="font-medium text-text-primary">{formatCurrency(remaining)}</span></span>
          </div>

          <div className="mb-2 flex items-center justify-between">
            <p className="text-[12px] font-semibold uppercase tracking-wide text-text-secondary">Milestone Vendor</p>
            <Button size="sm" variant="secondary" icon={<Plus className="h-3.5 w-3.5" />} onClick={onAddMilestone}>
              Tambah Milestone
            </Button>
          </div>

          {milestones.length === 0 ? (
            <p className="rounded-md border border-dashed border-border bg-white px-4 py-3 text-[13px] text-text-secondary">
              Belum ada milestone untuk vendor ini.
            </p>
          ) : (
            <div className="overflow-x-auto rounded-md border border-border bg-white">
              <table className="w-full text-left text-[13px]">
                <thead className="border-b border-border-light">
                  <tr>
                    <th className="px-3 py-2 font-semibold text-text-secondary">Milestone</th>
                    <th className="px-3 py-2 font-semibold text-text-secondary">Status</th>
                    <th className="px-3 py-2 font-semibold text-text-secondary">Target</th>
                    <th className="px-3 py-2 font-semibold text-text-secondary">Selesai</th>
                    <th className="px-3 py-2 font-semibold text-text-secondary">Evidence</th>
                    <th className="px-3 py-2 font-semibold text-text-secondary">Aksi</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-border-light">
                  {pageItems.map((m) => {
                    const evidenceCheck = evidenceComplete(m);
                    const milestoneCancelled = m.status === "Cancelled";
                    return (
                      <tr key={m.id} className={milestoneCancelled ? "opacity-50" : undefined}>
                        <td className={cn("px-3 py-2 font-medium text-text-primary", milestoneCancelled && "line-through")}>
                          {m.order}. {m.name}
                        </td>
                        <td className="px-3 py-2">
                          <Select
                            value={m.status}
                            onChange={(e) => onQuickStatusChange(m, e.target.value as MilestoneStatus)}
                            className="h-8 w-34 text-[12.5px]"
                          >
                            {MILESTONE_STATUS_OPTIONS.map((s) => (
                              <option key={s} value={s}>{s}</option>
                            ))}
                          </Select>
                        </td>
                        <td className="px-3 py-2 text-text-secondary">{formatDate(m.targetDate)}</td>
                        <td className="px-3 py-2 text-text-secondary">{formatDate(m.completedDate)}</td>
                        <td className="px-3 py-2">
                          {hasEvidence(m) ? (
                            <Badge tone="success">Evidence tersedia</Badge>
                          ) : evidenceCheck.requiresEvidence ? (
                            <Badge tone="warning">Belum lengkap</Badge>
                          ) : (
                            <span className="text-text-secondary">—</span>
                          )}
                        </td>
                        <td className="px-3 py-2">
                          <div className="flex items-center gap-1.5">
                            <IconActionButton
                              icon={Pencil}
                              label="Edit Milestone"
                              tone="neutral"
                              onClick={() => onEditMilestone(m, vendorName, milestones.length)}
                            />
                            {milestoneCancelled ? (
                              <IconActionButton
                                icon={CheckCircle2}
                                label="Aktifkan kembali"
                                tone="success"
                                onClick={() => onToggleCancelMilestone(m)}
                              />
                            ) : (
                              <IconActionButton
                                icon={Ban}
                                label="Batalkan milestone"
                                tone="danger"
                                onClick={() => onToggleCancelMilestone(m)}
                              />
                            )}
                          </div>
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
              <Pagination page={page} totalPages={totalPages} totalItems={totalItems} pageSize={pageSize} onPageChange={setPage} />
            </div>
          )}

          <div className="mt-3 grid grid-cols-1 gap-3 lg:grid-cols-2">
            <div className="rounded-md border border-border bg-white p-3">
              <p className="mb-2 text-[12px] font-semibold uppercase tracking-wide text-text-secondary">Pembayaran</p>
              {payments.length === 0 ? (
                <p className="text-[13px] text-text-secondary">Belum ada pembayaran tercatat.</p>
              ) : (
                <ul className="flex flex-col gap-1.5">
                  {payments.map((p) => (
                    <li key={p.id} className="flex items-center justify-between text-[13px]">
                      <span>{p.type} · {formatDate(p.paymentDate)}</span>
                      <span className="flex items-center gap-2 tabular-nums">
                        {formatCurrency(p.amount)}
                        {!p.evidenceComplete && <FileWarning className="h-3.5 w-3.5 text-warning" />}
                      </span>
                    </li>
                  ))}
                </ul>
              )}
            </div>
            <div className="rounded-md border border-border bg-white p-3">
              <p className="mb-2 text-[12px] font-semibold uppercase tracking-wide text-text-secondary">Kendala</p>
              {issues.length === 0 ? (
                <p className="text-[13px] text-text-secondary">Tidak ada kendala tercatat.</p>
              ) : (
                <ul className="flex flex-col gap-2">
                  {issues.map((i) => (
                    <li key={i.id} className="flex items-center justify-between gap-2 text-[13px]">
                      <span className="truncate">{i.title}</span>
                      <span className={cn("flex shrink-0 gap-1.5")}>
                        <IssueImpactBadge impact={i.impact} />
                        <IssueStatusBadge status={i.status} />
                      </span>
                    </li>
                  ))}
                </ul>
              )}
            </div>
          </div>
        </div>
      )}
    </li>
  );
}
