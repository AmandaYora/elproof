import { z } from "zod";
import { isValidCity } from "@/shared/constants/cities";

// rentalPrice/charge/capacity follow the same "0 in the form == not set"
// convention as ProjectFormModal's contractValue — converted to/from null at
// the store boundary (0 <-> null), not modeled as optional here. facilities/
// socialMedia are plain free text now (Revisi) — no array/chip-list state.
export const venueSchema = z.object({
  name: z.string().min(2, "Nama venue wajib diisi"),
  picName: z.string().min(2, "Nama PIC wajib diisi"),
  phonePic: z.string().min(6, "Nomor telepon PIC tidak valid"),
  phoneVenue: z.string().optional().default(""),
  email: z.union([z.literal(""), z.string().email("Email tidak valid")]).default(""),
  address: z.string().optional().default(""),
  city: z.string().optional().default(""),
  rentalPrice: z.coerce.number().min(0, "Harga sewa tidak valid"),
  charge: z.coerce.number().min(0, "Charge tidak valid"),
  capacity: z.coerce.number().min(0, "Kapasitas tidak valid"),
  facilities: z.string().optional().default(""),
  socialMedia: z.string().optional().default(""),
  notes: z.string().optional().default(""),
});

export type VenueFormValues = z.infer<typeof venueSchema>;

// Harga Sewa and Kota are mandatory only when creating a new venue (see
// ADR-0016's Revisi) — a migrated/incomplete venue can still be edited with
// either left unset. City is validated against the shared fixed CITIES list.
export const venueCreateSchema = venueSchema.extend({
  rentalPrice: z.coerce.number().min(1, "Harga sewa wajib diisi"),
  city: z.string().refine(isValidCity, "Pilih kota dari daftar yang tersedia"),
});

export type VenueCreateFormValues = z.infer<typeof venueCreateSchema>;
