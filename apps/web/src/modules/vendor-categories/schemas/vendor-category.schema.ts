import { z } from "zod";

export const vendorCategorySchema = z.object({
  name: z.string().min(2, "Isi nama kategori, minimal 2 karakter"),
  description: z.string().min(1, "Isi deskripsi kategori"),
});

export type VendorCategoryFormValues = z.infer<typeof vendorCategorySchema>;
