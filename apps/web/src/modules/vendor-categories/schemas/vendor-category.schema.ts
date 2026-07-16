import { z } from "zod";

export const vendorCategorySchema = z.object({
  name: z.string().min(2, "Nama kategori wajib diisi"),
  description: z.string().min(1, "Deskripsi wajib diisi"),
});

export type VendorCategoryFormValues = z.infer<typeof vendorCategorySchema>;
