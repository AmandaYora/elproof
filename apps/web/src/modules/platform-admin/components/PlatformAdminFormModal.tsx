import { useEffect, useRef, useState } from "react";
import { Eye, EyeOff } from "lucide-react";
import { Modal } from "@/shared/components/ui/Modal";
import { Button } from "@/shared/components/ui/Button";
import { Input, Select, Field } from "@/shared/components/ui/Input";
import {
  platformAdminSchema,
  platformAdminCreateSchema,
  PLATFORM_ADMIN_ROLE_OPTIONS,
  PLATFORM_ADMIN_ROLE_DESCRIPTIONS,
  type PlatformAdminFormValues,
  type PlatformAdminCreateFormValues,
} from "@/modules/platform-admin/schemas/platform-admin.schema";
import type { PlatformAdmin } from "@/modules/platform-admin/data/types";
import { useZodForm } from "@/shared/forms/useZodForm";
import { FormErrorSummary } from "@/shared/forms/FormErrorSummary";

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

const FIELD_LABELS: Record<string, string> = {
  name: "Nama lengkap", role: "Peran", title: "Jabatan", phone: "Nomor telepon", email: "Email", username: "Username", password: "Password", confirmPassword: "Konfirmasi password",
};

interface PlatformAdminFormModalProps {
  open: boolean;
  onClose: () => void;
  initialAdmin?: PlatformAdmin;
  /** Saves, then closes the modal. Throw to keep it open: the message is shown inside it, a 422 on its fields. */
  onSubmitCreate: (values: PlatformAdminCreateFormValues) => Promise<void>;
  onSubmitEdit: (values: PlatformAdminFormValues) => Promise<void>;
}

export function PlatformAdminFormModal({ open, onClose, initialAdmin, onSubmitCreate, onSubmitEdit }: PlatformAdminFormModalProps) {
  const isEditing = Boolean(initialAdmin);
  // Editing checks only the profile fields; the extra keys of the create schema are ignored there.
  const form = useZodForm({
    schema: (isEditing ? platformAdminSchema : platformAdminCreateSchema) as typeof platformAdminCreateSchema,
    initial: toFormState(initialAdmin),
  });
  const { values, errors, set, blur } = form;
  const formRef = useRef<HTMLFormElement>(null);
  const [showPassword, setShowPassword] = useState(false);

  // The modal stays mounted while closed: start clean each time it opens.
  useEffect(() => {
    form.reset(toFormState(initialAdmin));
    setShowPassword(false);
  }, [initialAdmin, open]);

  const submit = form.handleSubmit(
    async (parsed) => {
      if (isEditing) await onSubmitEdit(parsed as PlatformAdminFormValues);
      else await onSubmitCreate(parsed as PlatformAdminCreateFormValues);
    },
    { root: () => formRef.current, fallback: "Pengguna belum tersimpan. Periksa koneksi lalu coba lagi.", known: (key) => key in FIELD_LABELS },
  );

  return (
    <Modal
      open={open}
      onClose={onClose}
      busy={form.submitting}
      title={isEditing ? "Ubah Pengguna" : "Tambah Pengguna Baru"}
      description="Kelola akun tim internal ElProof yang dapat mengakses Platform Console."
      footer={
        <>
          <Button variant="secondary" onClick={onClose} disabled={form.submitting}>
            Batal
          </Button>
          <Button type="submit" form="platform-admin-form" disabled={form.submitting}>
            {form.submitting ? "Menyimpan..." : isEditing ? "Simpan perubahan" : "Simpan pengguna"}
          </Button>
        </>
      }
    >
      <form id="platform-admin-form" ref={formRef} noValidate onSubmit={(e) => void submit(e)} className="flex flex-col gap-4">
      <FormErrorSummary errors={form.errorList} formError={form.formError} labels={FIELD_LABELS} root={() => formRef.current} />
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <Field label="Nama Lengkap" required name="name" error={errors.name}>
          <Input value={values.name} onChange={(e) => set("name", e.target.value)} onBlur={() => blur("name")} placeholder="cth. Reza Hakim" />
        </Field>
        <Field label="Peran" required name="role" error={errors.role} hint={PLATFORM_ADMIN_ROLE_DESCRIPTIONS[values.role]}>
          <Select value={values.role} onChange={(e) => set("role", e.target.value as FormState["role"])}>
            {PLATFORM_ADMIN_ROLE_OPTIONS.map((r) => (
              <option key={r} value={r}>
                {r}
              </option>
            ))}
          </Select>
        </Field>
        <Field label="Jabatan" required name="title" error={errors.title}>
          <Input value={values.title} onChange={(e) => set("title", e.target.value)} onBlur={() => blur("title")} placeholder="cth. Super Admin ElProof" />
        </Field>
        <Field label="Nomor Telepon" required name="phone" error={errors.phone}>
          <Input value={values.phone} onChange={(e) => set("phone", e.target.value)} onBlur={() => blur("phone")} placeholder="08xx-xxxx-xxxx" />
        </Field>
        <div className="sm:col-span-2">
          <Field label="Email" required name="email" error={errors.email}>
            <Input type="email" value={values.email} onChange={(e) => set("email", e.target.value)} onBlur={() => blur("email")} placeholder="nama@elproof.id" />
          </Field>
        </div>
        <div className="sm:col-span-2">
          <Field label="Username" required={!isEditing} name="username" error={errors.username}>
            <Input
              value={values.username}
              disabled={isEditing}
              onChange={(e) => set("username", e.target.value)}
              onBlur={() => blur("username")}
              placeholder="cth. reza.hakim"
            />
          </Field>
        </div>

        {!isEditing && (
          <>
            <Field label="Password" required name="password" error={errors.password}>
              <div className="relative">
                <Input
                  type={showPassword ? "text" : "password"}
                  value={values.password}
                  onChange={(e) => set("password", e.target.value)}
                  onBlur={() => blur("password")}
                  aria-invalid={errors.password ? true : undefined}
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
            <Field label="Konfirmasi Password" required name="confirmPassword" error={errors.confirmPassword}>
              <Input
                type={showPassword ? "text" : "password"}
                value={values.confirmPassword}
                onChange={(e) => set("confirmPassword", e.target.value)}
                onBlur={() => blur("confirmPassword")}
                placeholder="Ulangi password"
              />
            </Field>
          </>
        )}
      </div>
      </form>
    </Modal>
  );
}
