import { z } from "zod";
import { isValidCity } from "@/shared/constants/cities";

// rentalPrice/charge/capacity follow the same "0 in the form == not set"
// convention as ProjectFormModal's contractValue — converted to/from null at
// the store boundary (0 <-> null), not modeled as optional here. facilities/
// socialMedia are plain free text now (Revisi) — no array/chip-list state.
export const venueSchema = z.object({
  name: z.string().min(2, "Isi nama venue"),
  picName: z.string().min(2, "Isi nama PIC"),
  phonePic: z.string().min(6, "Isi nomor telepon PIC, minimal 6 angka"),
  phoneVenue: z.string().optional().default(""),
  email: z.union([z.literal(""), z.string().email("Periksa email, contoh: nama@email.com")]).default(""),
  address: z.string().optional().default(""),
  city: z.string().optional().default(""),
  rentalPrice: z.coerce.number().min(0, "Harga sewa tidak boleh negatif"),
  charge: z.coerce.number().min(0, "Charge tidak boleh negatif"),
  capacity: z.coerce.number().min(0, "Kapasitas tidak boleh negatif"),
  facilities: z.string().optional().default(""),
  socialMedia: z.string().optional().default(""),
  notes: z.string().optional().default(""),
});

export type VenueFormValues = z.infer<typeof venueSchema>;

// Harga Sewa and Kota are mandatory only when creating a new venue (see
// ADR-0016's Revisi) — a migrated/incomplete venue can still be edited with
// either left unset. City is validated against the shared fixed CITIES list.
export const venueCreateSchema = venueSchema.extend({
  rentalPrice: z.coerce.number().min(1, "Isi harga sewa lebih dari 0"),
  city: z.string().refine(isValidCity, "Pilih kota dari daftar yang tersedia"),
});

export type VenueCreateFormValues = z.infer<typeof venueCreateSchema>;
