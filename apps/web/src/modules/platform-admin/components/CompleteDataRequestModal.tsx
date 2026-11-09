import { useEffect, useRef } from "react";
import { Modal } from "@/shared/components/ui/Modal";
import { Button } from "@/shared/components/ui/Button";
import { Field, Select, Textarea } from "@/shared/components/ui/Input";
import type { DataRequest } from "@/modules/platform-admin/data/console.types";
import { DATA_REQUEST_KIND_LABEL } from "@/modules/platform-admin/lib/data-requests";
import { completeDataRequestSchema } from "@/modules/platform-admin/schemas/console.schema";
import { useZodForm } from "@/shared/forms/useZodForm";
import { FormErrorSummary } from "@/shared/forms/FormErrorSummary";

const FIELD_LABELS: Record<string, string> = { status: "Hasil", note: "Bukti penyelesaian atau alasan penolakan" };

interface Props {
  request: DataRequest | undefined;
  onClose: () => void;
  /** Closes the request, then the modal. Throw to keep it open: the message is shown inside it. */
  onSubmit: (status: "done" | "rejected", note: string) => Promise<void>;
}

export function CompleteDataRequestModal({ request, onClose, onSubmit }: Props) {
  const form = useZodForm({ schema: completeDataRequestSchema, initial: { status: "done", note: "" } });
  const { values, errors, set, blur } = form;
  const formRef = useRef<HTMLFormElement>(null);

  useEffect(() => {
    form.reset({ status: "done", note: "" });
  }, [request?.id]);

  const submit = form.handleSubmit(async (parsed) => onSubmit(parsed.status, parsed.note), {
    root: () => formRef.current,
    fallback: "Permintaan belum bisa ditutup. Periksa koneksi lalu coba lagi.",
    known: (key) => key in FIELD_LABELS,
  });

  return (
    <Modal
      open={request !== undefined}
      onClose={onClose}
      busy={form.submitting}
      size="sm"
      title="Tutup permintaan data"
      description={request ? `${DATA_REQUEST_KIND_LABEL[request.kind]} dari ${request.requestedBy} untuk ${request.tenantName || "tenant ini"}.` : undefined}
      footer={
        <>
          <Button variant="secondary" onClick={onClose} disabled={form.submitting}>
            Batal
          </Button>
          <Button type="submit" form="complete-data-request-form" disabled={form.submitting}>
            {form.submitting ? "Menyimpan..." : "Tutup permintaan"}
          </Button>
        </>
      }
    >
      <form id="complete-data-request-form" ref={formRef} noValidate onSubmit={(e) => void submit(e)} className="grid grid-cols-1 gap-4">
        <FormErrorSummary errors={form.errorList} formError={form.formError} labels={FIELD_LABELS} root={() => formRef.current} />
        <Field label="Hasil" name="status" error={errors.status}>
          <Select value={values.status} onChange={(e) => set("status", e.target.value as "done" | "rejected")}>
            <option value="done">Selesai dipenuhi</option>
            <option value="rejected">Ditolak</option>
          </Select>
        </Field>
        <Field
          label={values.status === "done" ? "Bukti penyelesaian" : "Alasan penolakan"}
          required
          name="note"
          error={errors.note}
          hint={values.status === "done" ? "Misalnya: berkas dikirim ke email Owner tanggal ..., atau data dihapus tanggal ..." : "Jelaskan kepada pemohon mengapa permintaan tidak bisa dipenuhi."}
        >
          <Textarea rows={3} value={values.note} onChange={(e) => set("note", e.target.value)} onBlur={() => blur("note")} />
        </Field>
      </form>
    </Modal>
  );
}
