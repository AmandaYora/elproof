import { useEffect, useRef } from "react";
import { Modal } from "@/shared/components/ui/Modal";
import { Button } from "@/shared/components/ui/Button";
import { Input, Field } from "@/shared/components/ui/Input";
import { useZodForm } from "@/shared/forms/useZodForm";
import { FormErrorSummary } from "@/shared/forms/FormErrorSummary";
import { eventKindSchema, type EventKindFormValues } from "@/modules/event-kinds/schemas/event-kind.schema";
import type { EventKind } from "@/modules/event-kinds/types";

interface EventKindFormModalProps {
  open: boolean;
  onClose: () => void;
  /** Saves, then closes. Throw to keep the dialog open: a 422 lands on the field, anything else inside the dialog. */
  onSubmit: (values: EventKindFormValues) => Promise<void>;
  /** The kind being renamed; undefined adds a new one. */
  kind?: EventKind;
}

/** One field: the kind's name (DESAIN.md §6.7). */
export function EventKindFormModal({ open, onClose, onSubmit, kind }: EventKindFormModalProps) {
  const form = useZodForm({ schema: eventKindSchema, initial: { name: kind?.name ?? "" } });
  const { values, errors, set, blur } = form;
  const formRef = useRef<HTMLFormElement>(null);

  useEffect(() => {
    form.reset({ name: kind?.name ?? "" });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [kind, open]);

  const submit = form.handleSubmit(onSubmit, { root: () => formRef.current, fallback: "Jenis acara belum tersimpan. Periksa koneksi lalu coba lagi." });

  return (
    <Modal
      open={open}
      onClose={onClose}
      size="sm"
      title={kind ? "Ganti nama jenis acara" : "Tambah jenis acara"}
      description={kind ? "Nama baru dipakai di paket dan penawaran berikutnya." : "Misalnya Mapacci, Midodareni, atau Tasyakuran."}
      busy={form.submitting}
      footer={
        <>
          <Button variant="secondary" onClick={onClose} disabled={form.submitting}>
            Batal
          </Button>
          <Button type="submit" form="event-kind-form" disabled={form.submitting}>
            {form.submitting ? "Menyimpan..." : kind ? "Simpan nama" : "Tambah jenis acara"}
          </Button>
        </>
      }
    >
      <form id="event-kind-form" ref={formRef} noValidate onSubmit={(e) => void submit(e)} className="flex flex-col gap-4">
        <FormErrorSummary errors={form.errorList} formError={form.formError} labels={{ name: "Nama jenis acara" }} root={() => formRef.current} />
        <Field label="Nama jenis acara" required name="name" error={errors.name}>
          <Input value={values.name} maxLength={60} onChange={(e) => set("name", e.target.value)} onBlur={() => blur("name")} placeholder="cth. Mapacci" autoFocus />
        </Field>
      </form>
    </Modal>
  );
}
