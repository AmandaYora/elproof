import { z } from "zod";
import { isValidCity } from "@/shared/constants/cities";

// priceAkad/priceAkadResepsi follow the same "0 in the form == not set"
// convention as VenueFormModal's rentalPrice — converted to/from null at the
// store boundary (0 <-> null), not modeled as optional here.
export const vendorSchema = z.object({
  name: z.string().min(2, "Nama vendor wajib diisi"),
  categoryId: z.string().min(1, "Kategori wajib dipilih"),
  picName: z.string().min(2, "Nama PIC wajib diisi"),
  phone: z.string().min(6, "Nomor telepon tidak valid"),
  email: z.union([z.literal(""), z.string().email("Email tidak valid")]).default(""),
  socialMedia: z.string().optional().default(""),
  city: z.string().optional().default(""),
  address: z.string().optional().default(""),
  priceAkad: z.coerce.number().min(0, "Harga akad tidak valid"),
  priceAkadResepsi: z.coerce.number().min(0, "Harga akad+resepsi tidak valid"),
  notes: z.string().optional().default(""),
});

export type VendorFormValues = z.infer<typeof vendorSchema>;

// Kota dan kedua field harga wajib hanya saat membuat vendor baru (mirrors
// venue.schema.ts's exact split) — vendor lama hasil migrasi/belum lengkap
// tetap bisa diedit dengan field ini kosong.
export const vendorCreateSchema = vendorSchema.extend({
  city: z.string().refine(isValidCity, "Pilih kota dari daftar yang tersedia"),
  priceAkad: z.coerce.number().min(1, "Harga akad wajib diisi"),
  priceAkadResepsi: z.coerce.number().min(1, "Harga akad+resepsi wajib diisi"),
});

export type VendorCreateFormValues = z.infer<typeof vendorCreateSchema>;
