import { useMemo, useRef } from "react";
import { Modal } from "@/shared/components/ui/Modal";
import { Button } from "@/shared/components/ui/Button";
import { Input, Textarea, Select, Field } from "@/shared/components/ui/Input";
import { CurrencyInput } from "@/shared/components/ui/CurrencyInput";
import { useZodForm } from "@/shared/forms/useZodForm";
import { FormErrorSummary } from "@/shared/forms/FormErrorSummary";
import { formatCurrency } from "@/shared/lib/formatters";
import {
  supplierPaymentUpdateSchema,
  PAYMENT_METHOD_OPTIONS,
  PAYMENT_TYPE_OPTIONS,
  type SupplierPaymentUpdateFormValues,
} from "@/modules/projects/schemas/payment.schema";
import { overpaymentMessage } from "@/modules/projects/lib/payments";
import type { SupplierPayment } from "@/modules/projects/types";

const LABELS: Record<string, string> = {
  type: "Jenis pembayaran", amount: "Nominal", paymentDate: "Tanggal", method: "Metode", referenceNumber: "No. referensi", notes: "Catatan",
};

interface Props {
  payment: SupplierPayment;
  supplierName: string;
  /** "Biaya rekanan" of a vendor or venue the WO pays; 0 when not set or the client pays (nothing is limited). */
  agreedCost: number;
  /** The supplier's other payments (this one left out). */
  others: SupplierPayment[];
  onClose: () => void;
  onSubmit: (values: SupplierPaymentUpdateFormValues) => Promise<void>;
}

/**
 * "Ubah pembayaran" for a payment to a vendor or venue: the ledger fields only. The payment stays with its vendor or
 * venue (delete it and record it again to move it) and its files are managed in Dokumen.
 */
export function EditSupplierPaymentModal({ payment, supplierName, agreedCost, others, onClose, onSubmit }: Props) {
  const formRef = useRef<HTMLFormElement>(null);
  const schema = useMemo(
    () =>
      supplierPaymentUpdateSchema.superRefine((v, ctx) => {
        const message = overpaymentMessage(agreedCost, others, { type: v.type, amount: v.amount }, formatCurrency, "sisa biaya rekanan", payment);
        if (message) ctx.addIssue({ code: "custom", path: ["amount"], message });
      }) as unknown as typeof supplierPaymentUpdateSchema,
    [agreedCost, others, payment],
  );
  const form = useZodForm({
    schema,
    initial: {
      type: payment.type,
      amount: payment.amount,
      paymentDate: payment.paymentDate,
      method: (PAYMENT_METHOD_OPTIONS as readonly string[]).includes(payment.method)
        ? (payment.method as SupplierPaymentUpdateFormValues["method"])
        : "Transfer Bank",
      referenceNumber: payment.referenceNumber,
      notes: payment.notes,
    },
  });
  const { values, errors, blur } = form;

  const submit = form.handleSubmit(async (parsed) => onSubmit(parsed), {
    root: () => formRef.current,
    fallback: "Perubahan belum tersimpan. Periksa koneksi lalu coba lagi.",
    known: (key) => key in values,
  });

  return (
    <Modal
      open
      onClose={onClose}
      title="Ubah pembayaran"
      description={`Pembayaran ke ${supplierName}. Bukti yang sudah terlampir tidak berubah.`}
      busy={form.submitting}
      footer={
        <>
          <Button variant="secondary" onClick={onClose} disabled={form.submitting}>Batal</Button>
          <Button type="submit" form="edit-supplier-payment-form" disabled={form.submitting}>
            {form.submitting ? "Menyimpan..." : "Simpan perubahan"}
          </Button>
        </>
      }
    >
      <form id="edit-supplier-payment-form" ref={formRef} noValidate onSubmit={(e) => void submit(e)} className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <div className="sm:col-span-2 empty:hidden">
          <FormErrorSummary errors={form.errorList} formError={form.formError} labels={LABELS} root={() => formRef.current} />
        </div>
        <Field label="Jenis pembayaran" required name="type" error={errors.type}>
          <Select value={values.type} onChange={(e) => form.set("type", e.target.value as SupplierPaymentUpdateFormValues["type"])}>
            {PAYMENT_TYPE_OPTIONS.map((t) => (
              <option key={t} value={t}>{t}</option>
            ))}
          </Select>
        </Field>
        <Field label="Nominal (Rp)" required name="amount" error={errors.amount}>
          <CurrencyInput value={Number(values.amount) || 0} onChange={(n) => form.set("amount", n)} aria-invalid={errors.amount ? true : undefined} />
        </Field>
        <Field label="Tanggal" required name="paymentDate" error={errors.paymentDate}>
          <Input type="date" value={values.paymentDate} onChange={(e) => form.set("paymentDate", e.target.value)} onBlur={() => blur("paymentDate")} />
        </Field>
        <Field label="Metode" required name="method" error={errors.method}>
          <Select value={values.method} onChange={(e) => form.set("method", e.target.value as SupplierPaymentUpdateFormValues["method"])}>
            {PAYMENT_METHOD_OPTIONS.map((m) => (
              <option key={m} value={m}>{m}</option>
            ))}
          </Select>
        </Field>
        <div className="sm:col-span-2">
          <Field label="No. referensi" name="referenceNumber" error={errors.referenceNumber}>
            <Input value={values.referenceNumber} onChange={(e) => form.set("referenceNumber", e.target.value)} />
          </Field>
        </div>
        <div className="sm:col-span-2">
          <Field label="Catatan" name="notes" error={errors.notes}>
            <Textarea rows={2} value={values.notes} onChange={(e) => form.set("notes", e.target.value)} />
          </Field>
        </div>
      </form>
    </Modal>
  );
}
