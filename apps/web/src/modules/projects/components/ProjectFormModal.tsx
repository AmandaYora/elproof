import { useEffect, useId, useRef, useState } from "react";
import { useCan, useAllProjects } from "@/shared/stores/usePermissionStore";
import { Modal } from "@/shared/components/ui/Modal";
import { Button } from "@/shared/components/ui/Button";
import { Input, Textarea, Select, Field } from "@/shared/components/ui/Input";
import { CurrencyInput } from "@/shared/components/ui/CurrencyInput";
import { projectSchema, type ProjectFormValues } from "@/modules/projects/schemas/project.schema";
import type { Project } from "@/modules/projects/types";
import { useStaffStore } from "@/modules/users/stores/useStaffStore";
import { useAuthStore } from "@/shared/stores/useAuthStore";
import { getApiErrorMessage } from "@/shared/lib/api-error";

interface ProjectFormModalProps {
  open: boolean;
  onClose: () => void;
  /**
   * Saves the form. Throw to keep the modal open: the message is shown inside
   * it and what the user typed stays. The caller closes the modal on success.
   */
  onSubmit: (values: ProjectFormValues) => Promise<void>;
  initialProject?: Project;
  // "duplicate" pre-fills every field from initialProject (like edit) but is
  // meant to create a brand-new project — see ADR-0014. Name gets a "(Salinan)"
  // suffix and status resets to Draft as safer defaults; everything else
  // (including dates) is copied verbatim for the user to adjust as needed.
  mode?: "edit" | "duplicate";
}

function toFormValues(project?: Project, defaultStaffId = "", mode?: "edit" | "duplicate"): ProjectFormValues {
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
      picSalesStaffId: "",
      description: "",
    };
  }
  return {
    name: mode === "duplicate" ? `${project.name} (Salinan)` : project.name,
    brideName: project.brideName,
    groomName: project.groomName,
    eventDate: project.eventDate,
    venue: project.venue,
    prepStartDate: project.prepStartDate,
    packageName: project.packageName,
    contractValue: project.contractValue,
    status: mode === "duplicate" ? "Draft" : project.status,
    picStaffId: project.picStaffId,
    picSalesStaffId: project.picSalesStaffId,
    description: project.description,
  };
}

// The form's state lives in an inner component that only exists while the
// modal is open. Each opening therefore starts from the project as it is now
// (not as it was when the page first rendered) and never carries over what a
// previous opening left behind.
export function ProjectFormModal(props: ProjectFormModalProps) {
  if (!props.open) return null;
  return <ProjectFormModalBody {...props} />;
}

function ProjectFormModalBody({ open, onClose, onSubmit, initialProject, mode }: ProjectFormModalProps) {
  const formId = useId();
  const formRef = useRef<HTMLFormElement>(null);
  const [submitting, setSubmitting] = useState(false);
  const [submitError, setSubmitError] = useState<string | null>(null);
  const staffList = useStaffStore((s) => s.staffSummaries);
  const fetchStaff = useStaffStore((s) => s.fetchStaffSummaries);
  const canAssignPIC = useCan("project.assign_pic");
  const assignedOnly = !useAllProjects();
  const currentStaffId = useAuthStore((s) => s.currentStaffId);
  const [values, setValues] = useState<ProjectFormValues>(() => toFormValues(initialProject, "", mode));
  const [errors, setErrors] = useState<Partial<Record<keyof ProjectFormValues, string>>>({});

  // Wedding Planner never reassigns a project's PIC, even their own project's
  // (only Owner/Admin do — see PLAN.md's RBAC section); the backend already
  // rejects a changed picStaffId from this role, so this is a read-only
  // display for them, not a disabled-but-still-submittable control.
  const picLocked = !canAssignPIC && Boolean(initialProject) && mode !== "duplicate";
  // Sales always creates a project as their own PIC Sales — the backend
  // forces this server-side regardless of what's submitted, so the field is
  // shown read-only (locked to self) for them rather than a disabled-but-
  // submittable control. Owner/Admin keep a free picker, including on an
  // existing project (mirrors picStaffId's own edit affordance).
  // A role limited to its own projects becomes PIC Sales of what it creates
  // (the API enforces it), so the field is shown locked to self.
  const picSalesLocked = assignedOnly && !initialProject;
  // One place for the venue: while one is attached from the directory its name
  // is the location, so this text is not edited here (a duplicate starts free).
  const venueFromDirectory = Boolean(initialProject?.venueId) && mode !== "duplicate";

  useEffect(() => {
    void fetchStaff();
  }, [fetchStaff]);

  useEffect(() => {
    if (!initialProject && !values.picStaffId && staffList.length > 0 && !assignedOnly) {
      setValues((prev) => ({ ...prev, picStaffId: staffList[0].id }));
    }
  }, [initialProject, staffList, values.picStaffId, assignedOnly]);

  useEffect(() => {
    if (!initialProject && picSalesLocked) {
      setValues((prev) => (prev.picSalesStaffId === currentStaffId ? prev : { ...prev, picSalesStaffId: currentStaffId }));
    }
  }, [initialProject, picSalesLocked, currentStaffId]);

  function set<K extends keyof ProjectFormValues>(key: K, value: ProjectFormValues[K]) {
    setValues((prev) => ({ ...prev, [key]: value }));
  }

  async function handleSubmit() {
    if (submitting) return;
    setSubmitError(null);
    const result = projectSchema.safeParse(values);
    if (!result.success) {
      const fieldErrors: Partial<Record<keyof ProjectFormValues, string>> = {};
      for (const issue of result.error.issues) {
        const key = issue.path[0] as keyof ProjectFormValues;
        fieldErrors[key] ??= issue.message;
      }
      setErrors(fieldErrors);
      // Take the user to the first field that needs attention.
      requestAnimationFrame(() => formRef.current?.querySelector<HTMLElement>('[aria-invalid="true"]')?.focus());
      return;
    }
    setErrors({});
    setSubmitting(true);
    try {
      await onSubmit(result.data);
    } catch (err) {
      setSubmitError(getApiErrorMessage(err, "Project belum tersimpan. Periksa koneksi lalu coba lagi."));
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <Modal
      open={open}
      onClose={onClose}
      title={mode === "duplicate" ? "Duplikat Project" : initialProject ? "Ubah Project" : "Tambah Project Baru"}
      description={
        mode === "duplicate"
          ? "Project baru berdasarkan project ini — timeline dan daftar vendor akan ikut disalin sebagai template. Sesuaikan bagian yang berbeda sebelum menyimpan."
          : "Informasi dasar project pernikahan yang dikelola WO."
      }
      size="lg"
      busy={submitting}
      error={submitError}
      footer={
        <>
          <Button variant="secondary" onClick={onClose} disabled={submitting}>Batal</Button>
          <Button type="submit" form={formId} disabled={submitting}>
            {submitting
              ? "Menyimpan..."
              : mode === "duplicate"
                ? "Buat Duplikat"
                : initialProject
                  ? "Simpan Perubahan"
                  : "Simpan Project"}
          </Button>
        </>
      }
    >
      <form
        id={formId}
        ref={formRef}
        noValidate
        onSubmit={(e) => {
          e.preventDefault();
          void handleSubmit();
        }}
        className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <div className="sm:col-span-2">
        <Field label="Nama Project" required error={errors.name}>
          <Input value={values.name} onChange={(e) => set("name", e.target.value)} placeholder="cth. Aurelia & Bagas Wedding" aria-invalid={errors.name ? true : undefined} />
        </Field>
        </div>
        <Field label="Nama Mempelai Wanita" required error={errors.brideName}>
          <Input value={values.brideName} onChange={(e) => set("brideName", e.target.value)} aria-invalid={errors.brideName ? true : undefined} />
        </Field>
        <Field label="Nama Mempelai Pria" required error={errors.groomName}>
          <Input value={values.groomName} onChange={(e) => set("groomName", e.target.value)} aria-invalid={errors.groomName ? true : undefined} />
        </Field>
        <Field label="Tanggal Acara" required error={errors.eventDate}>
          <Input type="date" value={values.eventDate} onChange={(e) => set("eventDate", e.target.value)} aria-invalid={errors.eventDate ? true : undefined} />
        </Field>
        <Field label="Tanggal Booking" required error={errors.prepStartDate}>
          <Input type="date" value={values.prepStartDate} onChange={(e) => set("prepStartDate", e.target.value)} aria-invalid={errors.prepStartDate ? true : undefined} />
        </Field>
        <Field
          label="Lokasi / Venue"
          required
          error={errors.venue}
          hint={venueFromDirectory ? "Venue dipilih dari direktori. Ubah di tab Venue project ini." : undefined}
        >
          <Input
            value={values.venue}
            onChange={(e) => set("venue", e.target.value)}
            aria-invalid={errors.venue ? true : undefined}
            disabled={venueFromDirectory}
          />
        </Field>
        <Field label="Paket / Layanan" required error={errors.packageName}>
          <Input value={values.packageName} onChange={(e) => set("packageName", e.target.value)} aria-invalid={errors.packageName ? true : undefined} />
        </Field>
        <Field label="Nilai Kontrak (Rp)" required error={errors.contractValue}>
          <CurrencyInput value={values.contractValue} onChange={(n) => set("contractValue", n)} aria-invalid={errors.contractValue ? true : undefined} />
        </Field>
        <Field label="Penanggung Jawab WO (PIC Wedding Planner)" error={errors.picStaffId}>
          {picLocked ? (
            <Input value={staffList.find((s) => s.id === values.picStaffId)?.name ?? "..."} disabled />
          ) : (
            <Select value={values.picStaffId} onChange={(e) => set("picStaffId", e.target.value)}>
              <option value="">Belum ditugaskan</option>
              {staffList.map((s) => (
                <option key={s.id} value={s.id}>{s.name} — {s.title}</option>
              ))}
            </Select>
          )}
        </Field>
        <Field label="PIC Sales" error={errors.picSalesStaffId}>
          {picSalesLocked ? (
            <Input value={staffList.find((s) => s.id === values.picSalesStaffId)?.name ?? "..."} disabled />
          ) : (
            <Select value={values.picSalesStaffId} onChange={(e) => set("picSalesStaffId", e.target.value)}>
              <option value="">Belum ditugaskan</option>
              {staffList.map((s) => (
                <option key={s.id} value={s.id}>{s.name} — {s.title}</option>
              ))}
            </Select>
          )}
        </Field>
        <div className="sm:col-span-2">
          <Field label="Deskripsi / Catatan Project">
            <Textarea rows={3} value={values.description} onChange={(e) => set("description", e.target.value)} />
          </Field>
        </div>
      </form>
    </Modal>
  );
}
