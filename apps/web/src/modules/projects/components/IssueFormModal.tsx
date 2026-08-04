import { useState } from "react";
import { Modal } from "@/shared/components/ui/Modal";
import { Button } from "@/shared/components/ui/Button";
import { Input, Textarea, Select, Field } from "@/shared/components/ui/Input";
import { issueSchema, ISSUE_IMPACT_OPTIONS, ISSUE_STATUS_OPTIONS, type IssueFormValues } from "@/modules/projects/schemas/issue.schema";
import type { IssueImpact, IssueStatus, ProjectVendor, VendorMilestone } from "@/modules/projects/types";

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
  onSubmit: (values: IssueFormValues) => void;
  initialValues: IssueFormValues;
  // Edit mode allows reassigning the vendor entirely; create mode always
  // opens from a specific vendor's context, so the vendor field is locked.
  mode: "create" | "edit";
  // Locked (create-from-milestone) means the milestone field is shown as
  // read-only text instead of a picker -- zero extra picking for the common
  // case (PLAN.md "Retire the standalone Kendala tab").
  milestoneLocked: boolean;
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
  vendorEngagements,
  vendors,
  vendorMilestones,
  staff,
}: IssueFormModalProps) {
  const [values, setValues] = useState<IssueFormValues>(initialValues);
  const [errors, setErrors] = useState<Partial<Record<keyof IssueFormValues, string>>>({});

  function set<K extends keyof IssueFormValues>(key: K, value: IssueFormValues[K]) {
    setValues((prev) => ({ ...prev, [key]: value }));
  }

  function handleVendorChange(projectVendorId: string) {
    // Changing the vendor invalidates whatever milestone was picked --
    // milestones belong to exactly one vendor engagement.
    setValues((prev) => ({ ...prev, projectVendorId, vendorMilestoneId: "" }));
  }

  function handleSubmit() {
    const result = issueSchema.safeParse(values);
    if (!result.success) {
      const fieldErrors: Partial<Record<keyof IssueFormValues, string>> = {};
      for (const issue of result.error.issues) {
        fieldErrors[issue.path[0] as keyof IssueFormValues] = issue.message;
      }
      setErrors(fieldErrors);
      return;
    }
    onSubmit(result.data);
  }

  const selectedVendor = vendorEngagements.find((pv) => pv.id === values.projectVendorId);
  const milestoneOptions = vendorMilestones.filter((m) => m.projectVendorId === values.projectVendorId);
  const lockedMilestoneName = milestoneOptions.find((m) => m.id === values.vendorMilestoneId)?.name ?? "Umum";

  return (
    <Modal
      open={open}
      onClose={onClose}
      title={mode === "create" ? "Tambah Kendala" : "Ubah Kendala"}
      description="Catat kendala terkait vendor pada project ini."
      size="lg"
      footer={
        <>
          <Button variant="secondary" onClick={onClose}>Batal</Button>
          <Button onClick={handleSubmit}>{mode === "create" ? "Simpan Kendala" : "Simpan Perubahan"}</Button>
        </>
      }
    >
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <div className="sm:col-span-2">
          <Field label="Judul Kendala" required hint={errors.title}>
            <Input value={values.title} onChange={(e) => set("title", e.target.value)} />
          </Field>
        </div>
        <div className="sm:col-span-2">
          <Field label="Deskripsi" required hint={errors.description}>
            <Textarea rows={2} value={values.description} onChange={(e) => set("description", e.target.value)} />
          </Field>
        </div>

        {mode === "edit" ? (
          <Field label="Vendor" required hint={errors.projectVendorId}>
            <Select value={values.projectVendorId} onChange={(e) => handleVendorChange(e.target.value)}>
              {vendorEngagements.map((pv) => (
                <option key={pv.id} value={pv.id}>{vendorLabel(pv, vendors)}</option>
              ))}
            </Select>
          </Field>
        ) : (
          <Field label="Vendor">
            <p className="flex h-9 items-center text-[13px] font-medium text-text-primary">
              {selectedVendor ? vendorLabel(selectedVendor, vendors) : "Vendor tidak diketahui"}
            </p>
          </Field>
        )}

        {milestoneLocked ? (
          <Field label="Timeline">
            <p className="flex h-9 items-center text-[13px] font-medium text-text-primary">{lockedMilestoneName}</p>
          </Field>
        ) : (
          <Field label="Timeline" hint={errors.vendorMilestoneId}>
            <Select value={values.vendorMilestoneId} onChange={(e) => set("vendorMilestoneId", e.target.value)}>
              <option value="">Umum (tidak terikat ke timeline)</option>
              {milestoneOptions.map((m) => (
                <option key={m.id} value={m.id}>{milestoneLabel(m)}</option>
              ))}
            </Select>
          </Field>
        )}

        <Field label="Dampak" required>
          <Select value={values.impact} onChange={(e) => set("impact", e.target.value as IssueImpact)}>
            {ISSUE_IMPACT_OPTIONS.map((i) => (
              <option key={i} value={i}>{i}</option>
            ))}
          </Select>
        </Field>
        {mode === "edit" && (
          <Field label="Status" required>
            <Select value={values.status} onChange={(e) => set("status", e.target.value as IssueStatus)}>
              {ISSUE_STATUS_OPTIONS.map((s) => (
                <option key={s} value={s}>{s}</option>
              ))}
            </Select>
          </Field>
        )}
        <Field label="Tanggal Ditemukan" required hint={errors.foundDate}>
          <Input type="date" value={values.foundDate} onChange={(e) => set("foundDate", e.target.value)} />
        </Field>
        <Field label="Target Penyelesaian">
          <Input
            type="date"
            value={values.targetResolutionDate}
            onChange={(e) => set("targetResolutionDate", e.target.value)}
          />
        </Field>
        <Field label="PIC" required hint={errors.picStaffId}>
          <Select value={values.picStaffId} onChange={(e) => set("picStaffId", e.target.value)}>
            {staff.map((s) => (
              <option key={s.id} value={s.id}>{s.name} — {s.title}</option>
            ))}
          </Select>
        </Field>
        <div className="sm:col-span-2">
          <Field label="Rencana Penanganan">
            <Textarea rows={2} value={values.resolutionPlan} onChange={(e) => set("resolutionPlan", e.target.value)} />
          </Field>
        </div>
      </div>
    </Modal>
  );
}
