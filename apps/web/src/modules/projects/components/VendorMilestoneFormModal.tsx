import { useEffect, useState } from "react";
import { Modal } from "@/shared/components/ui/Modal";
import { Button } from "@/shared/components/ui/Button";
import { Input, Textarea, Select, Field } from "@/shared/components/ui/Input";
import { vendorMilestoneSchema, type VendorMilestoneFormValues } from "@/modules/projects/schemas/vendor-milestone.schema";
import { useStaffStore } from "@/modules/users/stores/useStaffStore";

interface VendorMilestoneFormModalProps {
  open: boolean;
  onClose: () => void;
  onSubmit: (values: VendorMilestoneFormValues) => void;
  vendorName: string;
}

function emptyValues(defaultStaffId = ""): VendorMilestoneFormValues {
  return { name: "", description: "", targetDate: "", picStaffId: defaultStaffId };
}

export function VendorMilestoneFormModal({ open, onClose, onSubmit, vendorName }: VendorMilestoneFormModalProps) {
  const staff = useStaffStore((s) => s.staff);
  const fetchStaff = useStaffStore((s) => s.fetchStaff);
  const [values, setValues] = useState<VendorMilestoneFormValues>(() => emptyValues());
  const [errors, setErrors] = useState<Partial<Record<keyof VendorMilestoneFormValues, string>>>({});

  useEffect(() => {
    void fetchStaff();
  }, [fetchStaff]);

  useEffect(() => {
    if (!values.picStaffId && staff.length > 0) {
      setValues((prev) => ({ ...prev, picStaffId: staff[0].id }));
    }
  }, [staff, values.picStaffId]);

  function set<K extends keyof VendorMilestoneFormValues>(key: K, value: VendorMilestoneFormValues[K]) {
    setValues((prev) => ({ ...prev, [key]: value }));
  }

  function handleSubmit() {
    const result = vendorMilestoneSchema.safeParse(values);
    if (!result.success) {
      const fieldErrors: Partial<Record<keyof VendorMilestoneFormValues, string>> = {};
      for (const issue of result.error.issues) {
        fieldErrors[issue.path[0] as keyof VendorMilestoneFormValues] = issue.message;
      }
      setErrors(fieldErrors);
      return;
    }
    onSubmit(result.data);
    setValues(emptyValues(staff[0]?.id ?? ""));
    setErrors({});
  }

  return (
    <Modal
      open={open}
      onClose={onClose}
      title="Tambah Milestone Vendor"
      description={`Milestone baru untuk ${vendorName}, ditambahkan di urutan paling akhir.`}
      footer={
        <>
          <Button variant="secondary" onClick={onClose}>Batal</Button>
          <Button onClick={handleSubmit}>Simpan Milestone</Button>
        </>
      }
    >
      <div className="flex flex-col gap-4">
        <Field label="Nama Milestone" required hint={errors.name}>
          <Input value={values.name} onChange={(e) => set("name", e.target.value)} placeholder="cth. Survey Lokasi" />
        </Field>
        <Field label="Deskripsi">
          <Textarea rows={2} value={values.description} onChange={(e) => set("description", e.target.value)} />
        </Field>
        <Field label="Target Tanggal" required hint={errors.targetDate}>
          <Input type="date" value={values.targetDate} onChange={(e) => set("targetDate", e.target.value)} />
        </Field>
        <Field label="PIC" required hint={errors.picStaffId}>
          <Select value={values.picStaffId} onChange={(e) => set("picStaffId", e.target.value)}>
            {staff.map((s) => (
              <option key={s.id} value={s.id}>{s.name} — {s.title}</option>
            ))}
          </Select>
        </Field>
      </div>
    </Modal>
  );
}
