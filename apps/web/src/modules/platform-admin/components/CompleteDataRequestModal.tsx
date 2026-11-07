import { useState } from "react";
import { Modal } from "@/shared/components/ui/Modal";
import { Button } from "@/shared/components/ui/Button";
import { Field, Select, Textarea } from "@/shared/components/ui/Input";
import type { DataRequest } from "@/modules/platform-admin/data/console.types";
import { DATA_REQUEST_KIND_LABEL } from "@/modules/platform-admin/lib/data-requests";
import { completeDataRequestSchema } from "@/modules/platform-admin/schemas/console.schema";

interface Props {
  request: DataRequest | undefined;
  onClose: () => void;
  onSubmit: (status: "done" | "rejected", note: string) => Promise<void>;
  error: string | null;
  isSubmitting: boolean;
}

export function CompleteDataRequestModal({ request, onClose, onSubmit, error, isSubmitting }: Props) {
  const [status, setStatus] = useState<"done" | "rejected">("done");
  const [note, setNote] = useState("");
  const [fieldError, setFieldError] = useState<string | undefined>();

  async function handleSubmit() {
    const parsed = completeDataRequestSchema.safeParse({ status, note });
    if (!parsed.success) {
      setFieldError(parsed.error.issues[0]?.message);
      return;
    }
    setFieldError(undefined);
    await onSubmit(parsed.data.status, parsed.data.note);
  }

  return (
    <Modal
      open={request !== undefined}
      onClose={onClose}
      busy={isSubmitting}
      error={error}
      size="sm"
      title="Tutup permintaan data"
      description={request ? `${DATA_REQUEST_KIND_LABEL[request.kind]} dari ${request.requestedBy} untuk ${request.tenantName || "tenant ini"}.` : undefined}
      footer={
        <>
          <Button variant="secondary" onClick={onClose} disabled={isSubmitting}>
            Batal
          </Button>
          <Button onClick={() => void handleSubmit()} disabled={isSubmitting}>
            {isSubmitting ? "Menyimpan..." : "Tutup permintaan"}
          </Button>
        </>
      }
    >
      <div className="grid grid-cols-1 gap-4">
        <Field label="Hasil">
          <Select value={status} onChange={(e) => setStatus(e.target.value as "done" | "rejected")}>
            <option value="done">Selesai dipenuhi</option>
            <option value="rejected">Ditolak</option>
          </Select>
        </Field>
        <Field
          label={status === "done" ? "Bukti penyelesaian" : "Alasan penolakan"}
          required
          error={fieldError}
          hint={status === "done" ? "Misalnya: berkas dikirim ke email Owner tanggal ..., atau data dihapus tanggal ..." : "Jelaskan kepada pemohon mengapa permintaan tidak bisa dipenuhi."}
        >
          <Textarea rows={3} value={note} onChange={(e) => setNote(e.target.value)} aria-invalid={fieldError ? true : undefined} />
        </Field>
      </div>
    </Modal>
  );
}
