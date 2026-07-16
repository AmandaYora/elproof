import { useState } from "react";
import { Modal } from "@/shared/components/ui/Modal";
import { Button } from "@/shared/components/ui/Button";
import { Input, Field } from "@/shared/components/ui/Input";
import { projectMilestoneSchema, type ProjectMilestoneFormValues } from "@/modules/projects/schemas/project-milestone.schema";

interface ProjectMilestoneFormModalProps {
  open: boolean;
  onClose: () => void;
  onSubmit: (values: ProjectMilestoneFormValues) => void;
}

function emptyValues(): ProjectMilestoneFormValues {
  return { name: "", targetDate: "" };
}

export function ProjectMilestoneFormModal({ open, onClose, onSubmit }: ProjectMilestoneFormModalProps) {
  const [values, setValues] = useState<ProjectMilestoneFormValues>(emptyValues);
  const [errors, setErrors] = useState<Partial<Record<keyof ProjectMilestoneFormValues, string>>>({});

  function set<K extends keyof ProjectMilestoneFormValues>(key: K, value: ProjectMilestoneFormValues[K]) {
    setValues((prev) => ({ ...prev, [key]: value }));
  }

  function handleSubmit() {
    const result = projectMilestoneSchema.safeParse(values);
    if (!result.success) {
      const fieldErrors: Partial<Record<keyof ProjectMilestoneFormValues, string>> = {};
      for (const issue of result.error.issues) {
        fieldErrors[issue.path[0] as keyof ProjectMilestoneFormValues] = issue.message;
      }
      setErrors(fieldErrors);
      return;
    }
    onSubmit(result.data);
    setValues(emptyValues());
    setErrors({});
  }

  return (
    <Modal
      open={open}
      onClose={onClose}
      title="Tambah Milestone"
      description="Milestone baru akan ditambahkan di urutan paling akhir — gunakan tombol urutkan untuk memindahkannya."
      footer={
        <>
          <Button variant="secondary" onClick={onClose}>Batal</Button>
          <Button onClick={handleSubmit}>Simpan Milestone</Button>
        </>
      }
    >
      <div className="flex flex-col gap-4">
        <Field label="Nama Milestone" required hint={errors.name}>
          <Input value={values.name} onChange={(e) => set("name", e.target.value)} placeholder="cth. Fitting Baju Pengantin" />
        </Field>
        <Field label="Target Tanggal" required hint={errors.targetDate}>
          <Input type="date" value={values.targetDate} onChange={(e) => set("targetDate", e.target.value)} />
        </Field>
      </div>
    </Modal>
  );
}
