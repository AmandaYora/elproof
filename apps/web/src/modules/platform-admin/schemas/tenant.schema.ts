import { z } from "zod";

export const tenantSchema = z.object({
  businessName: z.string().min(2, "Isi nama WO, minimal 2 karakter"),
  ownerName: z.string().min(2, "Isi nama owner, minimal 2 karakter"),
  email: z.string().trim().email("Isi email akun Google Owner, misalnya owner@gmail.com"),
  phone: z.string().min(6, "Isi nomor HP Owner, minimal 6 digit"),
  city: z.string().min(2, "Isi kota, minimal 2 karakter"),
});

export type TenantFormValues = z.infer<typeof tenantSchema>;

// Sales-assisted registration: no username or password — the Owner gets an
// invitation link and signs in with the Google account of this email
// (PLAN.md Fase 3, decision K5).
export const tenantCreateSchema = tenantSchema;

export type TenantCreateFormValues = z.infer<typeof tenantCreateSchema>;
