import { useEffect, useRef } from "react";
import { Modal } from "@/shared/components/ui/Modal";
import { Button } from "@/shared/components/ui/Button";
import { Input, Select, Field } from "@/shared/components/ui/Input";
import {
  milestoneTemplateSchema,
  toDaysBeforeEvent,
  fromDaysBeforeEvent,
  type MilestoneTemplateFormValues,
  type MilestoneTemplateSubmitValues,
} from "@/modules/milestone-templates/schemas/milestone-template.schema";
import type { MilestoneTemplate } from "@/modules/milestone-templates/types";
import { useZodForm } from "@/shared/forms/useZodForm";
import { FormErrorSummary } from "@/shared/forms/FormErrorSummary";

const FIELD_LABELS: Record<string, string> = { name: "Nama tahapan", daysOffset: "Jumlah hari", daysDirection: "Posisi" };

interface MilestoneTemplateFormModalProps {
  open: boolean;
  onClose: () => void;
  /** Saves, then closes the modal. Throw to keep it open: a 422 is shown on its fields, anything else inside the modal. */
  onSubmit: (values: MilestoneTemplateSubmitValues) => Promise<void>;
  initialTemplate?: MilestoneTemplate;
}

function toFormValues(template?: MilestoneTemplate): MilestoneTemplateFormValues {
  if (!template) {
    return { name: "", daysOffset: 0, daysDirection: "before" };
  }
  return { name: template.name, ...fromDaysBeforeEvent(template.daysBeforeEvent) };
}

export function MilestoneTemplateFormModal({ open, onClose, onSubmit, initialTemplate }: MilestoneTemplateFormModalProps) {
  const form = useZodForm({ schema: milestoneTemplateSchema, initial: toFormValues(initialTemplate) });
  const { values, errors, set, blur } = form;
  const formRef = useRef<HTMLFormElement>(null);

  // Reset on every fresh open (not after submit) -- this component doesn't
  // remount between two "Tambah Timeline" opens (same key upstream), so
  // resetting only on a successful submit would also wipe the fields on a
  // failed one, forcing the user to retype everything while an error banner
  // is showing. Resetting here instead means a failed submit's in-progress
  // values simply stay put (open never re-transitions to true) while a
  // fresh open — for either Add or Edit — always starts from the right values.
  useEffect(() => {
    if (open) {
      form.reset(toFormValues(initialTemplate));
    }
  }, [open, initialTemplate]);

  const submit = form.handleSubmit(
    (parsed) => onSubmit({ name: parsed.name, daysBeforeEvent: toDaysBeforeEvent(parsed.daysOffset, parsed.daysDirection) }),
    { root: () => formRef.current, fallback: "Tahapan belum tersimpan. Periksa koneksi Anda, lalu coba lagi." },
  );

  return (
    <Modal
      open={open}
      onClose={onClose}
      title={initialTemplate ? "Ubah tahapan jadwal bawaan" : "Tambah tahapan jadwal bawaan"}
      description="Tahapan ini otomatis ditambahkan ke tab Jadwal setiap proyek baru yang dibuat."
      busy={form.submitting}
      footer={
        <>
          <Button variant="secondary" onClick={onClose} disabled={form.submitting}>
            Batal
          </Button>
          <Button type="submit" form="milestone-template-form" disabled={form.submitting}>
            {form.submitting ? "Menyimpan..." : initialTemplate ? "Simpan perubahan" : "Simpan tahapan"}
          </Button>
        </>
      }
    >
      <form id="milestone-template-form" ref={formRef} noValidate onSubmit={(e) => void submit(e)} className="grid grid-cols-1 gap-4">
        <FormErrorSummary errors={form.errorList} formError={form.formError} labels={FIELD_LABELS} root={() => formRef.current} />
        <Field label="Nama tahapan" required name="name" error={errors.name}>
          <Input value={values.name} onChange={(e) => set("name", e.target.value)} onBlur={() => blur("name")} placeholder="cth. Survei Venue & Vendor" />
        </Field>
        <div className="grid grid-cols-2 gap-4">
          <Field label="Jumlah hari" required name="daysOffset" error={errors.daysOffset}>
            <Input
              type="number"
              min={0}
              value={values.daysOffset}
              onChange={(e) => set("daysOffset", Number(e.target.value))}
              onBlur={() => blur("daysOffset")}
            />
          </Field>
          <Field label="Posisi" required name="daysDirection" error={errors.daysDirection}>
            <Select
              value={values.daysDirection}
              onChange={(e) => set("daysDirection", e.target.value as MilestoneTemplateFormValues["daysDirection"])}
            >
              <option value="before">Sebelum acara</option>
              <option value="after">Setelah acara</option>
            </Select>
          </Field>
        </div>
      </form>
    </Modal>
  );
}
