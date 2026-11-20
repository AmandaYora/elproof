import { useState } from "react";
import { Modal } from "@/shared/components/ui/Modal";
import { Button } from "@/shared/components/ui/Button";
import { Field, Input } from "@/shared/components/ui/Input";
import { getApiErrorMessage, getApiFieldErrors } from "@/shared/lib/api-error";
import { formatDate, todayISO } from "@/shared/lib/formatters";
import { toast } from "@/shared/stores/useToastStore";
import { useQuotationStore } from "@/modules/offers/stores/useQuotationStore";
import { daysFromToday } from "@/modules/offers/lib/display";
import type { Quotation } from "@/modules/offers/types";

/** "Perpanjang masa berlaku": a new last day; an expired quotation becomes a draft again. */
export default function ExtendDialog({ quotation, onClose }: { quotation: Quotation; onClose: () => void }) {
  const extend = useQuotationStore((s) => s.extend);
  const base = quotation.validUntil && quotation.validUntil >= todayISO() ? quotation.validUntil : todayISO();
  const [date, setDate] = useState(() => {
    const d = new Date(`${base}T00:00:00`);
    d.setDate(d.getDate() + 14);
    const pad = (n: number) => String(n).padStart(2, "0");
    return base === todayISO() ? daysFromToday(14) : `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
  });
  const [saving, setSaving] = useState(false);
  const [fieldError, setFieldError] = useState<string | undefined>();
  const [error, setError] = useState<string | null>(null);

  async function save() {
    if (!date) {
      setFieldError("Pilih tanggal berlaku sampai");
      return;
    }
    setSaving(true);
    setError(null);
    setFieldError(undefined);
    try {
      await extend(quotation.id, date);
      toast.success("Masa berlaku diperpanjang");
      onClose();
    } catch (err) {
      const fields = getApiFieldErrors(err);
      if (fields.validUntil) setFieldError(fields.validUntil);
      else setError(getApiErrorMessage(err, "Masa berlaku belum diperpanjang. Periksa koneksi lalu coba lagi."));
    } finally {
      setSaving(false);
    }
  }

  return (
    <Modal
      open
      onClose={onClose}
      size="sm"
      title="Perpanjang masa berlaku"
      description={quotation.validUntil ? `Sekarang berlaku sampai ${formatDate(quotation.validUntil)}.` : undefined}
      busy={saving}
      error={error}
      footer={
        <>
          <Button variant="secondary" onClick={onClose} disabled={saving}>
            Batal
          </Button>
          <Button onClick={() => void save()} disabled={saving}>
            {saving ? "Menyimpan..." : "Perpanjang"}
          </Button>
        </>
      }
    >
      <Field label="Berlaku sampai" required name="validUntil" error={fieldError}>
        <Input type="date" min={todayISO()} value={date} onChange={(e) => setDate(e.target.value)} />
      </Field>
    </Modal>
  );
}
