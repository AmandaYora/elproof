import { useState } from "react";
import { Modal } from "@/shared/components/ui/Modal";
import { Button } from "@/shared/components/ui/Button";
import { Input, Select, Field } from "@/shared/components/ui/Input";
import type { MilestoneStatus, ProjectMilestone } from "@/modules/projects/types";
import { todayISO } from "@/modules/projects/lib/dates";

const STATUS_OPTIONS: MilestoneStatus[] = ["Not Started", "In Progress", "Completed", "Blocked", "Cancelled"];

export interface ProjectMilestoneEditFields {
  status: MilestoneStatus;
  targetDate: string;
  completedDate: string;
}

interface ProjectMilestoneEditModalProps {
  open: boolean;
  onClose: () => void;
  milestone: ProjectMilestone;
  onSave: (fields: ProjectMilestoneEditFields) => void;
}

export function ProjectMilestoneEditModal({ open, onClose, milestone, onSave }: ProjectMilestoneEditModalProps) {
  const [fields, setFields] = useState<ProjectMilestoneEditFields>({
    status: milestone.status,
    targetDate: milestone.targetDate,
    completedDate: milestone.completedDate ?? "",
  });

  function set<K extends keyof ProjectMilestoneEditFields>(key: K, value: ProjectMilestoneEditFields[K]) {
    setFields((prev) => {
      const next = { ...prev, [key]: value };
      if (key === "status" && value === "Completed" && !prev.completedDate) {
        next.completedDate = todayISO();
      }
      return next;
    });
  }

  return (
    <Modal
      open={open}
      onClose={onClose}
      title={milestone.name}
      description="Perbarui status dan jadwal timeline ini."
      size="sm"
      footer={
        <>
          <Button variant="secondary" onClick={onClose}>Batal</Button>
          <Button onClick={() => onSave(fields)}>Simpan Perubahan</Button>
        </>
      }
    >
      <div className="flex flex-col gap-4">
        <Field label="Status">
          <Select value={fields.status} onChange={(e) => set("status", e.target.value as MilestoneStatus)}>
            {STATUS_OPTIONS.map((s) => (
              <option key={s} value={s}>{s}</option>
            ))}
          </Select>
        </Field>
        <Field label="Target Tanggal">
          <Input type="date" value={fields.targetDate} onChange={(e) => set("targetDate", e.target.value)} />
        </Field>
        <Field label="Tanggal Selesai">
          <Input type="date" value={fields.completedDate} onChange={(e) => set("completedDate", e.target.value)} />
        </Field>
      </div>
    </Modal>
  );
}
