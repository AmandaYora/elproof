import { z } from "zod";

export const PAYMENT_TYPE_OPTIONS = ["DP", "Termin", "Pelunasan", "Tambahan", "Refund"] as const;

// Shared with client-payment.schema.ts (imported from there) so the two
// forms can never drift apart on which methods are offered.
export const PAYMENT_METHOD_OPTIONS = ["Tunai", "Transfer Bank", "QRIS"] as const;

// invoiceFile/proofFile are validated only client-side and never sent as
// part of the JSON body itself — the store action uploads each as a
// separate evidence call after the payment is created (see
// useProjectStore.createPayment), same convention as
// client-payment.schema.ts's own proofFile.
export const paymentSchema = z.object({
  projectVendorId: z.string().min(1, "Vendor wajib dipilih"),
  type: z.enum(PAYMENT_TYPE_OPTIONS),
  amount: z.coerce.number().positive("Nominal harus lebih dari 0"),
  paymentDate: z.string().min(1, "Tanggal wajib diisi"),
  method: z.enum(PAYMENT_METHOD_OPTIONS),
  referenceNumber: z.string().optional().default(""),
  notes: z.string().optional().default(""),
  invoiceFile: z.instanceof(File).optional(),
  proofFile: z.instanceof(File).optional(),
});

export type PaymentFormValues = z.infer<typeof paymentSchema>;

// Edit scope is the ledger fields only — no projectVendorId (re-parenting a
// payment to a different vendor engagement isn't supported; delete and
// recreate instead) and no file re-upload (already covered by the existing
// generic evidence tab) — see PLAN.md §2.5.
export const paymentUpdateSchema = z.object({
  type: z.enum(PAYMENT_TYPE_OPTIONS),
  amount: z.coerce.number().positive("Nominal harus lebih dari 0"),
  paymentDate: z.string().min(1, "Tanggal wajib diisi"),
  method: z.enum(PAYMENT_METHOD_OPTIONS),
  referenceNumber: z.string().optional().default(""),
  notes: z.string().optional().default(""),
});

export type PaymentUpdateFormValues = z.infer<typeof paymentUpdateSchema>;
