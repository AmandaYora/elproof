import { useEffect, useState } from "react";
import { Modal } from "@/shared/components/ui/Modal";
import { Button } from "@/shared/components/ui/Button";
import { Input, Textarea, Select, Field } from "@/shared/components/ui/Input";
import {
  projectVendorSchema,
  ENGAGEMENT_STATUS_OPTIONS,
  type ProjectVendorFormValues,
} from "@/modules/projects/schemas/project-vendor.schema";
import type { ProjectVendor } from "@/modules/projects/types";
import { useVendorStore } from "@/modules/vendors/stores/useVendorStore";
import { useStaffStore } from "@/modules/users/stores/useStaffStore";
import { formatCurrency } from "@/shared/lib/formatters";

interface ProjectVendorFormModalProps {
  open: boolean;
  onClose: () => void;
  onSubmit: (values: ProjectVendorFormValues) => void;
  initialProjectVendor?: ProjectVendor;
}

function toFormValues(pv?: ProjectVendor, defaults?: { vendorId: string; staffId: string }): ProjectVendorFormValues {
  if (!pv) {
    return {
      vendorId: defaults?.vendorId ?? "",
      scope: "",
      contractValue: 0,
      pricingTier: "Akad",
      engagementStatus: "Planned",
      bookingDate: "",
      dpAmount: 0,
      paidAmount: 0,
      dueDate: "",
      picStaffId: defaults?.staffId ?? "",
      notes: "",
    };
  }
  return {
    vendorId: pv.vendorId,
    scope: pv.scope,
    contractValue: pv.contractValue,
    pricingTier: pv.pricingTier,
    engagementStatus: pv.engagementStatus,
    bookingDate: pv.bookingDate ?? "",
    dpAmount: pv.dpAmount,
    paidAmount: pv.paidAmount,
    dueDate: pv.dueDate ?? "",
    picStaffId: pv.picStaffId,
    notes: pv.notes,
  };
}

export function ProjectVendorFormModal({ open, onClose, onSubmit, initialProjectVendor }: ProjectVendorFormModalProps) {
  const vendors = useVendorStore((s) => s.vendors);
  const fetchVendors = useVendorStore((s) => s.fetchVendors);
  const staff = useStaffStore((s) => s.staffSummaries);
  const fetchStaff = useStaffStore((s) => s.fetchStaffSummaries);
  const [values, setValues] = useState<ProjectVendorFormValues>(() => toFormValues(initialProjectVendor));
  const [errors, setErrors] = useState<Partial<Record<keyof ProjectVendorFormValues, string>>>({});
  // Auto-fill only ever proposes a starting number — an existing engagement
  // already has a real (possibly negotiated) value, so editing one starts
  // "touched" (auto-fill disabled) while adding a new one starts untouched.
  // Reset on every fresh open so re-opening "Tambah Vendor" after a previous
  // manual edit doesn't leave auto-fill permanently disabled.
  const [contractValueTouched, setContractValueTouched] = useState(Boolean(initialProjectVendor));

  useEffect(() => {
    void fetchVendors();
    void fetchStaff();
  }, [fetchVendors, fetchStaff]);

  useEffect(() => {
    if (open) {
      setContractValueTouched(Boolean(initialProjectVendor));
    }
  }, [open, initialProjectVendor]);

  useEffect(() => {
    if (!initialProjectVendor && !values.vendorId && vendors.length > 0) {
      setValues((prev) => ({ ...prev, vendorId: vendors[0].id }));
    }
    if (!initialProjectVendor && !values.picStaffId && staff.length > 0) {
      setValues((prev) => ({ ...prev, picStaffId: staff[0].id }));
    }
  }, [initialProjectVendor, vendors, staff, values.vendorId, values.picStaffId]);

  const selectedVendor = vendors.find((v) => v.id === values.vendorId);

  // Picking a vendor/paket proposes that vendor's own preset price as the
  // starting Nilai Kerja Sama — never overwrites a value staff already typed
  // themselves (contractValueTouched), and never fills in a price the vendor
  // hasn't set for that tier.
  useEffect(() => {
    if (contractValueTouched || !selectedVendor) return;
    const price = values.pricingTier === "Akad" ? selectedVendor.priceAkad : selectedVendor.priceAkadResepsi;
    if (price == null) return;
    setValues((prev) => (prev.contractValue === price ? prev : { ...prev, contractValue: price }));
  }, [selectedVendor, values.pricingTier, contractValueTouched]);

  function set<K extends keyof ProjectVendorFormValues>(key: K, value: ProjectVendorFormValues[K]) {
    setValues((prev) => ({ ...prev, [key]: value }));
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
        <Field label="Vendor" required hint={errors.vendorId}>
          <Select value={values.vendorId} onChange={(e) => set("vendorId", e.target.value)} disabled={Boolean(initialProjectVendor)}>
            {vendors.filter((v) => v.isActive || v.id === values.vendorId).map((v) => (
              <option key={v.id} value={v.id}>{v.city ? `${v.name} — ${v.city}` : v.name}</option>
            ))}
          </Select>
          {selectedVendor && (
            <p className="mt-1 text-[12px] text-text-secondary">
              Harga Akad: {selectedVendor.priceAkad != null ? formatCurrency(selectedVendor.priceAkad) : "belum diisi"} · Harga
              Akad+Resepsi: {selectedVendor.priceAkadResepsi != null ? formatCurrency(selectedVendor.priceAkadResepsi) : "belum diisi"}
            </p>
          )}
        </Field>
        <Field label="Paket" required>
          <Select value={values.pricingTier} onChange={(e) => set("pricingTier", e.target.value as ProjectVendorFormValues["pricingTier"])}>
            <option value="Akad">Akad Saja</option>
            <option value="AkadResepsi">Akad + Resepsi</option>
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
          <Input
            type="number"
            min={0}
            value={values.contractValue}
            onChange={(e) => {
              set("contractValue", Number(e.target.value));
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
        <Field label="Jumlah DP (Rp)" hint={errors.dpAmount}>
          <Input type="number" min={0} value={values.dpAmount} onChange={(e) => set("dpAmount", Number(e.target.value))} />
        </Field>
        <Field label="Total Sudah Dibayarkan (Rp)" hint={errors.paidAmount}>
          <Input type="number" min={0} value={values.paidAmount} onChange={(e) => set("paidAmount", Number(e.target.value))} />
        </Field>
        <div className="sm:col-span-2">
          <Field label="Catatan">
            <Textarea rows={2} value={values.notes} onChange={(e) => set("notes", e.target.value)} />
          </Field>
        </div>
      </div>
    </Modal>
  );
}
