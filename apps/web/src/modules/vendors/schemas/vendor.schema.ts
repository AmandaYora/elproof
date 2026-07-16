import { z } from "zod";

export const vendorSchema = z.object({
  name: z.string().min(2, "Nama vendor wajib diisi"),
  categoryId: z.string().min(1, "Kategori wajib dipilih"),
  picName: z.string().min(2, "Nama PIC wajib diisi"),
  phone: z.string().min(6, "Nomor telepon tidak valid"),
  email: z.string().email("Email tidak valid"),
  address: z.string().min(1, "Alamat wajib diisi"),
  notes: z.string().optional().default(""),
});

export type VendorFormValues = z.infer<typeof vendorSchema>;
