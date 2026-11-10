import { useEffect, useRef } from "react";
import { Modal } from "@/shared/components/ui/Modal";
import { Button } from "@/shared/components/ui/Button";
import { Input, Field } from "@/shared/components/ui/Input";
import { projectMilestoneSchema, type ProjectMilestoneFormValues } from "@/modules/projects/schemas/project-milestone.schema";
import { useZodForm } from "@/shared/forms/useZodForm";
import { FormErrorSummary } from "@/shared/forms/FormErrorSummary";

const FIELD_LABELS: Record<string, string> = { name: "Nama tahapan", targetDate: "Target tanggal" };

interface ProjectMilestoneFormModalProps {
  open: boolean;
  onClose: () => void;
  /** Saves, then closes the modal. Throw to keep it open: a 422 is shown on its fields, anything else inside the modal. */
  onSubmit: (values: ProjectMilestoneFormValues) => Promise<void>;
}

function emptyValues(): ProjectMilestoneFormValues {
  return { name: "", targetDate: "" };
}

export function ProjectMilestoneFormModal({ open, onClose, onSubmit }: ProjectMilestoneFormModalProps) {
  const form = useZodForm({ schema: projectMilestoneSchema, initial: emptyValues() });
  const { values, errors, set, blur } = form;
  const formRef = useRef<HTMLFormElement>(null);

  // Mounted once and kept alive while closed: every opening starts empty.
  useEffect(() => {
    form.reset(emptyValues());
  }, [open]);

  const submit = form.handleSubmit(onSubmit, { root: () => formRef.current, fallback: "Tahapan belum tersimpan. Periksa koneksi Anda, lalu coba lagi." });

  return (
    <Modal
      open={open}
      onClose={onClose}
      title="Tambah tahapan"
      description="Tahapan baru ditambahkan di urutan paling akhir — gunakan tombol urutkan untuk memindahkannya."
      busy={form.submitting}
      footer={
        <>
          <Button variant="secondary" onClick={onClose} disabled={form.submitting}>Batal</Button>
          <Button type="submit" form="project-milestone-form" disabled={form.submitting}>
            {form.submitting ? "Menyimpan..." : "Simpan tahapan"}
          </Button>
        </>
      }
    >
      <form id="project-milestone-form" ref={formRef} noValidate onSubmit={(e) => void submit(e)} className="flex flex-col gap-4">
        <FormErrorSummary errors={form.errorList} formError={form.formError} labels={FIELD_LABELS} root={() => formRef.current} />
        <Field label="Nama tahapan" required name="name" error={errors.name}>
          <Input value={values.name} onChange={(e) => set("name", e.target.value)} onBlur={() => blur("name")} placeholder="cth. Fitting Baju Pengantin" />
        </Field>
        <Field label="Target tanggal" required name="targetDate" error={errors.targetDate}>
          <Input type="date" value={values.targetDate} onChange={(e) => set("targetDate", e.target.value)} onBlur={() => blur("targetDate")} />
        </Field>
      </form>
    </Modal>
  );
}
