import { useState } from "react";
import { Modal } from "@/shared/components/ui/Modal";
import { Button } from "@/shared/components/ui/Button";
import { Input, Field } from "@/shared/components/ui/Input";
import { createAppSchema, type CreateAppFormValues } from "@/modules/platform-admin/schemas/app.schema";

interface AppFormModalProps {
  open: boolean;
  onClose: () => void;
  onSubmit: (values: CreateAppFormValues) => void;
}

const EMPTY_VALUES: CreateAppFormValues = { name: "", callbackUrl: "" };

export function AppFormModal({ open, onClose, onSubmit }: AppFormModalProps) {
  const [values, setValues] = useState<CreateAppFormValues>(EMPTY_VALUES);
  const [errors, setErrors] = useState<Partial<Record<keyof CreateAppFormValues, string>>>({});

  function set<K extends keyof CreateAppFormValues>(key: K, value: CreateAppFormValues[K]) {
    setValues((prev) => ({ ...prev, [key]: value }));
  }

  function handleSubmit() {
    const result = createAppSchema.safeParse(values);
    if (!result.success) {
      const fieldErrors: Partial<Record<keyof CreateAppFormValues, string>> = {};
      for (const issue of result.error.issues) {
        const key = issue.path[0] as keyof CreateAppFormValues;
        fieldErrors[key] = issue.message;
      }
      setErrors(fieldErrors);
      return;
    }
    setErrors({});
    onSubmit(result.data);
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
      title="Tambah Aplikasi Eksternal"
      description="Aplikasi lain akan menggunakan appId + secret ini untuk membuat charge lewat dompet merchant ElProof."
      footer={
        <>
          <Button variant="ghost" onClick={handleClose}>Batal</Button>
          <Button onClick={handleSubmit}>Daftarkan</Button>
        </>
      }
    >
      <div className="flex flex-col gap-4">
        <Field label="Nama Aplikasi" required>
          <Input value={values.name} onChange={(e) => set("name", e.target.value)} placeholder="Contoh: Kasir Elkasir" />
          {errors.name && <p className="text-xs text-danger">{errors.name}</p>}
        </Field>
        <Field label="URL Callback" required hint="Menerima relay webhook HMAC-signed saat charge lunas">
          <Input
            value={values.callbackUrl}
            onChange={(e) => set("callbackUrl", e.target.value)}
            placeholder="https://aplikasi-lain.example.com/webhooks/elproof-payment"
          />
          {errors.callbackUrl && <p className="text-xs text-danger">{errors.callbackUrl}</p>}
        </Field>
      </div>
    </Modal>
  );
}
