import { useState } from "react";
import { Modal } from "@/shared/components/ui/Modal";
import { Button } from "@/shared/components/ui/Button";
import { Field, Textarea } from "@/shared/components/ui/Input";
import { getApiErrorMessage } from "@/shared/lib/api-error";
import { isStaleConflict } from "@/shared/lib/concurrency";
import { toast } from "@/shared/stores/useToastStore";
import { useQuotationStore } from "@/modules/offers/stores/useQuotationStore";
import type { QuotationDetail } from "@/modules/offers/types";

const MAX = 20000;

/** "Syarat & ketentuan" of one quotation (it starts from the one in Profil usaha › Penawaran). */
export default function QuotationTermsTextModal({ detail, onClose }: { detail: QuotationDetail; onClose: () => void }) {
  const saveTermsText = useQuotationStore((s) => s.saveTermsText);
  const [text, setText] = useState(detail.termsText);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const tooLong = text.length > MAX;

  async function save() {
    if (tooLong) return;
    setSaving(true);
    setError(null);
    try {
      await saveTermsText(detail.quotation.id, text, detail.quotation.termsUpdatedAt);
      toast.success("Syarat & ketentuan disimpan");
      onClose();
    } catch (err) {
      if (!isStaleConflict(err)) setError(getApiErrorMessage(err, "Syarat & ketentuan belum tersimpan. Periksa koneksi lalu coba lagi."));
    } finally {
      setSaving(false);
    }
  }

  return (
    <Modal
      open
      onClose={onClose}
      size="lg"
      title="Ubah syarat & ketentuan"
      description="Tercetak di halaman terakhir PDF penawaran."
      busy={saving}
      error={error}
      footer={
        <>
          <Button variant="secondary" onClick={onClose} disabled={saving}>
            Batal
          </Button>
          <Button onClick={() => void save()} disabled={saving}>
            {saving ? "Menyimpan..." : "Simpan syarat & ketentuan"}
          </Button>
        </>
      }
    >
      <Field
        label="Syarat & ketentuan"
        name="termsText"
        error={tooLong ? `Paling banyak ${MAX.toLocaleString("id-ID")} karakter` : undefined}
        hint="Satu poin per baris."
      >
        <Textarea rows={14} value={text} onChange={(e) => setText(e.target.value)} />
      </Field>
    </Modal>
  );
}
