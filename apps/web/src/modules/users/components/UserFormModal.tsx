import { useState } from "react";
import { Eye, EyeOff } from "lucide-react";
import { Modal } from "@/shared/components/ui/Modal";
import { Button } from "@/shared/components/ui/Button";
import { Input, Select, Field } from "@/shared/components/ui/Input";
import {
  userSchema,
  userCreateSchema,
  STAFF_ROLE_OPTIONS,
  type UserFormValues,
  type UserCreateFormValues,
} from "@/modules/users/schemas/user.schema";
import type { StaffMember } from "@/modules/users/types";

interface FormState extends UserFormValues {
  username: string;
  password: string;
  confirmPassword: string;
}

function toFormState(user?: StaffMember): FormState {
  if (!user) {
    return { name: "", title: "", role: "Staff", email: "", phone: "", username: "", password: "", confirmPassword: "" };
  }
  return {
    name: user.name,
    title: user.title,
    role: user.role,
    email: user.email,
    phone: user.phone,
    username: user.username,
    password: "",
    confirmPassword: "",
  };
}

interface UserFormModalProps {
  open: boolean;
  onClose: () => void;
  initialUser?: StaffMember;
  onSubmitCreate: (values: UserCreateFormValues) => void;
  onSubmitEdit: (values: UserFormValues) => void;
}

export function UserFormModal({ open, onClose, initialUser, onSubmitCreate, onSubmitEdit }: UserFormModalProps) {
  const isEditing = Boolean(initialUser);
  const [values, setValues] = useState<FormState>(() => toFormState(initialUser));
  const [errors, setErrors] = useState<Partial<Record<keyof FormState, string>>>({});
  const [showPassword, setShowPassword] = useState(false);

  function set<K extends keyof FormState>(key: K, value: FormState[K]) {
    setValues((prev) => ({ ...prev, [key]: value }));
  }

  function handleSubmit() {
    if (isEditing) {
      const result = userSchema.safeParse(values);
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

    const result = userCreateSchema.safeParse(values);
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
    setValues(toFormState(initialUser));
    setErrors({});
    onClose();
  }

  return (
    <Modal
      open={open}
      onClose={handleClose}
      title={isEditing ? "Ubah Pengguna" : "Tambah Pengguna Baru"}
      description="Kelola akun staff internal WO yang dapat mengakses WO Console."
      footer={
        <>
          <Button variant="secondary" onClick={handleClose}>Batal</Button>
          <Button onClick={handleSubmit}>{isEditing ? "Simpan Perubahan" : "Simpan Pengguna"}</Button>
        </>
      }
    >
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <Field label="Nama Lengkap" required hint={errors.name}>
          <Input value={values.name} onChange={(e) => set("name", e.target.value)} placeholder="cth. Anisa Putri" />
        </Field>
        <Field label="Role" required>
          {initialUser?.role === "Owner" ? (
            <Input value="Owner" disabled />
          ) : (
            <Select value={values.role} onChange={(e) => set("role", e.target.value as UserFormValues["role"])}>
              {STAFF_ROLE_OPTIONS.map((r) => (
                <option key={r} value={r}>{r}</option>
              ))}
            </Select>
          )}
        </Field>
        <Field label="Jabatan" required hint={errors.title}>
          <Input value={values.title} onChange={(e) => set("title", e.target.value)} placeholder="cth. Lead Planner" />
        </Field>
        <Field label="Nomor Telepon" required hint={errors.phone}>
          <Input value={values.phone} onChange={(e) => set("phone", e.target.value)} placeholder="0812-xxxx-xxxx" />
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
              placeholder="cth. anisa.putri"
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
