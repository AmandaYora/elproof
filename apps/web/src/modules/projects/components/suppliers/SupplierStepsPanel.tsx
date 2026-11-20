import { lazy, Suspense, useState } from "react";
import { Pencil, Plus } from "lucide-react";
import { MILESTONE_STATUS_LABELS } from "@/shared/lib/labels";
import { Button } from "@/shared/components/ui/Button";
import { Badge } from "@/shared/components/ui/Badge";
import { Select } from "@/shared/components/ui/Input";
import { IconActionButton } from "@/shared/components/ui/IconActionButton";
import { toast } from "@/shared/stores/useToastStore";
import { getApiErrorMessage } from "@/shared/lib/api-error";
import { compressFileForUpload } from "@/shared/lib/image-compression";
import { formatDate } from "@/shared/lib/formatters";
import { cn } from "@/shared/lib/cn";
import { todayISO } from "@/modules/projects/lib/dates";
import { useProjectStore } from "@/modules/projects/stores/useProjectStore";
import { stepsOf } from "@/modules/projects/components/suppliers/supplier-helpers";
import type { MilestoneEditFields, NewEvidenceMeta } from "@/modules/projects/components/detail/VendorMilestoneEditModal";
import type { VendorMilestoneFormValues } from "@/modules/projects/schemas/vendor-milestone.schema";
import type { MilestoneStatus, ProjectSupplier, VendorMilestone } from "@/modules/projects/types";

// Opened by a click only.
const VendorMilestoneEditModal = lazy(() =>
  import("@/modules/projects/components/detail/VendorMilestoneEditModal").then((m) => ({ default: m.VendorMilestoneEditModal })),
);
const VendorMilestoneFormModal = lazy(() =>
  import("@/modules/projects/components/VendorMilestoneFormModal").then((m) => ({ default: m.VendorMilestoneFormModal })),
);

const STATUS_OPTIONS: MilestoneStatus[] = ["Not Started", "In Progress", "Completed", "Blocked", "Cancelled"];

/** "Tahapan" of one vendor or venue inside its drawer: quick status, edit (with history and proof) and add. */
export function SupplierStepsPanel({ projectId, supplier, canEdit }: { projectId: string; supplier: ProjectSupplier; canEdit: boolean }) {
  const vendorMilestones = useProjectStore((s) => s.vendorMilestones);
  const evidence = useProjectStore((s) => s.evidence);
  const milestoneHistory = useProjectStore((s) => s.milestoneHistory);
  const updateVendorMilestone = useProjectStore((s) => s.updateVendorMilestone);
  const createVendorMilestone = useProjectStore((s) => s.createVendorMilestone);
  const uploadEvidence = useProjectStore((s) => s.uploadEvidence);
  const [editing, setEditing] = useState<VendorMilestone | null>(null);
  const [adding, setAdding] = useState(false);

  const steps = stepsOf(vendorMilestones, supplier.id);
  const proofOf = (m: VendorMilestone) => evidence.filter((e) => e.relatedKind === "vendorMilestone" && e.relatedId === m.id);

  async function changeStatus(m: VendorMilestone, status: MilestoneStatus) {
    try {
      // Only the status (and the completion date when finishing); the rest of the step is not sent.
      await updateVendorMilestone(
        projectId,
        supplier.id,
        m.id,
        { status, ...(status === "Completed" && { completedDate: m.completedDate ?? todayISO() }) },
        m.updatedAt,
      );
    } catch (err) {
      toast.error(getApiErrorMessage(err, "Status tahapan belum bisa diperbarui. Periksa koneksi Anda, lalu coba lagi."));
    }
  }

  async function saveStep(fields: MilestoneEditFields) {
    if (!editing) return;
    await updateVendorMilestone(projectId, supplier.id, editing.id, fields, editing.updatedAt);
    setEditing(null);
  }

  async function addStepProof(file: File, meta: NewEvidenceMeta) {
    if (!editing) return;
    const compressed = await compressFileForUpload(file);
    await uploadEvidence(projectId, { ...compressed, ...meta, relatedKind: "vendorMilestone", relatedId: editing.id });
  }

  async function addStep(values: VendorMilestoneFormValues) {
    await createVendorMilestone(projectId, supplier.id, values);
    setAdding(false);
  }

  return (
    <section aria-labelledby="supplier-steps" className="flex flex-col gap-3">
      <div className="flex items-center justify-between gap-3">
        <h3 id="supplier-steps" className="text-[14px] font-semibold text-foreground">Tahapan</h3>
        {canEdit && (
          <Button size="sm" variant="secondary" icon={<Plus className="h-3.5 w-3.5" />} onClick={() => setAdding(true)}>
            Tambah tahapan
          </Button>
        )}
      </div>
      {steps.length === 0 ? (
        <p className="rounded-md border border-dashed border-border px-4 py-3 text-[13px] text-muted-foreground">
          Belum ada tahapan. Tambahkan langkah kerja sama, misalnya survei lokasi atau tes makanan.
        </p>
      ) : (
        <ol className="divide-y divide-border rounded-md border border-border">
          {steps.map((m) => {
            const cancelled = m.status === "Cancelled";
            const proofs = proofOf(m).length;
            return (
              <li key={m.id} className={cn("flex flex-wrap items-center gap-x-3 gap-y-2 px-3.5 py-2.5", cancelled && "opacity-60")}>
                <div className="min-w-0 flex-1 basis-48">
                  <p className={cn("truncate text-[13px] font-medium text-foreground", cancelled && "line-through")}>
                    {m.order}. {m.name}
                  </p>
                  <p className="text-[12px] text-muted-foreground">
                    Target {formatDate(m.targetDate)}
                    {m.completedDate ? `, selesai ${formatDate(m.completedDate)}` : ""}
                  </p>
                </div>
                {proofs > 0 && <Badge tone="success">{proofs} bukti</Badge>}
                {canEdit ? (
                  <Select
                    aria-label={`Status ${m.name}`}
                    value={m.status}
                    onChange={(e) => void changeStatus(m, e.target.value as MilestoneStatus)}
                    className="h-8 w-36 text-[12.5px]"
                  >
                    {STATUS_OPTIONS.map((s) => (
                      <option key={s} value={s}>{MILESTONE_STATUS_LABELS.staff[s]}</option>
                    ))}
                  </Select>
                ) : (
                  <Badge tone="neutral">{MILESTONE_STATUS_LABELS.staff[m.status]}</Badge>
                )}
                {canEdit && <IconActionButton icon={Pencil} label={`Ubah tahapan ${m.name}`} tone="neutral" onClick={() => setEditing(m)} />}
              </li>
            );
          })}
        </ol>
      )}

      {editing && (
        <Suspense fallback={null}>
          <VendorMilestoneEditModal
            open
            onClose={() => setEditing(null)}
            milestone={editing}
            vendorName={supplier.name}
            totalMilestones={steps.length}
            evidenceList={proofOf(editing)}
            historyEntries={milestoneHistory
              .filter((a) => a.entityId === editing.id)
              .map((a) => ({ id: a.id, actorName: a.actorName || "Sistem", timestamp: a.timestamp, text: a.description }))}
            onSave={saveStep}
            onAddEvidence={addStepProof}
          />
        </Suspense>
      )}
      {adding && (
        <Suspense fallback={null}>
          <VendorMilestoneFormModal open onClose={() => setAdding(false)} onSubmit={addStep} vendorName={supplier.name} />
        </Suspense>
      )}
    </section>
  );
}
