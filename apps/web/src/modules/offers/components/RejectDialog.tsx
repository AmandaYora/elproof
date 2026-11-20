import { useState } from "react";
import { Modal } from "@/shared/components/ui/Modal";
import { Button } from "@/shared/components/ui/Button";
import { Field, Textarea } from "@/shared/components/ui/Input";
import { getApiErrorMessage } from "@/shared/lib/api-error";
import { toast } from "@/shared/stores/useToastStore";
import { useQuotationStore } from "@/modules/offers/stores/useQuotationStore";
import type { Quotation } from "@/modules/offers/types";

/** "Tolak": the couple declined. The reason is optional; "Buka lagi" undoes it. */
export default function RejectDialog({ quotation, onClose }: { quotation: Quotation; onClose: () => void }) {
  const setStatus = useQuotationStore((s) => s.setStatus);
  const [reason, setReason] = useState("");
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function reject() {
    setSaving(true);
    setError(null);
    try {
      await setStatus(quotation.id, { status: "Rejected", reason: reason.trim() });
      toast.success("Penawaran ditandai ditolak");
      onClose();
    } catch (err) {
      setError(getApiErrorMessage(err, "Penawaran belum ditandai ditolak. Periksa koneksi lalu coba lagi."));
    } finally {
      setSaving(false);
    }
  }

  return (
    <Modal
      open
      onClose={onClose}
      size="sm"
      title={`Tolak penawaran ${quotation.number}`}
      description="Penawaran menjadi baca-saja. Bisa dibuka lagi bila klien berubah pikiran."
      busy={saving}
      error={error}
      footer={
        <>
          <Button variant="secondary" onClick={onClose} disabled={saving}>
            Batal
          </Button>
          <Button variant="danger" onClick={() => void reject()} disabled={saving || reason.length > 500}>
            {saving ? "Menyimpan..." : "Tandai ditolak"}
          </Button>
        </>
      }
    >
      <Field label="Alasan (opsional)" name="reason" error={reason.length > 500 ? "Alasan maksimal 500 karakter" : undefined} hint="cth. Memilih WO lain, anggaran belum cukup.">
        <Textarea rows={3} value={reason} onChange={(e) => setReason(e.target.value)} />
      </Field>
    </Modal>
  );
}
