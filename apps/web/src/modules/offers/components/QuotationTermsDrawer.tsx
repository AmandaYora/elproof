import { useRef } from "react";
import { ArrowDown, ArrowUp, Plus, Trash2 } from "lucide-react";
import { Drawer } from "@/shared/components/ui/Drawer";
import { Button } from "@/shared/components/ui/Button";
import { CurrencyInput } from "@/shared/components/ui/CurrencyInput";
import { IconActionButton } from "@/shared/components/ui/IconActionButton";
import { Field, Input, Select } from "@/shared/components/ui/Input";
import { FormErrorSummary } from "@/shared/forms/FormErrorSummary";
import { useZodForm } from "@/shared/forms/useZodForm";
import { isStaleConflict } from "@/shared/lib/concurrency";
import { formatCurrency } from "@/shared/lib/formatters";
import { TERM_DUE_BASIS_LABELS } from "@/shared/lib/labels";
import { toast } from "@/shared/stores/useToastStore";
import { termsSchema, TERM_DUE_BASIS_OPTIONS, type TermRowValues } from "@/modules/offers/schemas/quotation.schema";
import { useQuotationStore } from "@/modules/offers/stores/useQuotationStore";
import type { QuotationDetail } from "@/modules/offers/types";

type Row = TermRowValues;

function rowFromTerm(t: QuotationDetail["terms"][number]): Row {
  return {
    label: t.label,
    mode: t.percent !== null ? "percent" : "amount",
    percent: t.percent ?? 0,
    amount: t.amount ?? 0,
    dueBasis: t.dueBasis,
    days: Math.abs(t.dueOffsetDays),
    before: t.dueBasis === "Event" && t.dueOffsetDays < 0,
    dueEventId: t.dueEventId,
  };
}

const emptyRow = (): Row => ({ label: "", mode: "percent", percent: 0, amount: 0, dueBasis: "Event", days: 7, before: true, dueEventId: "" });

/**
 * "Termin pembayaran" of a quotation: each a percentage of the contract value or a fixed amount, due a number of days
 * after acceptance or before/after an event. Together they must come to the contract value before a revision can be
 * made (the server says so as "terms_uneven").
 */
export default function QuotationTermsDrawer({ detail, onClose }: { detail: QuotationDetail; onClose: () => void }) {
  const saveTerms = useQuotationStore((s) => s.saveTerms);
  const formRef = useRef<HTMLFormElement>(null);
  const total = detail.totals?.total ?? null;
  const form = useZodForm({ schema: termsSchema, initial: { terms: detail.terms.length > 0 ? detail.terms.map(rowFromTerm) : [emptyRow()] } });
  const rows = form.values.terms as Row[];
  const { errors } = form;

  function patch(i: number, next: Partial<Row>) {
    form.set(
      "terms",
      rows.map((r, j) => (j === i ? { ...r, ...next } : r)),
    );
  }
  function move(i: number, d: -1 | 1) {
    const next = [...rows];
    const [r] = next.splice(i, 1);
    next.splice(i + d, 0, r);
    form.set("terms", next);
  }

  const submit = form.handleSubmit(
    async (parsed) => {
      const input = parsed.terms.map((t) => ({
        label: t.label,
        percent: t.mode === "percent" ? t.percent : null,
        amount: t.mode === "amount" ? t.amount : null,
        dueBasis: t.dueBasis,
        dueOffsetDays: t.dueBasis === "Event" && t.before ? -t.days : t.days,
        dueEventId: t.dueBasis === "Event" ? t.dueEventId : "",
      }));
      try {
        await saveTerms(detail.quotation.id, input, detail.quotation.termsUpdatedAt);
      } catch (err) {
        if (isStaleConflict(err)) return;
        throw err;
      }
      toast.success("Termin disimpan");
      onClose();
    },
    { root: () => formRef.current, fallback: "Termin belum tersimpan. Periksa koneksi lalu coba lagi.", known: (key) => key.startsWith("terms") },
  );

  const percentSum = rows.reduce((s, r) => s + (r.mode === "percent" ? Number(r.percent) || 0 : 0), 0);
  const amountSum = rows.reduce((s, r) => s + (r.mode === "amount" ? Number(r.amount) || 0 : 0), 0);
  const comesTo = total === null ? null : Math.round((total * percentSum) / 100) + amountSum;

  return (
    <Drawer
      open
      onClose={onClose}
      size="lg"
      busy={form.submitting}
      title="Ubah termin pembayaran"
      description="Jumlah semua termin harus sama dengan nilai kontrak sebelum revisi dibuat."
      footer={
        <>
          <Button variant="secondary" onClick={onClose} disabled={form.submitting}>
            Batal
          </Button>
          <Button type="submit" form="quotation-terms-form" disabled={form.submitting}>
            {form.submitting ? "Menyimpan..." : "Simpan termin"}
          </Button>
        </>
      }
    >
      <form id="quotation-terms-form" ref={formRef} noValidate onSubmit={(e) => void submit(e)} className="flex flex-col gap-5">
        <FormErrorSummary errors={form.errorList} formError={form.formError} root={() => formRef.current} />
        {errors.terms && (
          <p role="alert" className="text-[13px] font-medium text-danger-text">
            {errors.terms}
          </p>
        )}
        <ol className="flex flex-col gap-4">
          {rows.map((r, i) => {
            const key = (f: string) => `terms[${i}].${f}`;
            const amountNow = r.mode === "percent" && total !== null ? Math.round((total * (Number(r.percent) || 0)) / 100) : null;
            return (
              <li key={i} className="flex flex-col gap-3 rounded-md border border-border p-4">
                <div className="flex items-center justify-between gap-2">
                  <p className="text-[13px] font-semibold text-foreground">Termin {i + 1}</p>
                  <div className="flex gap-1">
                    <IconActionButton label={`Naikkan termin ${i + 1}`} icon={ArrowUp} onClick={() => move(i, -1)} disabled={i === 0} />
                    <IconActionButton label={`Turunkan termin ${i + 1}`} icon={ArrowDown} onClick={() => move(i, 1)} disabled={i === rows.length - 1} />
                    <IconActionButton
                      label={`Hapus termin ${i + 1}`}
                      icon={Trash2}
                      onClick={() =>
                        form.set(
                          "terms",
                          rows.filter((_, j) => j !== i),
                        )
                      }
                      disabled={rows.length === 1}
                    />
                  </div>
                </div>
                <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
                  <Field label="Nama termin" required name={key("label")} error={errors[key("label")]}>
                    <Input value={r.label} onChange={(e) => patch(i, { label: e.target.value })} placeholder="cth. DP, Termin 2, Pelunasan" />
                  </Field>
                  <fieldset className="flex flex-col gap-1.5">
                    <legend className="mb-1.5 text-[13px] font-medium text-foreground">Besarnya</legend>
                    <div className="flex items-center gap-2">
                      <Select aria-label={`Cara hitung termin ${i + 1}`} className="w-32" value={r.mode} onChange={(e) => patch(i, { mode: e.target.value as Row["mode"] })}>
                        <option value="percent">Persen</option>
                        <option value="amount">Jumlah tetap</option>
                      </Select>
                      {r.mode === "percent" ? (
                        <span className="flex items-center gap-1.5" data-field={key("percent")}>
                          <Input
                            aria-label={`Persen termin ${i + 1}`}
                            inputMode="numeric"
                            className="w-20 text-right tabular-nums"
                            value={String(r.percent ?? "")}
                            onChange={(e) => patch(i, { percent: e.target.value.replace(/[^0-9]/g, "") as unknown as number })}
                          />
                          <span className="text-[13px] text-muted-foreground">%</span>
                        </span>
                      ) : (
                        <span className="flex-1" data-field={key("amount")}>
                          <CurrencyInput aria-invalid={Boolean(errors[key("amount")])} value={Number(r.amount) || 0} onChange={(n) => patch(i, { amount: n })} />
                        </span>
                      )}
                    </div>
                    {(errors[key("percent")] || errors[key("amount")]) && (
                      <span role="alert" className="text-xs font-medium text-danger-text">
                        {errors[key("percent")] ?? errors[key("amount")]}
                      </span>
                    )}
                    {amountNow !== null && <span className="text-xs text-muted-foreground">Sekarang {formatCurrency(amountNow)}</span>}
                  </fieldset>
                  <Field label="Jatuh tempo" name={key("dueBasis")}>
                    <Select aria-label={`Dasar jatuh tempo termin ${i + 1}`} value={r.dueBasis} onChange={(e) => patch(i, { dueBasis: e.target.value as Row["dueBasis"] })}>
                      {TERM_DUE_BASIS_OPTIONS.map((b) => (
                        <option key={b} value={b}>
                          {TERM_DUE_BASIS_LABELS[b]}
                        </option>
                      ))}
                    </Select>
                  </Field>
                  <fieldset className="flex flex-col gap-1.5" data-field={key("days")}>
                    <legend className="mb-1.5 text-[13px] font-medium text-foreground">Berapa hari</legend>
                    <div className="flex flex-wrap items-center gap-2">
                      <Input
                        aria-label={`Jumlah hari termin ${i + 1}`}
                        inputMode="numeric"
                        className="w-20 text-right tabular-nums"
                        value={String(r.days ?? "")}
                        onChange={(e) => patch(i, { days: e.target.value.replace(/[^0-9]/g, "") as unknown as number })}
                      />
                      {r.dueBasis === "Event" ? (
                        <Select aria-label={`Sebelum atau sesudah acara, termin ${i + 1}`} className="w-32" value={r.before ? "before" : "after"} onChange={(e) => patch(i, { before: e.target.value === "before" })}>
                          <option value="before">hari sebelum</option>
                          <option value="after">hari sesudah</option>
                        </Select>
                      ) : (
                        <span className="text-[13px] text-muted-foreground">hari setelah diterima</span>
                      )}
                    </div>
                    {errors[key("days")] && (
                      <span role="alert" className="text-xs font-medium text-danger-text">
                        {errors[key("days")]}
                      </span>
                    )}
                  </fieldset>
                  {r.dueBasis === "Event" && (
                    <div className="sm:col-span-2">
                      <Field label="Acara" name={key("dueEventId")} error={errors[key("dueEventId")]} hint="Kosongkan untuk memakai acara utama (tanggal paling awal).">
                        <Select aria-label={`Acara termin ${i + 1}`} value={r.dueEventId || "utama"} onChange={(e) => patch(i, { dueEventId: e.target.value === "utama" ? "" : e.target.value })}>
                          <option value="utama">Acara utama</option>
                          {detail.events.map((e) => (
                            <option key={e.id} value={e.id}>
                              {e.kindName}
                            </option>
                          ))}
                        </Select>
                      </Field>
                    </div>
                  )}
                </div>
              </li>
            );
          })}
        </ol>
        <div className="flex flex-wrap items-center justify-between gap-3">
          <Button type="button" variant="secondary" icon={<Plus className="h-4 w-4" />} onClick={() => form.set("terms", [...rows, emptyRow()])}>
            Tambah termin
          </Button>
          <p role="status" className="text-[13px] text-text-secondary tabular-nums">
            {comesTo === null
              ? `Total persen ${percentSum}%`
              : `Jumlah termin ${formatCurrency(comesTo)} dari nilai kontrak ${formatCurrency(total ?? 0)}`}
          </p>
        </div>
      </form>
    </Drawer>
  );
}
