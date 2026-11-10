import { useRef } from "react";
import { ISSUE_STATUS_LABELS } from "@/shared/lib/labels";
import { Modal } from "@/shared/components/ui/Modal";
import { Button } from "@/shared/components/ui/Button";
import { Input, Textarea, Select, Field } from "@/shared/components/ui/Input";
import { issueSchema, ISSUE_IMPACT_OPTIONS, ISSUE_STATUS_OPTIONS, type IssueFormValues } from "@/modules/projects/schemas/issue.schema";
import { useZodForm } from "@/shared/forms/useZodForm";
import { FormErrorSummary } from "@/shared/forms/FormErrorSummary";
import type { IssueImpact, IssueStatus, ProjectVendor, VendorMilestone } from "@/modules/projects/types";

const FIELD_LABELS: Record<string, string> = {
  title: "Judul kendala", description: "Deskripsi", projectVendorId: "Vendor", vendorMilestoneId: "Timeline", impact: "Dampak", status: "Status",
  foundDate: "Tanggal ditemukan", targetResolutionDate: "Target penyelesaian", picStaffId: "PIC", resolutionPlan: "Rencana penanganan",
};

interface VendorLookup {
  id: string;
  name: string;
  city: string | null;
}

interface StaffLookup {
  id: string;
  name: string;
  title: string;
}

interface IssueFormModalProps {
  open: boolean;
  onClose: () => void;
  /** Saves, then closes the modal. Throw to keep it open: a 422 is shown on its fields, anything else inside the modal. */
  onSubmit: (values: IssueFormValues) => Promise<void>;
  initialValues: IssueFormValues;
  // Edit mode allows reassigning the vendor entirely; create mode always
  // opens from a specific vendor's context, so the vendor field is locked.
  mode: "create" | "edit";
  // Locked (create-from-milestone) means the milestone field is shown as
  // read-only text instead of a picker -- zero extra picking for the common
  // case (PLAN.md "Retire the standalone Kendala tab").
  milestoneLocked: boolean;
  /** Opened from the Kendala tab: the vendor is a choice, including "Tidak terkait vendor" (an issue about the event itself). */
  vendorChoice?: boolean;
  vendorEngagements: ProjectVendor[];
  vendors: VendorLookup[];
  vendorMilestones: VendorMilestone[];
  staff: StaffLookup[];
}

function vendorLabel(pv: ProjectVendor, vendors: VendorLookup[]): string {
  const vendor = vendors.find((v) => v.id === pv.vendorId);
  if (!vendor) return "Vendor tidak diketahui";
  return vendor.city ? `${vendor.name} — ${vendor.city}` : vendor.name;
}

function milestoneLabel(m: VendorMilestone): string {
  return m.status === "Cancelled" ? `${m.order}. ${m.name} (Dibatalkan)` : `${m.order}. ${m.name}`;
}

export function IssueFormModal({
  open,
  onClose,
  onSubmit,
  initialValues,
  mode,
  milestoneLocked,
  vendorChoice = false,
  vendorEngagements,
  vendors,
  vendorMilestones,
  staff,
}: IssueFormModalProps) {
  const form = useZodForm({ schema: issueSchema, initial: initialValues });
  const { values, errors, set, blur } = form;
  const formRef = useRef<HTMLFormElement>(null);

  function handleVendorChange(projectVendorId: string) {
    // Changing the vendor invalidates whatever milestone was picked --
    // milestones belong to exactly one vendor engagement.
    set("projectVendorId", projectVendorId);
    set("vendorMilestoneId", "");
  }

  const submit = form.handleSubmit(onSubmit, { root: () => formRef.current, fallback: "Kendala belum tersimpan. Periksa koneksi Anda, lalu coba lagi." });

  const selectedVendor = vendorEngagements.find((pv) => pv.id === values.projectVendorId);
  const milestoneOptions = vendorMilestones.filter((m) => m.projectVendorId === values.projectVendorId);
  const lockedMilestoneName = milestoneOptions.find((m) => m.id === values.vendorMilestoneId)?.name ?? "Umum";

  return (
    <Modal
      open={open}
      onClose={onClose}
      title={mode === "create" ? "Tambah kendala" : "Ubah kendala"}
      description="Catat kendala terkait vendor pada proyek ini."
      size="lg"
      busy={form.submitting}
      footer={
        <>
          <Button variant="secondary" onClick={onClose} disabled={form.submitting}>Batal</Button>
          <Button type="submit" form="issue-form" disabled={form.submitting}>
            {form.submitting ? "Menyimpan..." : mode === "create" ? "Simpan kendala" : "Simpan perubahan"}
          </Button>
        </>
      }
    >
      <form id="issue-form" ref={formRef} noValidate onSubmit={(e) => void submit(e)} className="flex flex-col gap-4">
      <FormErrorSummary errors={form.errorList} formError={form.formError} labels={FIELD_LABELS} root={() => formRef.current} />
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <div className="sm:col-span-2">
          <Field label="Judul kendala" required name="title" error={errors.title}>
            <Input value={values.title} onChange={(e) => set("title", e.target.value)} onBlur={() => blur("title")} />
          </Field>
        </div>
        <div className="sm:col-span-2">
          <Field label="Deskripsi" required name="description" error={errors.description}>
            <Textarea rows={2} value={values.description} onChange={(e) => set("description", e.target.value)} onBlur={() => blur("description")} />
          </Field>
        </div>

        {mode === "edit" || vendorChoice ? (
          <Field label="Vendor" name="projectVendorId" error={errors.projectVendorId}>
            <Select value={values.projectVendorId} onChange={(e) => handleVendorChange(e.target.value)}>
              <option value="">Tidak terkait vendor</option>
              {vendorEngagements.map((pv) => (
                <option key={pv.id} value={pv.id}>{vendorLabel(pv, vendors)}</option>
              ))}
            </Select>
          </Field>
        ) : (
          <Field label="Vendor">
            <p className="flex h-9 items-center text-[13px] font-medium text-text-primary">
              {selectedVendor ? vendorLabel(selectedVendor, vendors) : "Tidak terkait vendor"}
            </p>
          </Field>
        )}

        {!values.projectVendorId ? null : milestoneLocked ? (
          <Field label="Tahapan">
            <p className="flex h-9 items-center text-[13px] font-medium text-text-primary">{lockedMilestoneName}</p>
          </Field>
        ) : (
          <Field label="Tahapan" name="vendorMilestoneId" error={errors.vendorMilestoneId}>
            <Select value={values.vendorMilestoneId} onChange={(e) => set("vendorMilestoneId", e.target.value)}>
              <option value="">Umum (tidak terikat ke tahapan)</option>
              {milestoneOptions.map((m) => (
                <option key={m.id} value={m.id}>{milestoneLabel(m)}</option>
              ))}
            </Select>
          </Field>
        )}

        <Field label="Dampak" required name="impact" error={errors.impact}>
          <Select value={values.impact} onChange={(e) => set("impact", e.target.value as IssueImpact)}>
            {ISSUE_IMPACT_OPTIONS.map((i) => (
              <option key={i} value={i}>{i}</option>
            ))}
          </Select>
        </Field>
        {mode === "edit" && (
          <Field label="Status" required name="status" error={errors.status}>
            <Select value={values.status} onChange={(e) => set("status", e.target.value as IssueStatus)}>
              {ISSUE_STATUS_OPTIONS.map((s) => (
                <option key={s} value={s}>{ISSUE_STATUS_LABELS.staff[s]}</option>
              ))}
            </Select>
          </Field>
        )}
        <Field label="Tanggal ditemukan" required name="foundDate" error={errors.foundDate}>
          <Input type="date" value={values.foundDate} onChange={(e) => set("foundDate", e.target.value)} onBlur={() => blur("foundDate")} />
        </Field>
        <Field label="Target penyelesaian" name="targetResolutionDate" error={errors.targetResolutionDate}>
          <Input
            type="date"
            value={values.targetResolutionDate}
            onChange={(e) => set("targetResolutionDate", e.target.value)}
            onBlur={() => blur("targetResolutionDate")}
          />
        </Field>
        <Field label="PIC" required name="picStaffId" error={errors.picStaffId}>
          <Select value={values.picStaffId} onChange={(e) => set("picStaffId", e.target.value)}>
            {staff.map((s) => (
              <option key={s.id} value={s.id}>{s.name} — {s.title}</option>
            ))}
          </Select>
        </Field>
        <div className="sm:col-span-2">
          <label className="flex items-start gap-2.5 text-[13px] text-text-primary">
            <input
              type="checkbox"
              checked={values.isClientVisible}
              onChange={(e) => set("isClientVisible", e.target.checked)}
              className="mt-0.5 size-4 shrink-0"
            />
            <span>
              <span className="font-medium">Tampilkan ke klien</span>
              <span className="block text-[12.5px] text-text-secondary">Klien melihat kendala ini dan penanganannya di portal. Matikan untuk catatan internal.</span>
            </span>
          </label>
        </div>
        <div className="sm:col-span-2">
          <Field label="Rencana penanganan" name="resolutionPlan" error={errors.resolutionPlan}>
            <Textarea rows={2} value={values.resolutionPlan} onChange={(e) => set("resolutionPlan", e.target.value)} onBlur={() => blur("resolutionPlan")} />
          </Field>
        </div>
      </div>
      </form>
    </Modal>
  );
}
