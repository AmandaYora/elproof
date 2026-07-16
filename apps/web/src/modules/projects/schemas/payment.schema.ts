import { z } from "zod";

export const PAYMENT_TYPE_OPTIONS = ["DP", "Termin", "Pelunasan", "Tambahan", "Refund"] as const;

export const paymentSchema = z.object({
  projectVendorId: z.string().min(1, "Vendor wajib dipilih"),
  type: z.enum(PAYMENT_TYPE_OPTIONS),
  amount: z.coerce.number().positive("Nominal harus lebih dari 0"),
  paymentDate: z.string().min(1, "Tanggal wajib diisi"),
  method: z.string().min(2, "Metode pembayaran wajib diisi"),
  referenceNumber: z.string().min(1, "No. referensi wajib diisi"),
  notes: z.string().optional().default(""),
});

export type PaymentFormValues = z.infer<typeof paymentSchema>;
