import { z } from "zod";

export const PAYMENT_TYPE_OPTIONS = ["DP", "Termin", "Pelunasan", "Tambahan", "Refund"] as const;

// Shared with client-payment.schema.ts (imported from there) so the two
// forms can never drift apart on which methods are offered.
export const PAYMENT_METHOD_OPTIONS = ["Tunai", "Transfer Bank", "QRIS"] as const;

// A payment to a vendor or venue of the project. Who pays it (the WO or the
// client directly) is the supplier's "Dibayar oleh": the server takes it from
// there, never from the form. invoiceFile/proofFile are validated here only and
// go up as upload tokens with the payment (useProjectStore.createSupplierPayment).
export const supplierPaymentSchema = z.object({
  projectSupplierId: z.string().min(1, "Pilih vendor atau venue"),
  type: z.enum(PAYMENT_TYPE_OPTIONS),
  amount: z.coerce.number().positive("Nominal harus lebih dari 0"),
  paymentDate: z.string().min(1, "Tanggal wajib diisi"),
  method: z.enum(PAYMENT_METHOD_OPTIONS),
  referenceNumber: z.string().optional().default(""),
  notes: z.string().optional().default(""),
  invoiceFile: z.instanceof(File).optional(),
  proofFile: z.instanceof(File).optional(),
});

export type SupplierPaymentFormValues = z.infer<typeof supplierPaymentSchema>;

// Edit scope is the ledger fields only: a payment does not move to another
// supplier (delete and record it again) and its files are managed in Dokumen.
export const supplierPaymentUpdateSchema = z.object({
  type: z.enum(PAYMENT_TYPE_OPTIONS),
  amount: z.coerce.number().positive("Nominal harus lebih dari 0"),
  paymentDate: z.string().min(1, "Tanggal wajib diisi"),
  method: z.enum(PAYMENT_METHOD_OPTIONS),
  referenceNumber: z.string().optional().default(""),
  notes: z.string().optional().default(""),
});

export type SupplierPaymentUpdateFormValues = z.infer<typeof supplierPaymentUpdateSchema>;
