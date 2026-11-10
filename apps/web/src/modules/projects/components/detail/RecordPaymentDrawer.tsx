import { useEffect, useMemo, useRef } from "react";
import { z } from "zod";
import { Drawer } from "@/shared/components/ui/Drawer";
import { Button } from "@/shared/components/ui/Button";
import { Input, Textarea, Select, Field } from "@/shared/components/ui/Input";
import { CurrencyInput } from "@/shared/components/ui/CurrencyInput";
import { useZodForm } from "@/shared/forms/useZodForm";
import { FormErrorSummary } from "@/shared/forms/FormErrorSummary";
import { newIdempotencyKey } from "@/shared/lib/direct-upload";
import { formatCurrency } from "@/shared/lib/formatters";
import { paymentSchema, PAYMENT_METHOD_OPTIONS, PAYMENT_TYPE_OPTIONS, type PaymentFormValues } from "@/modules/projects/schemas/payment.schema";
import { countsAgainstContract, overpaymentMessage, remainingToPay, type PaymentLike } from "@/modules/projects/lib/payments";
import { todayISO } from "@/modules/projects/lib/dates";
import type { SaveOptions } from "@/modules/projects/stores/useProjectStore";

/** What a payment is measured against: the contract (client), one engagement (vendor) or the venue. */
export interface OwedContext {
  /** 0 = not set yet: nothing is limited. */
  contractValue: number;
  existing: PaymentLike[];
  /** "sisa tagihan", "sisa pembayaran vendor", "sisa biaya venue": how the remaining amount is named. */
  owedLabel: string;
}

interface Props {
  open: boolean;
  onClose: () => void;
  title: string;
  description: string;
  /** Vendor payments pick an engagement; client and venue payments have none. */
  vendorOptions?: Array<{ value: string; label: string }>;
  /** Which file slots exist: a client payment has only a transfer proof. */
  slots: { invoice: boolean };
  owed: (projectVendorId: string) => OwedContext;
  /** Saves payment + proof in one call and closes the drawer. Throw to keep it open. */
  onSubmit: (values: PaymentFormValues, options: SaveOptions) => Promise<void>;
}

const LABELS: Record<string, string> = {
  projectVendorId: "Vendor", type: "Jenis pembayaran", amount: "Nominal", paymentDate: "Tanggal", method: "Metode",
  referenceNumber: "No. referensi", invoiceFile: "Invoice", proofFile: "Bukti transfer", notes: "Catatan",
};

function initialValues(vendorId: string): PaymentFormValues {
  return { projectVendorId: vendorId, type: "DP", amount: 0, paymentDate: todayISO(), method: "Transfer Bank", referenceNumber: "", notes: "", invoiceFile: undefined, proofFile: undefined };
}

const FILE_CLASS =
  "block w-full text-[13px] text-muted-foreground file:mr-3 file:rounded-md file:border-0 file:bg-primary file:px-3 file:py-1.5 file:text-[12.5px] file:font-semibold file:text-primary-foreground";

/**
 * "Catat pembayaran" as a side panel (PLAN.md Tahap 4): it shows what is still
 * owed, fills the remaining amount when the type is Pelunasan, and saves the
 * payment together with its proof in one request that can be retried safely
 * (Idempotency-Key). If saving fails nothing was saved and the panel stays open
 * with everything typed.
 */
export function RecordPaymentDrawer({ open, onClose, title, description, vendorOptions, slots, owed, onSubmit }: Props) {
  const isVendor = vendorOptions !== undefined;
  const owedRef = useRef(owed);
  owedRef.current = owed;
  const keyRef = useRef(newIdempotencyKey());
  const formRef = useRef<HTMLFormElement>(null);

  // A save refused for being over the contract is a field message on Nominal, same wording as the server's.
  const schema = useMemo(() => {
    // Client and venue payments have no engagement: the field exists in the shape but may be empty.
    const base = isVendor ? paymentSchema : paymentSchema.extend({ projectVendorId: z.string().default("") });
    return base.superRefine((v, ctx) => {
      const o = owedRef.current(v.projectVendorId ?? "");
      const message = overpaymentMessage(o.contractValue, o.existing, { type: v.type, amount: v.amount }, formatCurrency, o.owedLabel);
      if (message) ctx.addIssue({ code: "custom", path: ["amount"], message });
    }) as unknown as typeof paymentSchema;
  }, [isVendor]);

  const form = useZodForm({ schema, initial: initialValues(vendorOptions?.[0]?.value ?? "") });
  const { values, errors, blur } = form;

  useEffect(() => {
    if (open) {
      form.reset(initialValues(vendorOptions?.[0]?.value ?? ""));
      keyRef.current = newIdempotencyKey();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open]);

  // Any change makes a different request, so it must not reuse the earlier key.
  function set<K extends keyof PaymentFormValues & string>(key: K, value: PaymentFormValues[K]) {
    keyRef.current = newIdempotencyKey();
    form.set(key, value);
  }

  const o = owed(values.projectVendorId);
  const remaining = Math.max(remainingToPay(o.contractValue, o.existing), 0);

  // Pelunasan means "what is left", so one tap fills it.
  function changeType(type: PaymentFormValues["type"]) {
    set("type", type);
    if (type === "Pelunasan" && values.amount === 0 && remaining > 0) set("amount", remaining);
  }

  const submit = form.handleSubmit(
    async (parsed) => {
      await onSubmit({ ...parsed, projectVendorId: parsed.projectVendorId ?? "" }, { idempotencyKey: keyRef.current });
      keyRef.current = newIdempotencyKey();
    },
    { root: () => formRef.current, fallback: "Pembayaran belum tersimpan. Periksa koneksi lalu coba lagi." },
  );

  return (
    <Drawer
      open={open}
      onClose={onClose}
      title={title}
      description={description}
      busy={form.submitting}
      footer={
        <>
          <Button variant="secondary" onClick={onClose} disabled={form.submitting}>Batal</Button>
          <Button type="submit" form="record-payment-form" disabled={form.submitting}>
            {form.submitting ? "Menyimpan..." : "Simpan pembayaran"}
          </Button>
        </>
      }
    >
      <form id="record-payment-form" ref={formRef} noValidate onSubmit={(e) => void submit(e)} className="flex flex-col gap-4">
        <FormErrorSummary errors={form.errorList} formError={form.formError} labels={LABELS} root={() => formRef.current} />

        {o.contractValue > 0 && (
          <p className="rounded-md bg-muted px-3.5 py-2.5 text-[13px] text-foreground">
            {o.owedLabel[0].toUpperCase() + o.owedLabel.slice(1)}: <strong className="tabular-nums">{formatCurrency(remaining)}</strong>
          </p>
        )}

        {isVendor && (
          <Field label="Vendor" required name="projectVendorId" error={errors.projectVendorId}>
            <Select value={values.projectVendorId} onChange={(e) => set("projectVendorId", e.target.value)}>
              {vendorOptions!.map((v) => (
                <option key={v.value} value={v.value}>{v.label}</option>
              ))}
            </Select>
          </Field>
        )}
        <Field label="Jenis pembayaran" required name="type" error={errors.type}>
          <Select value={values.type} onChange={(e) => changeType(e.target.value as PaymentFormValues["type"])}>
            {PAYMENT_TYPE_OPTIONS.map((t) => (
              <option key={t} value={t}>{t}</option>
            ))}
          </Select>
        </Field>
        <Field
          label="Nominal (Rp)"
          required
          name="amount"
          error={errors.amount}
          hint={countsAgainstContract(values.type) && o.contractValue > 0 ? `Maksimal ${formatCurrency(remaining)} untuk jenis ini` : undefined}
        >
          <CurrencyInput value={values.amount} onChange={(n) => set("amount", n)} aria-invalid={errors.amount ? true : undefined} />
        </Field>
        <Field label="Tanggal" required name="paymentDate" error={errors.paymentDate}>
          <Input type="date" value={values.paymentDate} onChange={(e) => set("paymentDate", e.target.value)} onBlur={() => blur("paymentDate")} />
        </Field>
        <Field label="Metode" required name="method" error={errors.method}>
          <Select value={values.method} onChange={(e) => set("method", e.target.value as PaymentFormValues["method"])}>
            {PAYMENT_METHOD_OPTIONS.map((m) => (
              <option key={m} value={m}>{m}</option>
            ))}
          </Select>
        </Field>
        <Field label="No. referensi" name="referenceNumber" error={errors.referenceNumber}>
          <Input value={values.referenceNumber} onChange={(e) => set("referenceNumber", e.target.value)} />
        </Field>
        {slots.invoice && (
          <Field label="Berkas tagihan" name="invoiceFile" error={errors.invoiceFile} hint="Foto atau PDF, opsional. Bisa dilampirkan nanti.">
            <input type="file" accept="image/jpeg,image/png,application/pdf" onChange={(e) => set("invoiceFile", e.target.files?.[0] ?? undefined)} className={FILE_CLASS} />
          </Field>
        )}
        <Field label="Bukti transfer" name="proofFile" error={errors.proofFile} hint="Foto atau PDF, opsional. Pembayaran dan buktinya disimpan bersamaan.">
          <input type="file" accept="image/jpeg,image/png,application/pdf" onChange={(e) => set("proofFile", e.target.files?.[0] ?? undefined)} className={FILE_CLASS} />
        </Field>
        <Field label="Catatan" name="notes" error={errors.notes}>
          <Textarea rows={2} value={values.notes} onChange={(e) => set("notes", e.target.value)} />
        </Field>
      </form>
    </Drawer>
  );
}
