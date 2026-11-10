import { z } from "zod";

export const planSchema = z.object({
  // Every plan belongs to exactly one App (ADR-0040); ElProof's own App sells to WO tenants.
  appId: z.string().min(1, "Pilih aplikasi pemilik paket"),
  name: z.string().trim().min(2, "Isi nama paket, minimal 2 karakter"),
  durationMonths: z.coerce
    .number({ invalid_type_error: "Isi durasi dengan angka" })
    .int("Durasi harus bilangan bulat")
    .min(1, "Durasi minimal 1 bulan"),
  price: z.coerce
    .number({ invalid_type_error: "Isi harga dengan angka" })
    .int("Harga harus bilangan bulat, tanpa desimal")
    .min(0, "Harga tidak boleh negatif"),
  // Blank rows are dropped before the "at least one feature" check, so an untouched empty row is not a feature.
  features: z
    .array(z.string())
    .transform((rows) => rows.map((f) => f.trim()).filter((f) => f.length > 0))
    .pipe(z.array(z.string()).min(1, "Isi minimal 1 fitur paket")),
});

export type PlanFormValues = z.infer<typeof planSchema>;
