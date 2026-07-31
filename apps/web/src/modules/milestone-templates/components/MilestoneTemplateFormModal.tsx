import { useEffect, useState } from "react";
import { Modal } from "@/shared/components/ui/Modal";
import { Button } from "@/shared/components/ui/Button";
import { Input, Field } from "@/shared/components/ui/Input";
import {
  milestoneTemplateSchema,
  type MilestoneTemplateFormValues,
} from "@/modules/milestone-templates/schemas/milestone-template.schema";
import type { MilestoneTemplate } from "@/modules/milestone-templates/types";

interface MilestoneTemplateFormModalProps {
  open: boolean;
  onClose: () => void;
  onSubmit: (values: MilestoneTemplateFormValues) => void;
  initialTemplate?: MilestoneTemplate;
}

function toFormValues(template?: MilestoneTemplate): MilestoneTemplateFormValues {
  if (!template) {
    return { name: "", daysBeforeEvent: 0 };
  }
  return { name: template.name, daysBeforeEvent: template.daysBeforeEvent };
}

export function MilestoneTemplateFormModal({ open, onClose, onSubmit, initialTemplate }: MilestoneTemplateFormModalProps) {
  const [values, setValues] = useState<MilestoneTemplateFormValues>(() => toFormValues(initialTemplate));
  const [errors, setErrors] = useState<Partial<Record<keyof MilestoneTemplateFormValues, string>>>({});

  // Reset on every fresh open (not after submit) -- this component doesn't
  // remount between two "Tambah Timeline" opens (same key upstream), so
  // resetting only on a successful submit would also wipe the fields on a
  // failed one, forcing the user to retype everything while an error banner
  // is showing. Resetting here instead means a failed submit's in-progress
  // values simply stay put (open never re-transitions to true) while a
  // fresh open — for either Add or Edit — always starts from the right values.
  useEffect(() => {
    if (open) {
      setValues(toFormValues(initialTemplate));
      setErrors({});
    }
  }, [open, initialTemplate]);

  function set<K extends keyof MilestoneTemplateFormValues>(key: K, value: MilestoneTemplateFormValues[K]) {
    setValues((prev) => ({ ...prev, [key]: value }));
  }

  function handleSubmit() {
    const result = milestoneTemplateSchema.safeParse(values);
    if (!result.success) {
      const fieldErrors: Partial<Record<keyof MilestoneTemplateFormValues, string>> = {};
      for (const issue of result.error.issues) {
        const key = issue.path[0] as keyof MilestoneTemplateFormValues;
        fieldErrors[key] = issue.message;
      }
      setErrors(fieldErrors);
      return;
    }
    onSubmit(result.data);
  }

  return (
    <Modal
      open={open}
      onClose={onClose}
      title={initialTemplate ? "Ubah Timeline Default" : "Tambah Timeline Default"}
      description="Item ini akan otomatis ditambahkan ke tab Timeline setiap project baru yang dibuat."
      footer={
        <>
          <Button variant="secondary" onClick={onClose}>
            Batal
          </Button>
          <Button onClick={handleSubmit}>{initialTemplate ? "Simpan Perubahan" : "Simpan Timeline"}</Button>
        </>
      }
    >
      <div className="grid grid-cols-1 gap-4">
        <Field label="Nama Timeline" required hint={errors.name}>
          <Input value={values.name} onChange={(e) => set("name", e.target.value)} placeholder="cth. Survei Venue & Vendor" />
        </Field>
        <Field label="Hari Sebelum Acara (H-)" required hint={errors.daysBeforeEvent}>
          <Input
            type="number"
            min={0}
            value={values.daysBeforeEvent}
            onChange={(e) => set("daysBeforeEvent", Number(e.target.value))}
          />
        </Field>
      </div>
    </Modal>
  );
}
