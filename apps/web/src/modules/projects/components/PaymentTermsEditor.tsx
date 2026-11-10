import type { Dispatch, SetStateAction } from "react";
import { Plus, Trash2 } from "lucide-react";
import { Button } from "@/shared/components/ui/Button";
import { Input, Field } from "@/shared/components/ui/Input";
import { CurrencyInput } from "@/shared/components/ui/CurrencyInput";
import { formatCurrency } from "@/shared/lib/formatters";
import { defaultTermTemplate } from "@/modules/projects/lib/terms";
import type { PaymentTermInput } from "@/modules/projects/types";

/**
 * The editable list of installments, shared by the schedule drawer on a
 * project and step 2 of creating one. The caller owns the rows and decides
 * when and how to save them.
 */
export function PaymentTermsEditor({
  rows,
  setRows,
  rowErrors,
  contractValue,
  bookingDate,
  eventDate,
}: {
  rows: PaymentTermInput[];
  setRows: Dispatch<SetStateAction<PaymentTermInput[]>>;
  rowErrors: Record<number, string>;
  contractValue: number;
  bookingDate: string;
  eventDate: string;
}) {
  const total = rows.reduce((sum, r) => sum + r.amount, 0);
  const over = contractValue > 0 && total > contractValue;

  function update(index: number, patch: Partial<PaymentTermInput>) {
    setRows((prev) => prev.map((r, i) => (i === index ? { ...r, ...patch } : r)));
  }

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <p className={`text-[13px] font-medium tabular-nums ${over ? "text-danger-text" : "text-text-secondary"}`}>
          Total {formatCurrency(total)} dari {formatCurrency(contractValue)}
          {!over && contractValue > 0 && ` · sisa ${formatCurrency(contractValue - total)}`}
        </p>
        <Button variant="ghost" size="sm" onClick={() => setRows(defaultTermTemplate(contractValue, bookingDate, eventDate))}>
          Pakai templat 30/40/30
        </Button>
      </div>

      {rows.map((row, i) => (
        <fieldset key={i} className="grid grid-cols-1 gap-3 rounded-md border border-border p-3 sm:grid-cols-[1fr_1fr_1fr_auto] sm:items-start">
          <legend className="sr-only">Termin {i + 1}</legend>
          <Field label="Nama termin" error={rowErrors[i]}>
            <Input value={row.label} onChange={(e) => update(i, { label: e.target.value })} aria-invalid={rowErrors[i] && !row.label.trim() ? true : undefined} />
          </Field>
          <Field label="Nominal (Rp)">
            <CurrencyInput value={row.amount} onChange={(n) => update(i, { amount: n })} aria-invalid={rowErrors[i] && row.amount <= 0 ? true : undefined} />
          </Field>
          <Field label="Jatuh tempo">
            <Input type="date" value={row.dueDate} onChange={(e) => update(i, { dueDate: e.target.value })} aria-invalid={rowErrors[i] && !row.dueDate ? true : undefined} />
          </Field>
          <button
            type="button"
            aria-label={`Hapus termin ${i + 1}`}
            onClick={() => setRows((prev) => prev.filter((_, idx) => idx !== i))}
            className="mt-6 flex h-11 w-11 items-center justify-center rounded-md text-text-secondary hover:bg-surface-muted hover:text-danger-text focus-visible:outline focus-visible:outline-2 focus-visible:outline-ring sm:h-9 sm:w-9"
          >
            <Trash2 className="h-4 w-4" />
          </button>
        </fieldset>
      ))}

      <div>
        <Button
          variant="secondary"
          size="sm"
          icon={<Plus className="h-3.5 w-3.5" />}
          onClick={() => setRows((prev) => [...prev, { label: "", amount: Math.max(contractValue - total, 0), dueDate: eventDate }])}
        >
          Tambah termin
        </Button>
      </div>
    </div>
  );
}

/** Marks rows that cannot be saved; returns the problems keyed by row. */
export function validateTermRows(rows: PaymentTermInput[]): Record<number, string> {
  const problems: Record<number, string> = {};
  rows.forEach((r, i) => {
    if (!r.label.trim()) problems[i] = "Isi nama termin";
    else if (r.amount <= 0) problems[i] = "Nominal harus lebih dari 0";
    else if (!r.dueDate) problems[i] = "Isi tanggal jatuh tempo";
  });
  return problems;
}
