import { useEffect, useRef, useState } from "react";
import { Eye, EyeOff } from "lucide-react";
import { Modal } from "@/shared/components/ui/Modal";
import { Button } from "@/shared/components/ui/Button";
import { Input, Field } from "@/shared/components/ui/Input";
import {
  resetPlatformAdminPasswordSchema,
  type ResetPlatformAdminPasswordFormValues,
} from "@/modules/platform-admin/schemas/platform-admin.schema";
import { useZodForm } from "@/shared/forms/useZodForm";
import { FormErrorSummary } from "@/shared/forms/FormErrorSummary";

const EMPTY_VALUES: ResetPlatformAdminPasswordFormValues = { password: "", confirmPassword: "" };
const FIELD_LABELS: Record<string, string> = { password: "Password baru", confirmPassword: "Konfirmasi password baru" };

interface PlatformAdminResetPasswordModalProps {
  open: boolean;
  onClose: () => void;
  adminName?: string;
  /** Resets, then closes the modal. Throw to keep it open: the message is shown inside it. */
  onSubmit: (values: ResetPlatformAdminPasswordFormValues) => Promise<void>;
}

export function PlatformAdminResetPasswordModal({ open, onClose, adminName, onSubmit }: PlatformAdminResetPasswordModalProps) {
  const form = useZodForm({ schema: resetPlatformAdminPasswordSchema, initial: EMPTY_VALUES });
  const { values, errors, set, blur } = form;
  const formRef = useRef<HTMLFormElement>(null);
  const [showPassword, setShowPassword] = useState(false);

  // The modal stays mounted while closed: start clean each time it opens.
  useEffect(() => {
    form.reset(EMPTY_VALUES);
    setShowPassword(false);
  }, [open]);

  const submit = form.handleSubmit(onSubmit, {
    root: () => formRef.current,
    fallback: "Password belum direset. Periksa koneksi lalu coba lagi.",
    known: (key) => key in FIELD_LABELS,
  });

  return (
    <Modal
      open={open}
      onClose={onClose}
      busy={form.submitting}
      title="Reset Password Pengguna"
      description={adminName ? `Atur password login baru untuk ${adminName}.` : "Atur password login baru untuk pengguna ini."}
      size="sm"
      footer={
        <>
          <Button variant="secondary" onClick={onClose} disabled={form.submitting}>
            Batal
          </Button>
          <Button type="submit" form="reset-password-form" disabled={form.submitting}>
            {form.submitting ? "Menyimpan..." : "Reset password"}
          </Button>
        </>
      }
    >
      <form id="reset-password-form" ref={formRef} noValidate onSubmit={(e) => void submit(e)} className="grid grid-cols-1 gap-4">
        <FormErrorSummary errors={form.errorList} formError={form.formError} labels={FIELD_LABELS} root={() => formRef.current} />
        <Field label="Password Baru" required name="password" error={errors.password}>
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
        <Field label="Konfirmasi Password Baru" required name="confirmPassword" error={errors.confirmPassword}>
          <Input
            type={showPassword ? "text" : "password"}
            value={values.confirmPassword}
            onChange={(e) => set("confirmPassword", e.target.value)}
            onBlur={() => blur("confirmPassword")}
            placeholder="Ulangi password baru"
          />
        </Field>
      </form>
    </Modal>
  );
}
