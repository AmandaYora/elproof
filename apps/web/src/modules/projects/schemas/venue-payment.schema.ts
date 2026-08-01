import { z } from "zod";
import { PAYMENT_METHOD_OPTIONS } from "@/modules/projects/schemas/payment.schema";

export const VENUE_PAYMENT_TYPE_OPTIONS = ["DP", "Termin", "Pelunasan", "Tambahan", "Refund"] as const;

// Mirrors payment.schema.ts's paymentSchema (vendor's own two-slot evidence
// shape) minus projectVendorId -- a venue payment isn't tied to any vendor,
// same simplification client-payment.schema.ts already applies.
export const venuePaymentSchema = z.object({
  type: z.enum(VENUE_PAYMENT_TYPE_OPTIONS),
  amount: z.coerce.number().positive("Nominal harus lebih dari 0"),
  paymentDate: z.string().min(1, "Tanggal wajib diisi"),
  method: z.enum(PAYMENT_METHOD_OPTIONS),
  referenceNumber: z.string().optional().default(""),
  notes: z.string().optional().default(""),
  invoiceFile: z.instanceof(File).optional(),
  proofFile: z.instanceof(File).optional(),
});

export type VenuePaymentFormValues = z.infer<typeof venuePaymentSchema>;
