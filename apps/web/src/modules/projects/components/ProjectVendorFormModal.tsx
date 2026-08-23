import { useEffect, useState } from "react";
import { Plus, Eye } from "lucide-react";
import { Modal } from "@/shared/components/ui/Modal";
import { Button } from "@/shared/components/ui/Button";
import { Input, Textarea, Select, Field } from "@/shared/components/ui/Input";
import { CurrencyInput } from "@/shared/components/ui/CurrencyInput";
import { IconActionButton } from "@/shared/components/ui/IconActionButton";
import { EvidenceUploadModal } from "@/shared/components/ui/EvidenceUploadModal";
import { EvidenceViewerModal } from "@/shared/components/ui/EvidenceViewerModal";
import {
  projectVendorSchema,
  ENGAGEMENT_STATUS_OPTIONS,
  EVENT_HOURS_PRESETS,
  type ProjectVendorFormValues,
} from "@/modules/projects/schemas/project-vendor.schema";
import type { EvidenceUploadFormValues } from "@/modules/projects/schemas/evidence.schema";
import type { Evidence, ProjectVendor } from "@/modules/projects/types";
import { useProjectStore } from "@/modules/projects/stores/useProjectStore";
import { useVendorStore } from "@/modules/vendors/stores/useVendorStore";
import { useVendorCategoryStore } from "@/modules/vendor-categories/stores/useVendorCategoryStore";
import { useStaffStore } from "@/modules/users/stores/useStaffStore";
import { compressFileForUpload } from "@/shared/lib/image-compression";
import { getApiErrorMessage } from "@/shared/lib/api-error";
import { formatCurrency, formatDate } from "@/shared/lib/formatters";

interface ProjectVendorFormModalProps {
  projectId: string;
  open: boolean;
  onClose: () => void;
  onSubmit: (values: ProjectVendorFormValues) => void;
  initialProjectVendor?: ProjectVendor;
}

const CUSTOM_HOURS = "__custom__";

function toFormValues(pv?: ProjectVendor, defaults?: { vendorId: string; staffId: string }): ProjectVendorFormValues {
  if (!pv) {
    return {
      vendorId: defaults?.vendorId ?? "",
      categoryId: "",
      scope: "",
      contractValue: 0,
      pricingTier: "Akad",
      engagementStatus: "Planned",
      bookingDate: "",
      eventStartTime: "",
      eventEndTime: "",
      dpAmount: 0,
      dueDate: "",
      picStaffId: defaults?.staffId ?? "",
      notes: "",
    };
  }
  return {
    vendorId: pv.vendorId,
    categoryId: pv.categoryId,
    scope: pv.scope,
    contractValue: pv.contractValue,
    pricingTier: pv.pricingTier,
    engagementStatus: pv.engagementStatus,
    bookingDate: pv.bookingDate ?? "",
    eventStartTime: pv.eventStartTime ?? "",
    eventEndTime: pv.eventEndTime ?? "",
    dpAmount: pv.dpAmount,
    dueDate: pv.dueDate ?? "",
    picStaffId: pv.picStaffId,
    notes: pv.notes,
  };
}

// presetKeyFor finds which fixed preset (if any) matches the current
// start/end pair -- anything else (including both empty) falls back to
// "Custom", so a hand-typed pair round-trips instead of silently snapping to
// the nearest preset.
function presetKeyFor(start: string, end: string): string {
  const match = EVENT_HOURS_PRESETS.find((p) => p.start === start && p.end === end);
  return match ? match.label : CUSTOM_HOURS;
}

export function ProjectVendorFormModal({ projectId, open, onClose, onSubmit, initialProjectVendor }: ProjectVendorFormModalProps) {
  const vendors = useVendorStore((s) => s.vendors);
  const fetchVendors = useVendorStore((s) => s.fetchVendors);
  const categories = useVendorCategoryStore((s) => s.categories);
  const fetchCategories = useVendorCategoryStore((s) => s.fetchCategories);
  const staff = useStaffStore((s) => s.staffSummaries);
  const fetchStaff = useStaffStore((s) => s.fetchStaffSummaries);
  const evidence = useProjectStore((s) => s.evidence);
  const fetchEvidence = useProjectStore((s) => s.fetchEvidence);
  const uploadEvidence = useProjectStore((s) => s.uploadEvidence);
  const [values, setValues] = useState<ProjectVendorFormValues>(() => toFormValues(initialProjectVendor));
  const [errors, setErrors] = useState<Partial<Record<keyof ProjectVendorFormValues, string>>>({});
  const [bookingProofOpen, setBookingProofOpen] = useState(false);
  const [bookingProofError, setBookingProofError] = useState<string | null>(null);
  const [viewingEvidence, setViewingEvidence] = useState<Evidence | null>(null);
  // Auto-fill only ever proposes a starting number — an existing engagement
  // already has a real (possibly negotiated) value, so editing one starts
  // "touched" (auto-fill disabled) while adding a new one starts untouched.
  // Reset on every fresh open so re-opening "Tambah Vendor" after a previous
  // manual edit doesn't leave auto-fill permanently disabled.
  const [contractValueTouched, setContractValueTouched] = useState(Boolean(initialProjectVendor));

  useEffect(() => {
    void fetchVendors();
    void fetchCategories();
    void fetchStaff();
    if (initialProjectVendor) {
      void fetchEvidence(projectId);
    }
  }, [fetchVendors, fetchCategories, fetchStaff, fetchEvidence, projectId, initialProjectVendor]);

  useEffect(() => {
    if (open) {
      setContractValueTouched(Boolean(initialProjectVendor));
      setValues(toFormValues(initialProjectVendor));
      setErrors({});
    }
  }, [open, initialProjectVendor]);

  useEffect(() => {
    if (!initialProjectVendor && !values.vendorId && vendors.length > 0) {
      setValues((prev) => ({ ...prev, vendorId: vendors[0].id, categoryId: vendors[0].categoryId }));
    }
    if (!initialProjectVendor && !values.picStaffId && staff.length > 0) {
      setValues((prev) => ({ ...prev, picStaffId: staff[0].id }));
    }
  }, [initialProjectVendor, vendors, staff, values.vendorId, values.picStaffId]);

  const selectedVendor = vendors.find((v) => v.id === values.vendorId);
  const vendorsInCategory = vendors.filter(
    (v) => (v.isActive || v.id === values.vendorId) && (!values.categoryId || v.categoryId === values.categoryId)
  );

  // Picking a Kategori filters the Vendor dropdown above; picking a Vendor
  // always syncs categoryId back to that vendor's own category (PLAN.md
  // revisi-timeline-vendor-role-sales) -- categoryId is never a free-standing
  // choice independent of the selected vendor.
  function handleCategoryChange(categoryId: string) {
    setValues((prev) => {
      const stillValid = vendors.some((v) => v.id === prev.vendorId && v.categoryId === categoryId);
      if (stillValid) return { ...prev, categoryId };
      const firstInCategory = vendors.find((v) => v.categoryId === categoryId);
      return { ...prev, categoryId, vendorId: firstInCategory?.id ?? "" };
    });
  }

  function handleVendorChange(vendorId: string) {
    const vendor = vendors.find((v) => v.id === vendorId);
    setValues((prev) => ({ ...prev, vendorId, categoryId: vendor?.categoryId ?? prev.categoryId }));
  }

  // Picking a vendor/paket proposes that vendor's own preset price as the
  // starting Nilai Kerja Sama — never overwrites a value staff already typed
  // themselves (contractValueTouched), and never fills in a price the vendor
  // hasn't set for that tier.
  useEffect(() => {
    if (contractValueTouched || !selectedVendor) return;
    const price =
      values.pricingTier === "Akad"
        ? selectedVendor.priceAkad
        : values.pricingTier === "AkadResepsi"
          ? selectedVendor.priceAkadResepsi
          : selectedVendor.priceResepsi;
    if (price == null) return;
    setValues((prev) => (prev.contractValue === price ? prev : { ...prev, contractValue: price }));
  }, [selectedVendor, values.pricingTier, contractValueTouched]);

  function set<K extends keyof ProjectVendorFormValues>(key: K, value: ProjectVendorFormValues[K]) {
    setValues((prev) => ({ ...prev, [key]: value }));
  }

  function handleHoursPresetChange(key: string) {
    if (key === CUSTOM_HOURS) {
      setValues((prev) => ({ ...prev, eventStartTime: prev.eventStartTime, eventEndTime: prev.eventEndTime }));
      return;
    }
    const preset = EVENT_HOURS_PRESETS.find((p) => p.label === key);
    setValues((prev) => ({ ...prev, eventStartTime: preset?.start ?? "", eventEndTime: preset?.end ?? "" }));
  }

  function handleSubmit() {
    const result = projectVendorSchema.safeParse(values);
    if (!result.success) {
      const fieldErrors: Partial<Record<keyof ProjectVendorFormValues, string>> = {};
      for (const issue of result.error.issues) {
        const key = issue.path[0] as keyof ProjectVendorFormValues;
        fieldErrors[key] = issue.message;
      }
      setErrors(fieldErrors);
      return;
    }
    onSubmit(result.data);
    setErrors({});
  }

  const bookingProofs = initialProjectVendor
    ? evidence.filter((e) => e.relatedKind === "projectVendor" && e.relatedId === initialProjectVendor.id)
    : [];

  async function handleAddBookingProof(file: File, uploadValues: EvidenceUploadFormValues) {
    if (!initialProjectVendor) return;
    setBookingProofError(null);
    try {
      const compressed = await compressFileForUpload(file);
      await uploadEvidence(projectId, {
        ...compressed,
        name: uploadValues.name,
        type: uploadValues.type,
        documentDate: uploadValues.documentDate,
        description: uploadValues.description,
        relatedKind: "projectVendor",
        relatedId: initialProjectVendor.id,
      });
      setBookingProofOpen(false);
    } catch (err) {
      setBookingProofError(getApiErrorMessage(err, "Gagal mengunggah bukti booked"));
    }
  }

  const currentHoursPresetKey = presetKeyFor(values.eventStartTime, values.eventEndTime);

  return (
    <Modal
      open={open}
      onClose={onClose}
      title={initialProjectVendor ? "Ubah Vendor Project" : "Tambah Vendor ke Project"}
      description="Keterlibatan vendor pada project ini — nilai kerja sama, status, dan pembayaran khusus untuk project ini."
      size="lg"
      footer={
        <>
          <Button variant="secondary" onClick={onClose}>Batal</Button>
          <Button onClick={handleSubmit}>{initialProjectVendor ? "Simpan Perubahan" : "Tambah Vendor"}</Button>
        </>
      }
    >
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <Field label="Kategori Vendor">
          <Select value={values.categoryId} onChange={(e) => handleCategoryChange(e.target.value)} disabled={Boolean(initialProjectVendor)}>
            <option value="">Semua Kategori</option>
            {categories.map((c) => (
              <option key={c.id} value={c.id}>{c.name}</option>
            ))}
          </Select>
        </Field>
        <Field label="Vendor" required hint={errors.vendorId}>
          <Select value={values.vendorId} onChange={(e) => handleVendorChange(e.target.value)} disabled={Boolean(initialProjectVendor)}>
            {vendorsInCategory.map((v) => (
              <option key={v.id} value={v.id}>{v.city ? `${v.name} — ${v.city}` : v.name}</option>
            ))}
          </Select>
          {selectedVendor && (
            <p className="mt-1 text-[12px] text-text-secondary">
              Harga Pemberkatan: {selectedVendor.priceAkad != null ? formatCurrency(selectedVendor.priceAkad) : "belum diisi"} · Harga
              Pemberkatan+Resepsi: {selectedVendor.priceAkadResepsi != null ? formatCurrency(selectedVendor.priceAkadResepsi) : "belum diisi"} · Harga
              Resepsi Only: {selectedVendor.priceResepsi != null ? formatCurrency(selectedVendor.priceResepsi) : "belum diisi"}
            </p>
          )}
        </Field>
        <Field label="Paket" required>
          <Select value={values.pricingTier} onChange={(e) => set("pricingTier", e.target.value as ProjectVendorFormValues["pricingTier"])}>
            <option value="Akad">Pemberkatan</option>
            <option value="AkadResepsi">Pemberkatan + Resepsi</option>
            <option value="Resepsi">Resepsi Only</option>
          </Select>
        </Field>
        <Field label="Status Keterlibatan" required>
          <Select value={values.engagementStatus} onChange={(e) => set("engagementStatus", e.target.value as ProjectVendorFormValues["engagementStatus"])}>
            {ENGAGEMENT_STATUS_OPTIONS.map((s) => (
              <option key={s} value={s}>{s}</option>
            ))}
          </Select>
        </Field>
        <div className="sm:col-span-2">
          <Field label="Layanan / Scope Pekerjaan" required hint={errors.scope}>
            <Textarea rows={2} value={values.scope} onChange={(e) => set("scope", e.target.value)} placeholder="cth. Sewa ballroom + basic lighting rigging" />
          </Field>
        </div>
        <Field label="Nilai Kerja Sama (Rp)" required hint={errors.contractValue}>
          <CurrencyInput
            value={values.contractValue}
            onChange={(n) => {
              set("contractValue", n);
              setContractValueTouched(true);
            }}
          />
        </Field>
        <Field label="PIC Internal WO" required hint={errors.picStaffId}>
          <Select value={values.picStaffId} onChange={(e) => set("picStaffId", e.target.value)}>
            {staff.map((s) => (
              <option key={s.id} value={s.id}>{s.name} — {s.title}</option>
            ))}
          </Select>
        </Field>
        <Field label="Tanggal Booking">
          <Input type="date" value={values.bookingDate} onChange={(e) => set("bookingDate", e.target.value)} />
        </Field>
        <Field label="Jatuh Tempo Berikutnya">
          <Input type="date" value={values.dueDate} onChange={(e) => set("dueDate", e.target.value)} />
        </Field>
        <Field label="Jam Acara">
          <Select value={currentHoursPresetKey} onChange={(e) => handleHoursPresetChange(e.target.value)}>
            {EVENT_HOURS_PRESETS.map((p) => (
              <option key={p.label} value={p.label}>{p.label}</option>
            ))}
            <option value={CUSTOM_HOURS}>Custom</option>
          </Select>
        </Field>
        {currentHoursPresetKey === CUSTOM_HOURS && (
          <Field label="Jam Mulai - Selesai (Custom)">
            <div className="flex items-center gap-2">
              <Input type="time" value={values.eventStartTime} onChange={(e) => set("eventStartTime", e.target.value)} />
              <span className="text-text-secondary">–</span>
              <Input type="time" value={values.eventEndTime} onChange={(e) => set("eventEndTime", e.target.value)} />
            </div>
          </Field>
        )}
        <Field label="Jumlah DP (Rp)" hint={errors.dpAmount}>
          <CurrencyInput value={values.dpAmount} onChange={(n) => set("dpAmount", n)} />
        </Field>
        <div className="sm:col-span-2">
          <Field label="Catatan">
            <Textarea rows={2} value={values.notes} onChange={(e) => set("notes", e.target.value)} />
          </Field>
        </div>

        {initialProjectVendor && (
          <div className="sm:col-span-2">
            <Field label="Bukti Booked">
              <div className="flex flex-col gap-2 rounded-md border border-border p-3">
                {bookingProofError && <p className="text-[12.5px] font-medium text-danger">{bookingProofError}</p>}
                {bookingProofs.length === 0 ? (
                  <p className="text-[13px] text-text-secondary">Belum ada bukti booked yang diunggah.</p>
                ) : (
                  <ul className="flex flex-col gap-1.5">
                    {bookingProofs.map((e) => (
                      <li key={e.id} className="flex items-center justify-between gap-2 text-[13px]">
                        <span className="min-w-0 truncate">{e.name} · {formatDate(e.documentDate)}</span>
                        <IconActionButton icon={Eye} label="Lihat Bukti" tone="info" onClick={() => setViewingEvidence(e)} />
                      </li>
                    ))}
                  </ul>
                )}
                <Button
                  type="button"
                  size="sm"
                  variant="secondary"
                  icon={<Plus className="h-3.5 w-3.5" />}
                  onClick={() => setBookingProofOpen(true)}
                >
                  Tambah Bukti Booked
                </Button>
              </div>
            </Field>
          </div>
        )}
      </div>

      {bookingProofOpen && (
        <EvidenceUploadModal
          open
          onClose={() => setBookingProofOpen(false)}
          onSubmit={handleAddBookingProof}
          error={bookingProofError}
          lockedRelatedKind="projectVendor"
          lockedRelatedId={initialProjectVendor?.id}
          defaultType="Booking Proof"
          title="Tambah Bukti Booked"
          description="Unggah bukti booked untuk kerja sama vendor ini."
        />
      )}

      {viewingEvidence && (
        <EvidenceViewerModal
          open
          onClose={() => setViewingEvidence(null)}
          projectId={projectId}
          evidence={viewingEvidence}
          contextLabel="Bukti Booked"
        />
      )}
    </Modal>
  );
}
