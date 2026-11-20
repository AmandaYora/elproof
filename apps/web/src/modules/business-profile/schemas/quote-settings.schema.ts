import { z } from "zod";

// Mirrors the API's own rules (platform SaveQuoteSettings) with the same sentences.
export const quoteSettingsSchema = z.object({
  documentCode: z.string().regex(/^[A-Z0-9]{2,5}$/, "Kode usaha 2–5 huruf atau angka, contoh MO"),
  isPkp: z.boolean(),
  defaultValidityDays: z
    .string()
    .regex(/^\d+$/, "Masa berlaku 1–365 hari")
    .refine((v) => Number(v) >= 1 && Number(v) <= 365, "Masa berlaku 1–365 hari"),
  defaultTermsText: z.string().max(5000, "Syarat & ketentuan maksimal 5.000 karakter"),
});

export type QuoteSettingsFormValues = z.infer<typeof quoteSettingsSchema>;
