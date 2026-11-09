import { useEffect, useRef, useState } from "react";
import { MILESTONE_STATUS_LABELS } from "@/shared/lib/labels";
import { Plus, FileText } from "lucide-react";
import { Modal } from "@/shared/components/ui/Modal";
import { Button } from "@/shared/components/ui/Button";
import { Badge } from "@/shared/components/ui/Badge";
import { Input, Textarea, Select, Field } from "@/shared/components/ui/Input";
import { EmptyState } from "@/shared/components/feedback/EmptyState";
import { EVIDENCE_TYPE_OPTIONS } from "@/modules/projects/schemas/evidence.schema";
import { vendorMilestoneEditSchema } from "@/modules/projects/schemas/vendor-milestone.schema";
import { useStaffStore } from "@/modules/users/stores/useStaffStore";
import type { Evidence, EvidenceType, MilestoneStatus, VendorMilestone } from "@/modules/projects/types";
import { todayISO } from "@/modules/projects/lib/dates";
import { formatDate, formatDateTime } from "@/shared/lib/formatters";
import { useZodForm } from "@/shared/forms/useZodForm";
import { FormErrorSummary } from "@/shared/forms/FormErrorSummary";

const FIELD_LABELS: Record<string, string> = {
  status: "Status", picStaffId: "PIC", targetDate: "Target tanggal", completedDate: "Tanggal selesai", description: "Deskripsi", notes: "Catatan",
};

const STATUS_OPTIONS: MilestoneStatus[] = ["Not Started", "In Progress", "Completed", "Blocked", "Cancelled"];

export interface MilestoneHistoryEntry {
  id: string;
  actorName: string;
  timestamp: string;
  text: string;
}

export interface MilestoneEditFields {
  status: MilestoneStatus;
  targetDate: string;
  completedDate: string;
  picStaffId: string;
  description: string;
  notes: string;
}

export interface NewEvidenceMeta {
  name: string;
  type: EvidenceType;
  documentDate: string;
  description: string;
}

interface VendorMilestoneEditModalProps {
  open: boolean;
  onClose: () => void;
  milestone: VendorMilestone;
  vendorName: string;
  totalMilestones: number;
  evidenceList: Evidence[];
  historyEntries: MilestoneHistoryEntry[];
  /** Saves, then closes the modal. Throw to keep it open: a 422 is shown on its fields, anything else inside the modal. */
  onSave: (fields: MilestoneEditFields) => Promise<void>;
  onAddEvidence: (file: File, meta: NewEvidenceMeta) => Promise<void>;
}

export function VendorMilestoneEditModal({
  open,
  onClose,
  milestone,
  vendorName,
  totalMilestones,
  evidenceList,
  historyEntries,
  onSave,
  onAddEvidence,
}: VendorMilestoneEditModalProps) {
  const staff = useStaffStore((s) => s.staffSummaries);
  const fetchStaff = useStaffStore((s) => s.fetchStaffSummaries);
  const form = useZodForm({
    schema: vendorMilestoneEditSchema,
    initial: {
      status: milestone.status,
      targetDate: milestone.targetDate,
      completedDate: milestone.completedDate ?? "",
      picStaffId: milestone.picStaffId,
      description: milestone.description,
      notes: milestone.notes,
    },
  });
  const { values: fields, errors, set, blur } = form;
  const formRef = useRef<HTMLFormElement>(null);
  const [showEvidenceForm, setShowEvidenceForm] = useState(false);
  const [evidenceForm, setEvidenceForm] = useState({ name: "", type: "Document" as EvidenceType, documentDate: "", description: "" });
  const [file, setFile] = useState<File | null>(null);
  const [uploading, setUploading] = useState(false);
  const [uploadError, setUploadError] = useState<string | null>(null);

  useEffect(() => {
    void fetchStaff();
  }, [fetchStaff]);

  function setStatus(status: MilestoneStatus) {
    set("status", status);
    if (status === "Completed" && !fields.completedDate) set("completedDate", todayISO());
  }

  const submit = form.handleSubmit(onSave, { root: () => formRef.current, fallback: "Timeline belum tersimpan. Periksa koneksi lalu coba lagi." });

  async function handleAddEvidence() {
    if (!file || !evidenceForm.name || !evidenceForm.documentDate) return;
    setUploading(true);
    setUploadError(null);
    try {
      await onAddEvidence(file, evidenceForm);
      setEvidenceForm({ name: "", type: "Document", documentDate: "", description: "" });
      setFile(null);
      setShowEvidenceForm(false);
    } catch {
      setUploadError("Gagal mengunggah evidence.");
    } finally {
      setUploading(false);
    }
  }

  return (
    <Modal
      open={open}
      onClose={onClose}
      title={milestone.name}
      description={`${vendorName} · Timeline ${milestone.order} dari ${totalMilestones}`}
      size="lg"
      busy={form.submitting}
      footer={
        <>
          <Button variant="secondary" onClick={onClose} disabled={form.submitting}>Batal</Button>
          <Button type="submit" form="vendor-milestone-edit-form" disabled={form.submitting}>
            {form.submitting ? "Menyimpan..." : "Simpan perubahan"}
          </Button>
        </>
      }
    >
      <div className="flex flex-col gap-6">
        <form id="vendor-milestone-edit-form" ref={formRef} noValidate onSubmit={(e) => void submit(e)} className="flex flex-col gap-6">
        <FormErrorSummary errors={form.errorList} formError={form.formError} labels={FIELD_LABELS} root={() => formRef.current} />
        <section>
          <p className="mb-3 text-[12px] font-semibold uppercase tracking-wide text-text-secondary">Status &amp; Jadwal</p>
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <Field label="Status" name="status" error={errors.status}>
              <Select value={fields.status} onChange={(e) => setStatus(e.target.value as MilestoneStatus)}>
                {STATUS_OPTIONS.map((s) => (
                  <option key={s} value={s}>{MILESTONE_STATUS_LABELS.staff[s]}</option>
                ))}
              </Select>
            </Field>
            <Field label="PIC" name="picStaffId" error={errors.picStaffId}>
              <Select value={fields.picStaffId} onChange={(e) => set("picStaffId", e.target.value)}>
                {staff.map((s) => (
                  <option key={s.id} value={s.id}>{s.name} — {s.title}</option>
                ))}
              </Select>
            </Field>
            <Field label="Target Tanggal" name="targetDate" error={errors.targetDate}>
              <Input type="date" value={fields.targetDate} onChange={(e) => set("targetDate", e.target.value)} onBlur={() => blur("targetDate")} />
            </Field>
            <Field label="Tanggal Selesai" name="completedDate" error={errors.completedDate}>
              <Input type="date" value={fields.completedDate} onChange={(e) => set("completedDate", e.target.value)} onBlur={() => blur("completedDate")} />
            </Field>
          </div>
        </section>

        <section>
          <p className="mb-3 text-[12px] font-semibold uppercase tracking-wide text-text-secondary">Deskripsi &amp; Catatan</p>
          <div className="flex flex-col gap-4">
            <Field label="Deskripsi" name="description" error={errors.description}>
              <Textarea rows={2} value={fields.description} onChange={(e) => set("description", e.target.value)} onBlur={() => blur("description")} />
            </Field>
            <Field label="Catatan" name="notes" error={errors.notes}>
              <Textarea rows={2} value={fields.notes} onChange={(e) => set("notes", e.target.value)} onBlur={() => blur("notes")} placeholder="Tambahkan catatan perkembangan timeline ini..." />
            </Field>
          </div>
        </section>
        </form>

        <section>
          <div className="mb-3 flex items-center justify-between">
            <p className="text-[12px] font-semibold uppercase tracking-wide text-text-secondary">Bukti</p>
            <Button size="sm" variant="secondary" icon={<Plus className="h-3.5 w-3.5" />} onClick={() => setShowEvidenceForm((v) => !v)}>
              Tambah Bukti
            </Button>
          </div>

          {showEvidenceForm && (
            <div className="mb-3 flex flex-col gap-3 rounded-md border border-border bg-surface-muted/60 p-3">
              {uploadError && <p className="text-[12.5px] font-medium text-danger-text">{uploadError}</p>}
              <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
                <Field label="Berkas" required>
                  <input
                    type="file"
                    accept="image/jpeg,image/png,application/pdf"
                    onChange={(e) => setFile(e.target.files?.[0] ?? null)}
                    className="block w-full text-[13px] text-text-secondary file:mr-3 file:rounded-md file:border-0 file:bg-navy-900 file:px-3 file:py-1.5 file:text-[12.5px] file:font-semibold file:text-white"
                  />
                </Field>
                <Field label="Nama Bukti" required>
                  <Input value={evidenceForm.name} onChange={(e) => setEvidenceForm((p) => ({ ...p, name: e.target.value }))} placeholder="cth. Invoice DP" />
                </Field>
                <Field label="Jenis">
                  <Select value={evidenceForm.type} onChange={(e) => setEvidenceForm((p) => ({ ...p, type: e.target.value as EvidenceType }))}>
                    {EVIDENCE_TYPE_OPTIONS.map((t) => (
                      <option key={t} value={t}>{t}</option>
                    ))}
                  </Select>
                </Field>
                <Field label="Tanggal Dokumen" required>
                  <Input type="date" value={evidenceForm.documentDate} onChange={(e) => setEvidenceForm((p) => ({ ...p, documentDate: e.target.value }))} />
                </Field>
                <Field label="Deskripsi">
                  <Input value={evidenceForm.description} onChange={(e) => setEvidenceForm((p) => ({ ...p, description: e.target.value }))} />
                </Field>
              </div>
              <div className="flex justify-end gap-2">
                <Button size="sm" variant="secondary" onClick={() => setShowEvidenceForm(false)}>Batal</Button>
                <Button size="sm" onClick={() => void handleAddEvidence()} disabled={uploading || !file}>
                  {uploading ? "Mengunggah..." : "Simpan Bukti"}
                </Button>
              </div>
            </div>
          )}

          {evidenceList.length === 0 ? (
            <p className="rounded-md border border-dashed border-border px-4 py-3 text-[13px] text-text-secondary">
              Belum ada evidence untuk timeline ini.
            </p>
          ) : (
            <ul className="flex flex-col gap-2">
              {evidenceList.map((e) => (
                <li key={e.id} className="flex items-center gap-3 rounded-md border border-border px-3 py-2 text-[13px]">
                  <FileText className="h-4 w-4 shrink-0 text-text-secondary" />
                  <span className="min-w-0 flex-1 truncate font-medium text-text-primary">{e.name}</span>
                  <Badge tone="neutral">{e.type}</Badge>
                  <span className="shrink-0 text-text-secondary">{formatDate(e.documentDate)}</span>
                </li>
              ))}
            </ul>
          )}
        </section>

        <section>
          <p className="mb-3 text-[12px] font-semibold uppercase tracking-wide text-text-secondary">Riwayat Perubahan</p>
          {historyEntries.length === 0 ? (
            <EmptyState title="Belum ada riwayat" description="Perubahan pada timeline ini akan tercatat di sini." />
          ) : (
            <ul className="flex flex-col">
              {historyEntries.map((h, idx) => (
                <li key={h.id} className="flex gap-3">
                  <div className="flex flex-col items-center">
                    <span className="mt-1.5 h-2 w-2 shrink-0 rounded-full bg-navy-900" />
                    {idx !== historyEntries.length - 1 && <span className="w-px flex-1 bg-border" />}
                  </div>
                  <div className="pb-4">
                    <p className="text-[13px] text-text-primary">
                      <strong className="font-semibold">{h.actorName}</strong> {h.text}
                    </p>
                    <p className="mt-0.5 text-[12px] text-text-secondary">{formatDateTime(h.timestamp)}</p>
                  </div>
                </li>
              ))}
            </ul>
          )}
        </section>
      </div>
    </Modal>
  );
}
