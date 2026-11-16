import { z } from "zod";
import { todayISO } from "@/shared/lib/formatters";

/** The same methods the WO records (payment.schema.ts), transfer first: it is what a couple almost always does. */
export const SUBMIT_METHOD_OPTIONS = ["Transfer Bank", "QRIS", "Tunai"] as const;

const ACCEPTED = ["image/jpeg", "image/png", "image/webp", "application/pdf"];

/** "Kirim bukti transfer" from the Portal: the WO verifies it before it counts toward the contract. */
export const submitProofSchema = z.object({
  amount: z.coerce.number().int("Nominal dalam rupiah penuh").positive("Isi nominal yang Anda transfer"),
  paymentDate: z
    .string()
    .min(1, "Isi tanggal transfer")
    .refine((d) => d <= todayISO(), "Tanggal transfer tidak boleh setelah hari ini"),
  method: z.enum(SUBMIT_METHOD_OPTIONS),
  referenceNumber: z.string().max(100, "Paling banyak 100 karakter").optional().default(""),
  notes: z.string().max(500, "Paling banyak 500 karakter").optional().default(""),
  // Starts empty (undefined) in the form, but a submission always needs one.
  proofFile: z
    .custom<File | undefined>((f) => f instanceof File, "Lampirkan foto atau PDF bukti transfer")
    .refine((f) => !f || ACCEPTED.includes(f.type), "Pilih foto (JPG, PNG) atau PDF"),
});

export type SubmitProofValues = z.infer<typeof submitProofSchema>;
