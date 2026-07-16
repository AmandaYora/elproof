import { useEffect, useState } from "react";
import { Modal } from "@/shared/components/ui/Modal";
import { Button } from "@/shared/components/ui/Button";
import { Input, Textarea, Select, Field } from "@/shared/components/ui/Input";
import { projectSchema, PROJECT_STATUS_OPTIONS, type ProjectFormValues } from "@/modules/projects/schemas/project.schema";
import type { Project } from "@/modules/projects/types";
import { useStaffStore } from "@/modules/users/stores/useStaffStore";

interface ProjectFormModalProps {
  open: boolean;
  onClose: () => void;
  onSubmit: (values: ProjectFormValues) => void;
  initialProject?: Project;
}

function toFormValues(project?: Project, defaultStaffId = ""): ProjectFormValues {
  if (!project) {
    return {
      name: "",
      brideName: "",
      groomName: "",
      eventDate: "",
      venue: "",
      prepStartDate: "",
      packageName: "",
      contractValue: 0,
      status: "Draft",
      picStaffId: defaultStaffId,
      description: "",
    };
  }
  return {
    name: project.name,
    brideName: project.brideName,
    groomName: project.groomName,
    eventDate: project.eventDate,
    venue: project.venue,
    prepStartDate: project.prepStartDate,
    packageName: project.packageName,
    contractValue: project.contractValue,
    status: project.status,
    picStaffId: project.picStaffId,
    description: project.description,
  };
}

export function ProjectFormModal({ open, onClose, onSubmit, initialProject }: ProjectFormModalProps) {
  const staffList = useStaffStore((s) => s.staff);
  const fetchStaff = useStaffStore((s) => s.fetchStaff);
  const [values, setValues] = useState<ProjectFormValues>(() => toFormValues(initialProject));
  const [errors, setErrors] = useState<Partial<Record<keyof ProjectFormValues, string>>>({});

  useEffect(() => {
    void fetchStaff();
  }, [fetchStaff]);

  useEffect(() => {
    if (!initialProject && !values.picStaffId && staffList.length > 0) {
      setValues((prev) => ({ ...prev, picStaffId: staffList[0].id }));
    }
  }, [initialProject, staffList, values.picStaffId]);

  function set<K extends keyof ProjectFormValues>(key: K, value: ProjectFormValues[K]) {
    setValues((prev) => ({ ...prev, [key]: value }));
  }

  function handleSubmit() {
    const result = projectSchema.safeParse(values);
    if (!result.success) {
      const fieldErrors: Partial<Record<keyof ProjectFormValues, string>> = {};
      for (const issue of result.error.issues) {
        const key = issue.path[0] as keyof ProjectFormValues;
        fieldErrors[key] = issue.message;
      }
      setErrors(fieldErrors);
      return;
    }
    onSubmit(result.data);
    setValues(toFormValues(undefined, staffList[0]?.id ?? ""));
    setErrors({});
  }

  return (
    <Modal
      open={open}
      onClose={onClose}
      title={initialProject ? "Ubah Project" : "Tambah Project Baru"}
      description="Informasi dasar project pernikahan yang dikelola WO."
      size="lg"
      footer={
        <>
          <Button variant="secondary" onClick={onClose}>Batal</Button>
          <Button onClick={handleSubmit}>{initialProject ? "Simpan Perubahan" : "Simpan Project"}</Button>
        </>
      }
    >
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <Field label="Nama Project" required hint={errors.name}>
          <Input value={values.name} onChange={(e) => set("name", e.target.value)} placeholder="cth. Aurelia & Bagas Wedding" />
        </Field>
        <Field label="Status Project" required>
          <Select value={values.status} onChange={(e) => set("status", e.target.value as ProjectFormValues["status"])}>
            {PROJECT_STATUS_OPTIONS.map((s) => (
              <option key={s} value={s}>{s}</option>
            ))}
          </Select>
        </Field>
        <Field label="Nama Mempelai Wanita" required hint={errors.brideName}>
          <Input value={values.brideName} onChange={(e) => set("brideName", e.target.value)} />
        </Field>
        <Field label="Nama Mempelai Pria" required hint={errors.groomName}>
          <Input value={values.groomName} onChange={(e) => set("groomName", e.target.value)} />
        </Field>
        <Field label="Tanggal Acara" required hint={errors.eventDate}>
          <Input type="date" value={values.eventDate} onChange={(e) => set("eventDate", e.target.value)} />
        </Field>
        <Field label="Tanggal Mulai Persiapan" required hint={errors.prepStartDate}>
          <Input type="date" value={values.prepStartDate} onChange={(e) => set("prepStartDate", e.target.value)} />
        </Field>
        <Field label="Lokasi / Venue" required hint={errors.venue}>
          <Input value={values.venue} onChange={(e) => set("venue", e.target.value)} />
        </Field>
        <Field label="Paket / Layanan" required hint={errors.packageName}>
          <Input value={values.packageName} onChange={(e) => set("packageName", e.target.value)} />
        </Field>
        <Field label="Nilai Kontrak (Rp)" required hint={errors.contractValue}>
          <Input
            type="number"
            min={0}
            value={values.contractValue}
            onChange={(e) => set("contractValue", Number(e.target.value))}
          />
        </Field>
        <Field label="Penanggung Jawab WO" required hint={errors.picStaffId}>
          <Select value={values.picStaffId} onChange={(e) => set("picStaffId", e.target.value)}>
            {staffList.map((s) => (
              <option key={s.id} value={s.id}>{s.name} — {s.title}</option>
            ))}
          </Select>
        </Field>
        <div className="sm:col-span-2">
          <Field label="Deskripsi / Catatan Project">
            <Textarea rows={3} value={values.description} onChange={(e) => set("description", e.target.value)} />
          </Field>
        </div>
      </div>
    </Modal>
  );
}
