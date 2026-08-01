import { z } from "zod";

export const CLIENT_PAYMENT_TYPE_OPTIONS = ["DP", "Termin", "Pelunasan", "Tambahan", "Refund"] as const;

// Simpler than payment.schema.ts's paymentSchema — no projectVendorId (a
// client payment isn't tied to any vendor). proofFile is validated only
// client-side and never sent as part of the JSON body itself — the store
// action uploads it as a separate evidence call after the payment is
// created (see useProjectStore.createClientPayment).
export const clientPaymentSchema = z.object({
  type: z.enum(CLIENT_PAYMENT_TYPE_OPTIONS),
  amount: z.coerce.number().positive("Nominal harus lebih dari 0"),
  paymentDate: z.string().min(1, "Tanggal wajib diisi"),
  method: z.string().min(2, "Metode pembayaran wajib diisi"),
  referenceNumber: z.string().min(1, "No. referensi wajib diisi"),
  notes: z.string().optional().default(""),
  proofFile: z.instanceof(File).optional(),
});

export type ClientPaymentFormValues = z.infer<typeof clientPaymentSchema>;
