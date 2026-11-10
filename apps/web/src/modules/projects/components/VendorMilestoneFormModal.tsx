import { useEffect, useRef } from "react";
import { Modal } from "@/shared/components/ui/Modal";
import { Button } from "@/shared/components/ui/Button";
import { Input, Textarea, Select, Field } from "@/shared/components/ui/Input";
import { vendorMilestoneSchema, type VendorMilestoneFormValues } from "@/modules/projects/schemas/vendor-milestone.schema";
import { useStaffStore } from "@/modules/users/stores/useStaffStore";
import { useZodForm } from "@/shared/forms/useZodForm";
import { FormErrorSummary } from "@/shared/forms/FormErrorSummary";

const FIELD_LABELS: Record<string, string> = { name: "Nama tahapan", targetDate: "Target tanggal", picStaffId: "PIC" };

interface VendorMilestoneFormModalProps {
  open: boolean;
  onClose: () => void;
  /** Saves, then closes the modal. Throw to keep it open: a 422 is shown on its fields, anything else inside the modal. */
  onSubmit: (values: VendorMilestoneFormValues) => Promise<void>;
  vendorName: string;
}

function emptyValues(defaultStaffId = ""): VendorMilestoneFormValues {
  return { name: "", description: "", targetDate: "", picStaffId: defaultStaffId };
}

export function VendorMilestoneFormModal({ open, onClose, onSubmit, vendorName }: VendorMilestoneFormModalProps) {
  const staff = useStaffStore((s) => s.staffSummaries);
  const fetchStaff = useStaffStore((s) => s.fetchStaffSummaries);
  const form = useZodForm({ schema: vendorMilestoneSchema, initial: emptyValues() });
  const { values, errors, set, blur } = form;
  const formRef = useRef<HTMLFormElement>(null);

  useEffect(() => {
    void fetchStaff();
  }, [fetchStaff]);

  useEffect(() => {
    if (!values.picStaffId && staff.length > 0) {
      set("picStaffId", staff[0].id);
    }
  }, [staff, values.picStaffId, set]);

  const submit = form.handleSubmit(onSubmit, { root: () => formRef.current, fallback: "Tahapan vendor belum tersimpan. Periksa koneksi Anda, lalu coba lagi." });

  return (
    <Modal
      open={open}
      onClose={onClose}
      title="Tambah tahapan vendor"
      description={`Tahapan baru untuk ${vendorName}, ditambahkan di urutan paling akhir.`}
      busy={form.submitting}
      footer={
        <>
          <Button variant="secondary" onClick={onClose} disabled={form.submitting}>Batal</Button>
          <Button type="submit" form="vendor-milestone-form" disabled={form.submitting}>
            {form.submitting ? "Menyimpan..." : "Simpan tahapan"}
          </Button>
        </>
      }
    >
      <form id="vendor-milestone-form" ref={formRef} noValidate onSubmit={(e) => void submit(e)} className="flex flex-col gap-4">
        <FormErrorSummary errors={form.errorList} formError={form.formError} labels={FIELD_LABELS} root={() => formRef.current} />
        <Field label="Nama tahapan" required name="name" error={errors.name}>
          <Input value={values.name} onChange={(e) => set("name", e.target.value)} onBlur={() => blur("name")} placeholder="cth. Survey Lokasi" />
        </Field>
        <Field label="Deskripsi" name="description" error={errors.description}>
          <Textarea rows={2} value={values.description} onChange={(e) => set("description", e.target.value)} onBlur={() => blur("description")} />
        </Field>
        <Field label="Target tanggal" required name="targetDate" error={errors.targetDate}>
          <Input type="date" value={values.targetDate} onChange={(e) => set("targetDate", e.target.value)} onBlur={() => blur("targetDate")} />
        </Field>
        <Field label="PIC" required name="picStaffId" error={errors.picStaffId}>
          <Select value={values.picStaffId} onChange={(e) => set("picStaffId", e.target.value)}>
            {staff.map((s) => (
              <option key={s.id} value={s.id}>{s.name} — {s.title}</option>
            ))}
          </Select>
        </Field>
      </form>
    </Modal>
  );
}
