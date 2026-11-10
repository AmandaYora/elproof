import { useState } from "react";
import { Modal } from "@/shared/components/ui/Modal";
import { Button } from "@/shared/components/ui/Button";
import { Input, Textarea, Select, Field } from "@/shared/components/ui/Input";
import {
  evidenceUploadSchema,
  EVIDENCE_TYPE_OPTIONS,
  EVIDENCE_RELATED_KIND_OPTIONS,
  type EvidenceUploadFormValues,
} from "@/modules/projects/schemas/evidence.schema";
import { todayISO } from "@/modules/projects/lib/dates";
import type { EvidenceRelatedKind, EvidenceType } from "@/modules/projects/types";
import { evidenceTypeLabel } from "@/shared/lib/labels";

const RELATED_KIND_LABEL: Record<EvidenceRelatedKind, string> = {
  vendorMilestone: "Tahapan vendor",
  payment: "Pembayaran",
  projectVendor: "Kerja sama vendor",
  issue: "Kendala",
  clientPayment: "Pembayaran klien",
  venuePayment: "Pembayaran venue",
  general: "Umum (tidak terkait spesifik)",
  projectMilestone: "Tahapan",
};

interface EvidenceUploadModalProps {
  open: boolean;
  onClose: () => void;
  onSubmit: (file: File, values: EvidenceUploadFormValues) => Promise<void>;
  error: string | null;
  // Omitted when lockedRelatedKind is set — the "Konteks"/"Terkait" fields
  // are hidden entirely in that case, so there's nothing to look options up
  // for.
  relatedOptionsFor?: (kind: EvidenceRelatedKind) => { id: string; label: string }[];
  // When both are set, "Konteks" and "Terkait" are hidden and the upload is
  // pinned to this relatedKind/relatedId — used by callers that already know
  // exactly what they're attaching evidence to (Timeline, bukti booked on a
  // vendor engagement), see PLAN.md revisi-timeline-vendor-role-sales.
  lockedRelatedKind?: EvidenceRelatedKind;
  lockedRelatedId?: string;
  defaultType?: EvidenceType;
  title?: string;
  description?: string;
  /** Offer "Tampilkan dokumen ini ke portal klien"; off for a role that may add proofs but not share them (ADR-0026). */
  canShareWithClient?: boolean;
}

function emptyValues(defaultRelatedId: string, defaultType?: EvidenceType): EvidenceUploadFormValues {
  return {
    name: "",
    type: defaultType ?? "Document",
    relatedKind: "vendorMilestone",
    relatedId: defaultRelatedId,
    documentDate: todayISO(),
    description: "",
    isClientVisible: false,
  };
}

// Shared upload form for project-scoped evidence — extracted from
// ProjectEvidenceSection's own AddEvidenceModal (PLAN.md
// revisi-timeline-vendor-role-sales) so Timeline's "Lampirkan Bukti" and
// vendor engagement's "Bukti Booked" reuse the same file-compress +
// validation flow instead of duplicating it.
export function EvidenceUploadModal({
  open,
  onClose,
  onSubmit,
  error,
  relatedOptionsFor,
  lockedRelatedKind,
  lockedRelatedId,
  defaultType,
  canShareWithClient = true,
  title = "Tambah dokumen",
  description = "Unggah dokumen pendukung dan kaitkan dengan jadwal, pembayaran, kerja sama vendor, atau kendala pada proyek ini.",
}: EvidenceUploadModalProps) {
  const locked = lockedRelatedKind !== undefined;
  const [values, setValues] = useState<EvidenceUploadFormValues>(() =>
    locked
      ? { ...emptyValues(lockedRelatedId ?? "", defaultType), relatedKind: lockedRelatedKind, relatedId: lockedRelatedId ?? "" }
      : emptyValues(relatedOptionsFor?.("vendorMilestone")[0]?.id ?? "", defaultType)
  );
  const [file, setFile] = useState<File | null>(null);
  const [errors, setErrors] = useState<Partial<Record<keyof EvidenceUploadFormValues, string>>>({});
  const [fileError, setFileError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  const options = locked ? [] : relatedOptionsFor?.(values.relatedKind) ?? [];

  function set<K extends keyof EvidenceUploadFormValues>(key: K, value: EvidenceUploadFormValues[K]) {
    setValues((prev) => ({ ...prev, [key]: value }));
  }

  function handleRelatedKindChange(kind: EvidenceRelatedKind) {
    const firstOption = relatedOptionsFor?.(kind)[0]?.id ?? "";
    setValues((prev) => ({
      ...prev,
      relatedKind: kind,
      relatedId: firstOption,
      isClientVisible: kind === "general" ? prev.isClientVisible : false,
    }));
  }

  async function handleSubmit() {
    if (!file) {
      setFileError("Berkas wajib dipilih");
      return;
    }
    const result = evidenceUploadSchema.safeParse(values);
    if (!result.success) {
      const fieldErrors: Partial<Record<keyof EvidenceUploadFormValues, string>> = {};
      for (const issue of result.error.issues) {
        fieldErrors[issue.path[0] as keyof EvidenceUploadFormValues] = issue.message;
      }
      setErrors(fieldErrors);
      return;
    }
    setSubmitting(true);
    try {
      await onSubmit(file, result.data);
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <Modal
      open={open}
      onClose={onClose}
      title={title}
      description={description}
      footer={
        <>
          <Button variant="secondary" onClick={onClose}>Batal</Button>
          <Button onClick={() => void handleSubmit()} disabled={submitting || (!locked && values.relatedKind !== "general" && options.length === 0)}>
            {submitting ? "Mengunggah..." : "Simpan bukti"}
          </Button>
        </>
      }
    >
      <div className="flex flex-col gap-4">
        {error && <p className="text-[12.5px] font-medium text-danger-text">{error}</p>}
        <Field label="Berkas" required hint={fileError ?? undefined}>
          <input
            type="file"
            accept="image/jpeg,image/png,application/pdf"
            onChange={(e) => {
              setFile(e.target.files?.[0] ?? null);
              setFileError(null);
            }}
            className="block w-full text-[13px] text-text-secondary file:mr-3 file:rounded-md file:border-0 file:bg-primary file:px-3 file:py-1.5 file:text-[12.5px] file:font-semibold file:text-primary-foreground"
          />
        </Field>
        <Field label="Nama bukti" required hint={errors.name}>
          <Input value={values.name} onChange={(e) => set("name", e.target.value)} placeholder="cth. Tagihan DP venue" />
        </Field>
        <Field label="Jenis dokumen" required>
          <Select value={values.type} onChange={(e) => set("type", e.target.value as EvidenceType)}>
            {EVIDENCE_TYPE_OPTIONS.map((t) => (
              <option key={t} value={t}>{evidenceTypeLabel(t)}</option>
            ))}
          </Select>
        </Field>
        {!locked && (
          <>
            <Field label="Konteks">
              <Select value={values.relatedKind} onChange={(e) => handleRelatedKindChange(e.target.value as EvidenceRelatedKind)}>
                {EVIDENCE_RELATED_KIND_OPTIONS.map((k) => (
                  <option key={k} value={k}>{RELATED_KIND_LABEL[k]}</option>
                ))}
              </Select>
            </Field>
            {values.relatedKind !== "general" && (
              <Field label="Terkait" required hint={errors.relatedId}>
                <Select value={values.relatedId} onChange={(e) => set("relatedId", e.target.value)}>
                  {options.length === 0 && <option value="">Tidak ada data tersedia</option>}
                  {options.map((o) => (
                    <option key={o.id} value={o.id}>{o.label}</option>
                  ))}
                </Select>
              </Field>
            )}
          </>
        )}
        <Field label="Tanggal dokumen" required hint={errors.documentDate}>
          <Input type="date" value={values.documentDate} onChange={(e) => set("documentDate", e.target.value)} />
        </Field>
        <Field label="Deskripsi">
          <Textarea rows={2} value={values.description} onChange={(e) => set("description", e.target.value)} />
        </Field>
        {canShareWithClient && !locked && values.relatedKind === "general" && (
          <label className="flex items-center gap-2 rounded-md border border-border px-3 py-2 text-[13px] text-text-secondary">
            <input
              type="checkbox"
              checked={values.isClientVisible}
              onChange={(e) => set("isClientVisible", e.target.checked)}
            />
            Tampilkan dokumen ini ke portal klien
          </label>
        )}
      </div>
    </Modal>
  );
}
