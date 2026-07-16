import { useState } from "react";
import { Eye, EyeOff } from "lucide-react";
import { Modal } from "@/shared/components/ui/Modal";
import { Button } from "@/shared/components/ui/Button";
import { Input, Field } from "@/shared/components/ui/Input";
import {
  resetTenantPasswordSchema,
  type ResetTenantPasswordFormValues,
} from "@/modules/platform-admin/schemas/tenant.schema";

const EMPTY_VALUES: ResetTenantPasswordFormValues = { password: "", confirmPassword: "" };

interface TenantResetPasswordModalProps {
  open: boolean;
  onClose: () => void;
  tenantName?: string;
  onSubmit: (values: ResetTenantPasswordFormValues) => void;
}

export function TenantResetPasswordModal({ open, onClose, tenantName, onSubmit }: TenantResetPasswordModalProps) {
  const [values, setValues] = useState<ResetTenantPasswordFormValues>(EMPTY_VALUES);
  const [errors, setErrors] = useState<Partial<Record<keyof ResetTenantPasswordFormValues, string>>>({});
  const [showPassword, setShowPassword] = useState(false);

  function set<K extends keyof ResetTenantPasswordFormValues>(key: K, value: ResetTenantPasswordFormValues[K]) {
    setValues((prev) => ({ ...prev, [key]: value }));
  }

  function handleSubmit() {
    const result = resetTenantPasswordSchema.safeParse(values);
    if (!result.success) {
      const fieldErrors: Partial<Record<keyof ResetTenantPasswordFormValues, string>> = {};
      for (const issue of result.error.issues) {
        fieldErrors[issue.path[0] as keyof ResetTenantPasswordFormValues] = issue.message;
      }
      setErrors(fieldErrors);
      return;
    }
    onSubmit(result.data);
    setValues(EMPTY_VALUES);
    setErrors({});
  }

  function handleClose() {
    setValues(EMPTY_VALUES);
    setErrors({});
    onClose();
  }

  return (
    <Modal
      open={open}
      onClose={handleClose}
      title="Reset Password Owner"
      description={tenantName ? `Atur password login baru untuk owner ${tenantName}.` : "Atur password login baru untuk owner tenant."}
      size="sm"
      footer={
        <>
          <Button variant="secondary" onClick={handleClose}>
            Batal
          </Button>
          <Button onClick={handleSubmit}>Reset Password</Button>
        </>
      }
    >
      <div className="grid grid-cols-1 gap-4">
        <Field label="Password Baru" required hint={errors.password}>
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
        <Field label="Konfirmasi Password Baru" required hint={errors.confirmPassword}>
          <Input
            type={showPassword ? "text" : "password"}
            value={values.confirmPassword}
            onChange={(e) => set("confirmPassword", e.target.value)}
            placeholder="Ulangi password baru"
          />
        </Field>
      </div>
    </Modal>
  );
}
