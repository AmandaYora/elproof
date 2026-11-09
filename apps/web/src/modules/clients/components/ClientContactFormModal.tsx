import { useRef } from "react";
import { Modal } from "@/shared/components/ui/Modal";
import { Button } from "@/shared/components/ui/Button";
import { Input, Field } from "@/shared/components/ui/Input";
import { clientContactSchema, type ClientContactFormValues } from "@/modules/clients/schemas/client.schema";
import { PortalRoleSelect } from "@/modules/clients/components/PortalRoleSelect";
import { useZodForm } from "@/shared/forms/useZodForm";
import { FormErrorSummary } from "@/shared/forms/FormErrorSummary";

const FIELD_LABELS: Record<string, string> = { name: "Nama", phone: "Nomor WhatsApp", email: "Email", accessRoleId: "Akses portal" };

interface ClientContactFormModalProps {
  open: boolean;
  onClose: () => void;
  /** Saves, then closes the modal. Throw to keep it open: a 422 is shown on its fields, anything else inside the modal. */
  onSubmit: (values: ClientContactFormValues) => Promise<void>;
  initialValues: { name: string; phone: string; email: string; accessRoleId?: number };
  username?: string;
  title?: string;
  description?: string;
  submitLabel?: string;
}

export function ClientContactFormModal({
  open,
  onClose,
  onSubmit,
  initialValues,
  username,
  title = "Ubah Kontak Client",
  description = "Perbarui nama dan informasi kontak client.",
  submitLabel = "Simpan perubahan",
}: ClientContactFormModalProps) {
  // accessRoleId stays undefined when the caller does not manage the portal role (the picker is then hidden).
  const form = useZodForm({ schema: clientContactSchema, initial: initialValues as Required<ClientContactFormValues> });
  const { values, errors, set, blur } = form;
  const formRef = useRef<HTMLFormElement>(null);

  const submit = form.handleSubmit(onSubmit, { root: () => formRef.current, fallback: "Kontak belum tersimpan. Periksa koneksi lalu coba lagi." });

  return (
    <Modal
      open={open}
      onClose={onClose}
      title={title}
      description={description}
      busy={form.submitting}
      footer={
        <>
          <Button variant="secondary" onClick={onClose} disabled={form.submitting}>Batal</Button>
          <Button type="submit" form="client-contact-form" disabled={form.submitting}>
            {form.submitting ? "Menyimpan..." : submitLabel}
          </Button>
        </>
      }
    >
      <form id="client-contact-form" ref={formRef} noValidate onSubmit={(e) => void submit(e)} className="flex flex-col gap-4">
        <FormErrorSummary errors={form.errorList} formError={form.formError} labels={FIELD_LABELS} root={() => formRef.current} />
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <div className="sm:col-span-2">
            <Field label="Nama" required name="name" error={errors.name}>
              <Input value={values.name} onChange={(e) => set("name", e.target.value)} onBlur={() => blur("name")} placeholder="Nama lengkap" />
            </Field>
          </div>
          <Field label="Nomor WhatsApp" required name="phone" error={errors.phone}>
            <Input value={values.phone} onChange={(e) => set("phone", e.target.value)} onBlur={() => blur("phone")} placeholder="08xx-xxxx-xxxx" inputMode="tel" />
          </Field>
          <Field label="Email" name="email" error={errors.email} hint="Opsional">
            <Input type="email" value={values.email} onChange={(e) => set("email", e.target.value)} onBlur={() => blur("email")} placeholder="nama@email.com" />
          </Field>
          {values.accessRoleId !== undefined && (
            <div className="sm:col-span-2" data-field="accessRoleId">
              <PortalRoleSelect value={values.accessRoleId} onChange={(id) => set("accessRoleId", id)} />
            </div>
          )}
          {username !== undefined && (
            <div className="sm:col-span-2">
              <Field label="Username">
                <Input value={username} disabled />
              </Field>
            </div>
          )}
        </div>
      </form>
    </Modal>
  );
}
