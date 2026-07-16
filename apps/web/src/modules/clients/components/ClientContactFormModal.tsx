import { useState } from "react";
import { Modal } from "@/shared/components/ui/Modal";
import { Button } from "@/shared/components/ui/Button";
import { Input, Field } from "@/shared/components/ui/Input";
import { clientContactSchema, type ClientContactFormValues } from "@/modules/clients/schemas/client.schema";

interface ClientContactFormModalProps {
  open: boolean;
  onClose: () => void;
  onSubmit: (values: ClientContactFormValues) => void;
  initialValues: { name: string; phone: string; email: string };
  title?: string;
  description?: string;
}

export function ClientContactFormModal({
  open,
  onClose,
  onSubmit,
  initialValues,
  title = "Ubah Kontak Client",
  description = "Perbarui nama dan informasi kontak client.",
}: ClientContactFormModalProps) {
  const [values, setValues] = useState<ClientContactFormValues>(initialValues);
  const [errors, setErrors] = useState<Partial<Record<keyof ClientContactFormValues, string>>>({});

  function set<K extends keyof ClientContactFormValues>(key: K, value: ClientContactFormValues[K]) {
    setValues((prev) => ({ ...prev, [key]: value }));
  }

  function handleSubmit() {
    const result = clientContactSchema.safeParse(values);
    if (!result.success) {
      const fieldErrors: Partial<Record<keyof ClientContactFormValues, string>> = {};
      for (const issue of result.error.issues) {
        const key = issue.path[0] as keyof ClientContactFormValues;
        fieldErrors[key] = issue.message;
      }
      setErrors(fieldErrors);
      return;
    }
    onSubmit(result.data);
    setErrors({});
  }

  return (
    <Modal
      open={open}
      onClose={onClose}
      title={title}
      description={description}
      footer={
        <>
          <Button variant="secondary" onClick={onClose}>Batal</Button>
          <Button onClick={handleSubmit}>Simpan Perubahan</Button>
        </>
      }
    >
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <div className="sm:col-span-2">
          <Field label="Nama" required hint={errors.name}>
            <Input value={values.name} onChange={(e) => set("name", e.target.value)} placeholder="Nama lengkap" />
          </Field>
        </div>
        <Field label="Nomor Telepon" required hint={errors.phone}>
          <Input value={values.phone} onChange={(e) => set("phone", e.target.value)} placeholder="08xx-xxxx-xxxx" />
        </Field>
        <Field label="Email" required hint={errors.email}>
          <Input type="email" value={values.email} onChange={(e) => set("email", e.target.value)} placeholder="nama@email.com" />
        </Field>
      </div>
    </Modal>
  );
}
