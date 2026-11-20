import { z } from "zod";
import { isValidCity } from "@/shared/constants/cities";

// Capacities and prices follow the same "0 in the form == not set" convention
// as ProjectFormModal's contractValue — converted to/from null at the store
// boundary (0 <-> null), not modeled as optional here. facilities/socialMedia
// are plain free text (Revisi) — no array/chip-list state.
const MAX_CAPACITY = 4294967295;

/** Both capacity fields carry it when neither is filled (the server says the same). */
export const CAPACITY_REQUIRED_MESSAGE = "Isi kapasitas berdiri atau duduk, minimal satu";

const venueFields = z.object({
  name: z.string().trim().min(2, "Isi nama venue, minimal 2 huruf").max(150, "Nama venue maksimal 150 karakter"),
  picName: z.string().min(2, "Isi nama PIC"),
  phonePic: z.string().min(6, "Isi nomor telepon PIC, minimal 6 angka"),
  phoneVenue: z.string().optional().default(""),
  email: z.union([z.literal(""), z.string().email("Periksa email, contoh: nama@email.com")]).default(""),
  address: z.string().optional().default(""),
  city: z.string().optional().default(""),
  capacityStanding: z.coerce.number().min(0, "Kapasitas berdiri tidak boleh kurang dari 0").max(MAX_CAPACITY, "Kapasitas terlalu besar. Periksa lagi angkanya."),
  capacitySeated: z.coerce.number().min(0, "Kapasitas duduk tidak boleh kurang dari 0").max(MAX_CAPACITY, "Kapasitas terlalu besar. Periksa lagi angkanya."),
  priceAkad: z.coerce.number().min(0, "Harga Akad tidak boleh kurang dari 0"),
  priceResepsi: z.coerce.number().min(0, "Harga Resepsi tidak boleh kurang dari 0"),
  priceAkadResepsi: z.coerce.number().min(0, "Harga Akad + Resepsi tidak boleh kurang dari 0"),
  charge: z.coerce.number().min(0, "Charge tambahan tidak boleh kurang dari 0"),
  facilities: z.string().optional().default(""),
  socialMedia: z.string().optional().default(""),
  notes: z.string().optional().default(""),
});

// At least one capacity, at least 1 (PLAN paket-wo §3.3): the message sits on both fields.
function requireCapacity(v: { capacityStanding: number; capacitySeated: number }, ctx: z.RefinementCtx) {
  if (v.capacityStanding >= 1 || v.capacitySeated >= 1) return;
  for (const path of ["capacityStanding", "capacitySeated"]) {
    ctx.addIssue({ code: z.ZodIssueCode.custom, path: [path], message: CAPACITY_REQUIRED_MESSAGE });
  }
}

export const venueSchema = venueFields.superRefine(requireCapacity);

export type VenueFormValues = z.infer<typeof venueSchema>;

// Kota is mandatory only when creating a new venue (see ADR-0016's Revisi) — a
// migrated/incomplete venue can still be edited with it unset. City is
// validated against the shared fixed CITIES list.
export const venueCreateSchema = venueFields
  .extend({ city: z.string().refine(isValidCity, "Pilih kota dari daftar yang tersedia") })
  .superRefine(requireCapacity);

export type VenueCreateFormValues = z.infer<typeof venueCreateSchema>;
