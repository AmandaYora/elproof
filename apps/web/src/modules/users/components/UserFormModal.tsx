import { useState } from "react";
import { Modal } from "@/shared/components/ui/Modal";
import { Button } from "@/shared/components/ui/Button";
import { Input, Select, Field } from "@/shared/components/ui/Input";
import { userSchema, STAFF_ROLE_OPTIONS, type UserFormValues } from "@/modules/users/schemas/user.schema";
import { ROLE_LABELS, type StaffMember } from "@/modules/users/types";
import { useTenantBrandingStore } from "@/shared/stores/useTenantBrandingStore";
import { APP_NAME } from "@/shared/constants/brand";

function toFormState(user?: StaffMember): UserFormValues {
  if (!user) return { name: "", title: "", role: "Staff", email: "", phone: "" };
  return { name: user.name, title: user.title, role: user.role, email: user.email, phone: user.phone };
}

interface UserFormModalProps {
  open: boolean;
  onClose: () => void;
  initialUser?: StaffMember;
  /** Inviting: the parent shows the invitation link afterwards. */
  onSubmitCreate: (values: UserFormValues) => void;
  onSubmitEdit: (values: UserFormValues) => void;
  submitting?: boolean;
  /** Server refusal (e.g. trial limit), shown inside the form. */
  error?: string | null;
}

export function UserFormModal({ open, onClose, initialUser, onSubmitCreate, onSubmitEdit, submitting = false, error = null }: UserFormModalProps) {
  const businessName = useTenantBrandingStore((s) => s.businessName) ?? APP_NAME;
  const isEditing = Boolean(initialUser);
  const pendingInvite = initialUser?.loginStatus === "invited";
  const [values, setValues] = useState<UserFormValues>(() => toFormState(initialUser));
  const [errors, setErrors] = useState<Partial<Record<keyof UserFormValues, string>>>({});

  function set<K extends keyof UserFormValues>(key: K, value: UserFormValues[K]) {
    setValues((prev) => ({ ...prev, [key]: value }));
    setErrors((prev) => ({ ...prev, [key]: undefined }));
  }

  function handleSubmit() {
    const result = userSchema.safeParse(values);
    if (!result.success) {
      const fieldErrors: Partial<Record<keyof UserFormValues, string>> = {};
      for (const issue of result.error.issues) fieldErrors[issue.path[0] as keyof UserFormValues] = issue.message;
      setErrors(fieldErrors);
      return;
    }
    setErrors({});
    if (isEditing) onSubmitEdit(result.data);
    else onSubmitCreate(result.data);
  }

  function handleClose() {
    setValues(toFormState(initialUser));
    setErrors({});
    onClose();
  }

  return (
    <Modal
      open={open}
      onClose={handleClose}
      title={isEditing ? "Ubah Pengguna" : "Undang Pengguna"}
      description={
        isEditing
          ? `Data staf ${businessName}.`
          : `Orang ini menerima link undangan dan masuk ke ${businessName} dengan akun Google-nya.`
      }
      footer={
        <>
          <Button variant="secondary" onClick={handleClose} disabled={submitting}>
            Batal
          </Button>
          <Button onClick={handleSubmit} disabled={submitting}>
            {isEditing ? "Simpan Perubahan" : submitting ? "Membuat undangan..." : "Buat Undangan"}
          </Button>
        </>
      }
    >
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        {error && (
          <p role="alert" className="rounded-md border border-danger/30 bg-danger-soft px-3.5 py-2.5 text-[13px] font-medium text-danger sm:col-span-2">
            {error}
          </p>
        )}
        <Field label="Nama Lengkap" required error={errors.name}>
          <Input value={values.name} onChange={(e) => set("name", e.target.value)} placeholder="cth. Anisa Putri" />
        </Field>
        <Field label="Role" required>
          {initialUser?.role === "Owner" ? (
            <Input value="Owner" disabled />
          ) : (
            <Select value={values.role} onChange={(e) => set("role", e.target.value as UserFormValues["role"])}>
              {STAFF_ROLE_OPTIONS.map((r) => (
                <option key={r} value={r}>
                  {ROLE_LABELS[r]}
                </option>
              ))}
            </Select>
          )}
        </Field>
        <Field label="Jabatan" required error={errors.title}>
          <Input value={values.title} onChange={(e) => set("title", e.target.value)} placeholder="cth. Lead Planner" />
        </Field>
        <Field label="Nomor WhatsApp" error={errors.phone} hint="Untuk mengirim link undangan lewat WhatsApp">
          <Input value={values.phone} onChange={(e) => set("phone", e.target.value)} placeholder="0812 3456 7890" inputMode="tel" />
        </Field>
        <div className="sm:col-span-2">
          <Field
            label="Email akun Google"
            required
            error={errors.email}
            hint={
              isEditing && !pendingInvite
                ? "Akun ini sudah terhubung ke Google. Mengubah email tidak mengganti akun Google yang dipakai masuk."
                : "Undangan hanya bisa diterima dengan akun Google yang memakai email ini."
            }
          >
            <Input type="email" value={values.email} onChange={(e) => set("email", e.target.value)} placeholder="nama@gmail.com" autoComplete="off" />
          </Field>
        </div>
      </div>
    </Modal>
  );
}
