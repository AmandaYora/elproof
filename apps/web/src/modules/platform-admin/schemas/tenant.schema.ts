import { z } from "zod";
import { usernameSchema } from "@/shared/lib/validators";
import { isBrandColorPresetKey, type BrandColorPresetKey } from "@/theme/brandPresets";

const tenantBaseSchema = z.object({
  businessName: z.string().min(2, "Nama WO wajib diisi"),
  ownerName: z.string().min(2, "Nama owner wajib diisi"),
  email: z.string().email("Email tidak valid"),
  phone: z.string().min(6, "Nomor telepon tidak valid"),
  city: z.string().min(2, "Kota wajib diisi"),
});

// brandColorPreset/logo are edit-only (a new tenant starts on the "navy"
// default — see PLAN.md §4) so they're on tenantSchema, not the create flow.
export const tenantSchema = tenantBaseSchema.extend({
  brandColorPreset: z.string().refine(
    (v): v is BrandColorPresetKey => isBrandColorPresetKey(v),
    "Pilih salah satu preset warna yang tersedia"
  ),
});

export type TenantFormValues = z.infer<typeof tenantSchema>;

export const tenantCreateSchema = tenantBaseSchema
  .extend({
    username: usernameSchema,
    password: z.string().min(8, "Password minimal 8 karakter"),
    confirmPassword: z.string().min(1, "Konfirmasi password wajib diisi"),
  })
  .refine((values) => values.password === values.confirmPassword, {
    message: "Konfirmasi password tidak sama",
    path: ["confirmPassword"],
  });

export type TenantCreateFormValues = z.infer<typeof tenantCreateSchema>;

export const resetTenantPasswordSchema = z
  .object({
    password: z.string().min(8, "Password minimal 8 karakter"),
    confirmPassword: z.string().min(1, "Konfirmasi password wajib diisi"),
  })
  .refine((values) => values.password === values.confirmPassword, {
    message: "Konfirmasi password tidak sama",
    path: ["confirmPassword"],
  });

export type ResetTenantPasswordFormValues = z.infer<typeof resetTenantPasswordSchema>;
