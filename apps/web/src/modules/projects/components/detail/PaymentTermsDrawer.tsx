import { useState } from "react";
import { Button } from "@/shared/components/ui/Button";
import { Drawer } from "@/shared/components/ui/Drawer";
import { toast } from "@/shared/stores/useToastStore";
import { getApiErrorMessage, getApiFieldErrors } from "@/shared/lib/api-error";
import { formatCurrency } from "@/shared/lib/formatters";
import { useProjectStore } from "@/modules/projects/stores/useProjectStore";
import { defaultTermTemplate } from "@/modules/projects/lib/terms";
import { PaymentTermsEditor, validateTermRows } from "@/modules/projects/components/PaymentTermsEditor";
import type { PaymentTermInput } from "@/modules/projects/types";

/** "Atur jadwal termin": edits the whole schedule. Loaded on the click that opens it (PaymentTermsSection). */
export function PaymentTermsDrawer({
  projectId,
  contractValue,
  bookingDate,
  eventDate,
  initial,
  onClose,
}: {
  projectId: string;
  contractValue: number;
  bookingDate: string;
  eventDate: string;
  initial: PaymentTermInput[];
  onClose: () => void;
}) {
  const save = useProjectStore((s) => s.savePaymentTerms);
  const [rows, setRows] = useState<PaymentTermInput[]>(() =>
    initial.length > 0 ? initial : defaultTermTemplate(contractValue, bookingDate, eventDate),
  );
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [rowErrors, setRowErrors] = useState<Record<number, string>>({});

  const total = rows.reduce((sum, r) => sum + r.amount, 0);
  const over = contractValue > 0 && total > contractValue;

  async function handleSave() {
    const problems = validateTermRows(rows);
    setRowErrors(problems);
    if (Object.keys(problems).length > 0 || over) {
      setError(over ? `Jumlah termin melebihi nilai kontrak ${formatCurrency(contractValue)}.` : "Lengkapi dulu bagian yang ditandai.");
      return;
    }
    setError(null);
    setSaving(true);
    try {
      await save(projectId, rows);
      toast.success("Jadwal termin disimpan");
      onClose();
    } catch (err) {
      // The server reports every bad row at once ("terms[2].amount"): mark each row, keep what was typed.
      const fields = getApiFieldErrors(err);
      const marked: Record<number, string> = {};
      for (const [key, message] of Object.entries(fields)) {
        const row = /^terms\[(\d+)\]/.exec(key);
        if (row) marked[Number(row[1])] ??= message;
      }
      setRowErrors(marked);
      setError(Object.keys(marked).length > 0 ? "Lengkapi dulu bagian yang ditandai." : getApiErrorMessage(err, "Jadwal belum tersimpan. Periksa koneksi lalu coba lagi."));
    } finally {
      setSaving(false);
    }
  }

  return (
    <Drawer
      open
      onClose={onClose}
      busy={saving}
      size="lg"
      title="Atur jadwal termin"
      description="Jumlah semua termin tidak boleh melebihi nilai kontrak."
      footer={
        <>
          <Button variant="secondary" onClick={onClose} disabled={saving}>Batal</Button>
          <Button onClick={() => void handleSave()} disabled={saving}>{saving ? "Menyimpan..." : "Simpan jadwal"}</Button>
        </>
      }
    >
      <div className="flex flex-col gap-4">
        <PaymentTermsEditor
          rows={rows}
          setRows={setRows}
          rowErrors={rowErrors}
          contractValue={contractValue}
          bookingDate={bookingDate}
          eventDate={eventDate}
        />

        {error && (
          <p role="alert" className="rounded-md border border-danger/30 bg-danger-soft px-3.5 py-2.5 text-[13px] font-medium text-danger-text">
            {error}
          </p>
        )}
      </div>
    </Drawer>
  );
}
