import { z } from "zod";

export const planSchema = z.object({
  name: z.string().min(2, "Nama paket wajib diisi"),
  durationMonths: z.coerce.number().int().min(1, "Durasi minimal 1 bulan"),
  price: z.coerce.number().int().min(0, "Harga tidak valid"),
  features: z.array(z.string().min(1, "Fitur tidak boleh kosong")).min(1, "Minimal 1 fitur wajib diisi"),
});

export type PlanFormValues = z.infer<typeof planSchema>;
