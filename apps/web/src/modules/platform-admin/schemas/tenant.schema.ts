import { z } from "zod";
import { isBrandColorPresetKey, type BrandColorPresetKey } from "@/theme/brandPresets";

const tenantBaseSchema = z.object({
  businessName: z.string().min(2, "Isi nama WO, minimal 2 karakter"),
  ownerName: z.string().min(2, "Isi nama owner, minimal 2 karakter"),
  email: z.string().trim().email("Isi email akun Google Owner, misalnya owner@gmail.com"),
  phone: z.string().min(6, "Isi nomor HP Owner, minimal 6 digit"),
  city: z.string().min(2, "Isi kota, minimal 2 karakter"),
});

// brandColorPreset is edit-only (a new tenant starts on the "navy" default),
// so it is on tenantSchema, not the create flow.
export const tenantSchema = tenantBaseSchema.extend({
  brandColorPreset: z.string().refine(
    (v): v is BrandColorPresetKey => isBrandColorPresetKey(v),
    "Pilih salah satu preset warna yang tersedia"
  ),
});

export type TenantFormValues = z.infer<typeof tenantSchema>;

// Sales-assisted registration: no username or password — the Owner gets an
// invitation link and signs in with the Google account of this email
// (PLAN.md Fase 3, decision K5).
export const tenantCreateSchema = tenantBaseSchema;

export type TenantCreateFormValues = z.infer<typeof tenantCreateSchema>;
