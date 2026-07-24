import { useState } from "react";
import { Eye, EyeOff } from "lucide-react";
import { Modal } from "@/shared/components/ui/Modal";
import { Button } from "@/shared/components/ui/Button";
import { Input, Select, Field } from "@/shared/components/ui/Input";
import {
  platformAdminSchema,
  platformAdminCreateSchema,
  PLATFORM_ADMIN_ROLE_OPTIONS,
  type PlatformAdminFormValues,
  type PlatformAdminCreateFormValues,
} from "@/modules/platform-admin/schemas/platform-admin.schema";
import type { PlatformAdmin } from "@/modules/platform-admin/data/types";

interface FormState extends PlatformAdminFormValues {
  username: string;
  password: string;
  confirmPassword: string;
}

function toFormState(admin?: PlatformAdmin): FormState {
  if (!admin) {
    return { name: "", title: "", role: "Support", email: "", phone: "", username: "", password: "", confirmPassword: "" };
  }
  return {
    name: admin.name,
    title: admin.title,
    role: admin.role,
    email: admin.email,
    phone: admin.phone,
    username: admin.username,
    password: "",
    confirmPassword: "",
  };
}

interface PlatformAdminFormModalProps {
  open: boolean;
  onClose: () => void;
  initialAdmin?: PlatformAdmin;
  onSubmitCreate: (values: PlatformAdminCreateFormValues) => void;
  onSubmitEdit: (values: PlatformAdminFormValues) => void;
}

export function PlatformAdminFormModal({ open, onClose, initialAdmin, onSubmitCreate, onSubmitEdit }: PlatformAdminFormModalProps) {
  const isEditing = Boolean(initialAdmin);
  const [values, setValues] = useState<FormState>(() => toFormState(initialAdmin));
  const [errors, setErrors] = useState<Partial<Record<keyof FormState, string>>>({});
  const [showPassword, setShowPassword] = useState(false);

  function set<K extends keyof FormState>(key: K, value: FormState[K]) {
    setValues((prev) => ({ ...prev, [key]: value }));
  }

  function handleSubmit() {
    if (isEditing) {
      const result = platformAdminSchema.safeParse(values);
      if (!result.success) {
        const fieldErrors: Partial<Record<keyof FormState, string>> = {};
        for (const issue of result.error.issues) {
          fieldErrors[issue.path[0] as keyof FormState] = issue.message;
        }
        setErrors(fieldErrors);
        return;
      }
      onSubmitEdit(result.data);
      setErrors({});
      return;
    }

    const result = platformAdminCreateSchema.safeParse(values);
    if (!result.success) {
      const fieldErrors: Partial<Record<keyof FormState, string>> = {};
      for (const issue of result.error.issues) {
        fieldErrors[issue.path[0] as keyof FormState] = issue.message;
      }
      setErrors(fieldErrors);
      return;
    }
    onSubmitCreate(result.data);
    setErrors({});
  }

  function handleClose() {
    setValues(toFormState(initialAdmin));
    setErrors({});
    onClose();
  }

  return (
    <Modal
      open={open}
      onClose={handleClose}
      title={isEditing ? "Ubah Pengguna" : "Tambah Pengguna Baru"}
      description="Kelola akun tim internal ElProof yang dapat mengakses Platform Console."
      footer={
        <>
          <Button variant="secondary" onClick={handleClose}>
            Batal
          </Button>
          <Button onClick={handleSubmit}>{isEditing ? "Simpan Perubahan" : "Simpan Pengguna"}</Button>
        </>
      }
    >
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <Field label="Nama Lengkap" required hint={errors.name}>
          <Input value={values.name} onChange={(e) => set("name", e.target.value)} placeholder="cth. Reza Hakim" />
        </Field>
        <Field label="Role" required>
          <Select value={values.role} onChange={(e) => set("role", e.target.value as FormState["role"])}>
            {PLATFORM_ADMIN_ROLE_OPTIONS.map((r) => (
              <option key={r} value={r}>
                {r}
              </option>
            ))}
          </Select>
        </Field>
        <Field label="Jabatan" required hint={errors.title}>
          <Input value={values.title} onChange={(e) => set("title", e.target.value)} placeholder="cth. Super Admin ElProof" />
        </Field>
        <Field label="Nomor Telepon" required hint={errors.phone}>
          <Input value={values.phone} onChange={(e) => set("phone", e.target.value)} placeholder="08xx-xxxx-xxxx" />
        </Field>
        <div className="sm:col-span-2">
          <Field label="Email" required hint={errors.email}>
            <Input type="email" value={values.email} onChange={(e) => set("email", e.target.value)} placeholder="nama@elproof.id" />
          </Field>
        </div>
        <div className="sm:col-span-2">
          <Field label="Username" required={!isEditing} hint={errors.username}>
            <Input
              value={values.username}
              disabled={isEditing}
              onChange={(e) => set("username", e.target.value)}
              placeholder="cth. reza.hakim"
            />
          </Field>
        </div>

        {!isEditing && (
          <>
            <Field label="Password" required hint={errors.password}>
              <div className="relative">
                <Input
                  type={showPassword ? "text" : "password"}
                  value={values.password}
                  onChange={(e) => set("password", e.target.value)}
                  placeholder="Minimal 8 karakter"
                  className="pr-10"
                />
                <button
                  type="button"
                  onClick={() => setShowPassword((v) => !v)}
                  aria-label={showPassword ? "Sembunyikan password" : "Tampilkan password"}
                  className="absolute right-3 top-1/2 -translate-y-1/2 text-text-secondary hover:text-text-primary"
                >
                  {showPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                </button>
              </div>
            </Field>
            <Field label="Konfirmasi Password" required hint={errors.confirmPassword}>
              <Input
                type={showPassword ? "text" : "password"}
                value={values.confirmPassword}
                onChange={(e) => set("confirmPassword", e.target.value)}
                placeholder="Ulangi password"
              />
            </Field>
          </>
        )}
      </div>
    </Modal>
  );
}
