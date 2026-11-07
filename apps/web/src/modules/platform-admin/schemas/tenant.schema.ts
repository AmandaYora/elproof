import { z } from "zod";
import { isBrandColorPresetKey, type BrandColorPresetKey } from "@/theme/brandPresets";

const tenantBaseSchema = z.object({
  businessName: z.string().min(2, "Nama WO wajib diisi"),
  ownerName: z.string().min(2, "Nama owner wajib diisi"),
  email: z.string().email("Email tidak valid"),
  phone: z.string().min(6, "Nomor telepon tidak valid"),
  city: z.string().min(2, "Kota wajib diisi"),
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
