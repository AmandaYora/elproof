import { useEffect, useMemo, useRef } from "react";
import { z } from "zod";
import { Modal } from "@/shared/components/ui/Modal";
import { Button } from "@/shared/components/ui/Button";
import { Select, Field } from "@/shared/components/ui/Input";
import { useZodForm } from "@/shared/forms/useZodForm";
import { FormErrorSummary } from "@/shared/forms/FormErrorSummary";
import { useCan } from "@/shared/stores/usePermissionStore";
import { useAuthStore } from "@/shared/stores/useAuthStore";
import { useProjectStore } from "@/modules/projects/stores/useProjectStore";
import { useVendorStore } from "@/modules/vendors/stores/useVendorStore";
import { useStaffStore } from "@/modules/users/stores/useStaffStore";
import { SupplierFields, SUPPLIER_FIELD_LABELS, supplierSchema } from "@/modules/rincian";

// "Tambah vendor" picks from the vendor directory; a venue comes in through an event (Acara & rincian).
const addVendorSchema = supplierSchema.extend({ vendorId: z.string().min(1, "Pilih vendor dari direktori") });

interface Props {
  projectId: string;
  onClose: () => void;
  /** Called after the vendor was added (the list is refetched by the store). */
  onAdded?: () => void;
}

/** "Tambah vendor": a vendor from the directory, who pays it and what the WO agreed to pay (with the margin permission). */
export function AddVendorModal({ projectId, onClose, onAdded }: Props) {
  const suppliers = useProjectStore((s) => s.suppliers);
  const createSupplier = useProjectStore((s) => s.createSupplier);
  const vendors = useVendorStore((s) => s.vendors);
  const fetchVendors = useVendorStore((s) => s.fetchVendors);
  const staff = useStaffStore((s) => s.staffSummaries);
  const fetchStaff = useStaffStore((s) => s.fetchStaffSummaries);
  const currentStaffId = useAuthStore((s) => s.currentStaffId);
  const canMargin = useCan("finance.margin.view");
  const formRef = useRef<HTMLFormElement>(null);

  useEffect(() => {
    void fetchVendors();
    void fetchStaff();
  }, [fetchVendors, fetchStaff]);

  // Active directory vendors not in this project yet (a vendor is engaged once per project).
  const choices = useMemo(() => {
    const taken = new Set(suppliers.filter((s) => s.kind === "vendor").map((s) => s.supplierId));
    return vendors.filter((v) => v.isActive && !taken.has(v.id)).sort((a, b) => a.name.localeCompare(b.name, "id"));
  }, [vendors, suppliers]);

  const form = useZodForm({
    schema: addVendorSchema,
    initial: {
      vendorId: "",
      paidBy: "WO",
      showName: true,
      agreedCost: "",
      engagementStatus: "Planned",
      bookingDate: "",
      dueDate: "",
      picStaffId: "",
      scope: "",
      notes: "",
    },
  });
  const { values, errors, set, blur } = form;

  // PIC starts as the person adding it when they are on the team list, else the first one.
  useEffect(() => {
    if (values.picStaffId || staff.length === 0) return;
    set("picStaffId", staff.some((s) => s.id === currentStaffId) ? currentStaffId : staff[0].id);
  }, [staff, currentStaffId, values.picStaffId, set]);

  const submit = form.handleSubmit(
    async (parsed) => {
      await createSupplier(projectId, parsed, canMargin);
      onAdded?.();
      onClose();
    },
    { root: () => formRef.current, fallback: "Vendor belum ditambahkan. Periksa koneksi Anda, lalu coba lagi.", known: (key) => key in values },
  );

  const chosen = vendors.find((v) => v.id === values.vendorId);

  return (
    <Modal
      open
      onClose={onClose}
      title="Tambah vendor"
      description="Pilih vendor dari direktori. Venue masuk lewat acara di tab Acara & rincian."
      size="lg"
      busy={form.submitting}
      footer={
        <>
          <Button variant="secondary" onClick={onClose} disabled={form.submitting}>Batal</Button>
          <Button type="submit" form="add-vendor-form" disabled={form.submitting}>
            {form.submitting ? "Menyimpan..." : "Tambah vendor"}
          </Button>
        </>
      }
    >
      <form id="add-vendor-form" ref={formRef} noValidate onSubmit={(e) => void submit(e)} className="flex flex-col gap-5">
        <FormErrorSummary errors={form.errorList} formError={form.formError} labels={SUPPLIER_FIELD_LABELS} root={() => formRef.current} />
        <Field
          label="Vendor"
          required
          name="vendorId"
          error={errors.vendorId}
          hint={
            choices.length === 0 && vendors.length > 0
              ? "Semua vendor aktif di direktori sudah ada di proyek ini."
              : chosen
                ? [chosen.categoryName, chosen.city].filter(Boolean).join(", ") || undefined
                : undefined
          }
        >
          <Select aria-label="Vendor" placeholder="Pilih vendor" value={values.vendorId} onChange={(e) => set("vendorId", e.target.value)}>
            {choices.map((v) => (
              <option key={v.id} value={v.id}>{v.name}</option>
            ))}
          </Select>
        </Field>
        <SupplierFields values={values} errors={errors} set={set} blur={blur} canMargin={canMargin} staff={staff} />
      </form>
    </Modal>
  );
}
