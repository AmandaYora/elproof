import { useRef } from "react";
import { MILESTONE_STATUS_LABELS } from "@/shared/lib/labels";
import { Modal } from "@/shared/components/ui/Modal";
import { Button } from "@/shared/components/ui/Button";
import { Input, Select, Field } from "@/shared/components/ui/Input";
import type { MilestoneStatus, ProjectMilestone } from "@/modules/projects/types";
import { todayISO } from "@/modules/projects/lib/dates";
import { projectMilestoneEditSchema, MILESTONE_STATUS_OPTIONS } from "@/modules/projects/schemas/project-milestone.schema";
import { useZodForm } from "@/shared/forms/useZodForm";
import { FormErrorSummary } from "@/shared/forms/FormErrorSummary";
import { formatDayCountdown } from "@/shared/lib/formatters";

const STATUS_OPTIONS: MilestoneStatus[] = [...MILESTONE_STATUS_OPTIONS];
const FIELD_LABELS: Record<string, string> = { status: "Status", targetDate: "Target tanggal", completedDate: "Tanggal selesai", forClient: "Tugas untuk pasangan" };

export interface ProjectMilestoneEditFields {
  status: MilestoneStatus;
  targetDate: string;
  completedDate: string;
  forClient: boolean;
}

interface ProjectMilestoneEditModalProps {
  open: boolean;
  onClose: () => void;
  milestone: ProjectMilestone;
  /** Saves, then closes the modal. Throw to keep it open: a 422 is shown on its fields, anything else inside the modal. */
  onSave: (fields: ProjectMilestoneEditFields) => Promise<void>;
}

/** Says that the target follows the event date (EP-06), and that setting it by hand stops that. */
function followHint(daysBeforeEvent: number | null | undefined): string | undefined {
  if (daysBeforeEvent === null || daysBeforeEvent === undefined) return undefined;
  const offset = formatDayCountdown(daysBeforeEvent);
  return `Mengikuti tanggal acara (${offset}). Bila diubah di sini, tanggal ini tidak lagi ikut bergeser saat tanggal acara berubah.`;
}

export function ProjectMilestoneEditModal({ open, onClose, milestone, onSave }: ProjectMilestoneEditModalProps) {
  const form = useZodForm({
    schema: projectMilestoneEditSchema,
    initial: { status: milestone.status, targetDate: milestone.targetDate, completedDate: milestone.completedDate ?? "", forClient: milestone.forClient },
  });
  const { values, errors, set, blur } = form;
  const formRef = useRef<HTMLFormElement>(null);

  function setStatus(status: MilestoneStatus) {
    set("status", status);
    if (status === "Completed" && !values.completedDate) set("completedDate", todayISO());
    // Not done any more: it has no completed date (the server clears it the same way).
    if (status !== "Completed" && values.completedDate) set("completedDate", "");
  }

  const submit = form.handleSubmit(onSave, { root: () => formRef.current, fallback: "Timeline belum tersimpan. Periksa koneksi lalu coba lagi." });

  return (
    <Modal
      open={open}
      onClose={onClose}
      title={milestone.name}
      description="Perbarui status dan jadwal tahapan ini."
      size="sm"
      busy={form.submitting}
      footer={
        <>
          <Button variant="secondary" onClick={onClose} disabled={form.submitting}>Batal</Button>
          <Button type="submit" form="project-milestone-edit-form" disabled={form.submitting}>
            {form.submitting ? "Menyimpan..." : "Simpan perubahan"}
          </Button>
        </>
      }
    >
      <form id="project-milestone-edit-form" ref={formRef} noValidate onSubmit={(e) => void submit(e)} className="flex flex-col gap-4">
        <FormErrorSummary errors={form.errorList} formError={form.formError} labels={FIELD_LABELS} root={() => formRef.current} />
        <Field label="Status" name="status" error={errors.status}>
          <Select value={values.status} onChange={(e) => setStatus(e.target.value as MilestoneStatus)}>
            {STATUS_OPTIONS.map((s) => (
              <option key={s} value={s}>{MILESTONE_STATUS_LABELS.staff[s]}</option>
            ))}
          </Select>
        </Field>
        <Field
          label="Target Tanggal"
          name="targetDate"
          error={errors.targetDate}
          hint={followHint(milestone.daysBeforeEvent)}
        >
          <Input type="date" value={values.targetDate} onChange={(e) => set("targetDate", e.target.value)} onBlur={() => blur("targetDate")} />
        </Field>
        <Field label="Tanggal Selesai" name="completedDate" error={errors.completedDate}>
          <Input type="date" value={values.completedDate} onChange={(e) => set("completedDate", e.target.value)} onBlur={() => blur("completedDate")} />
        </Field>
        <label className="flex items-start gap-2.5 rounded-md border border-border-light bg-surface-muted/40 px-3 py-2.5 text-[13px] text-text-primary">
          <input
            type="checkbox"
            name="forClient"
            checked={values.forClient}
            onChange={(e) => set("forClient", e.target.checked)}
            className="mt-0.5 size-4 shrink-0 accent-primary"
          />
          <span>
            <span className="font-medium">Tugas untuk pasangan</span>
            <span className="block text-[12.5px] text-text-secondary">
              Tampil di Portal Klien sebagai hal yang perlu mereka lakukan, dan pasangan bisa menandainya selesai sendiri.
            </span>
          </span>
        </label>
        {milestone.completedByClient && milestone.status === "Completed" && (
          <p className="text-[12.5px] text-text-secondary">Ditandai selesai oleh pasangan dari Portal Klien.</p>
        )}
      </form>
    </Modal>
  );
}
