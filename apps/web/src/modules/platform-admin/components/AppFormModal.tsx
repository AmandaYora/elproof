import { useEffect, useRef } from "react";
import { Modal } from "@/shared/components/ui/Modal";
import { Button } from "@/shared/components/ui/Button";
import { Input, Field } from "@/shared/components/ui/Input";
import { createAppSchema, type CreateAppFormValues } from "@/modules/platform-admin/schemas/app.schema";
import { useZodForm } from "@/shared/forms/useZodForm";
import { FormErrorSummary } from "@/shared/forms/FormErrorSummary";

const FIELD_LABELS: Record<string, string> = { name: "Nama aplikasi", callbackUrl: "URL callback" };

interface AppFormModalProps {
  open: boolean;
  onClose: () => void;
  /** Registers, then closes the modal. Throw to keep it open: the message is shown inside it, a 422 on its fields. */
  onSubmit: (values: CreateAppFormValues) => Promise<void>;
}

const EMPTY_VALUES: CreateAppFormValues = { name: "", callbackUrl: "" };

export function AppFormModal({ open, onClose, onSubmit }: AppFormModalProps) {
  const form = useZodForm({ schema: createAppSchema, initial: EMPTY_VALUES });
  const { values, errors, set, blur } = form;
  const formRef = useRef<HTMLFormElement>(null);

  // The modal stays mounted while closed: start clean each time it opens.
  useEffect(() => {
    form.reset(EMPTY_VALUES);
  }, [open]);

  const submit = form.handleSubmit(onSubmit, {
    root: () => formRef.current,
    fallback: "Aplikasi belum terdaftar. Periksa koneksi lalu coba lagi.",
    known: (key) => key in FIELD_LABELS,
  });

  return (
    <Modal
      open={open}
      onClose={onClose}
      busy={form.submitting}
      title="Tambah Aplikasi Eksternal"
      description="Aplikasi ini akan menerima App ID dan Secret untuk memproses pembayaran melalui ElProof."
      footer={
        <>
          <Button variant="ghost" onClick={onClose} disabled={form.submitting}>
            Batal
          </Button>
          <Button type="submit" form="app-form" disabled={form.submitting}>
            {form.submitting ? "Mendaftarkan..." : "Daftarkan aplikasi"}
          </Button>
        </>
      }
    >
      <form id="app-form" ref={formRef} noValidate onSubmit={(e) => void submit(e)} className="flex flex-col gap-4">
        <FormErrorSummary errors={form.errorList} formError={form.formError} labels={FIELD_LABELS} root={() => formRef.current} />
        <Field label="Nama Aplikasi" required name="name" error={errors.name}>
          <Input value={values.name} onChange={(e) => set("name", e.target.value)} onBlur={() => blur("name")} placeholder="Contoh: Kasir Elkasir" />
        </Field>
        <Field label="URL Callback" required name="callbackUrl" error={errors.callbackUrl} hint="Menerima relay webhook HMAC-signed saat charge lunas">
          <Input
            value={values.callbackUrl}
            onChange={(e) => set("callbackUrl", e.target.value)}
            onBlur={() => blur("callbackUrl")}
            placeholder="https://aplikasi-lain.example.com/webhooks/elproof-payment"
          />
        </Field>
      </form>
    </Modal>
  );
}
