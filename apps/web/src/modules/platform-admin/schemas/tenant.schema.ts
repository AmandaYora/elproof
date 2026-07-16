import { z } from "zod";

export const tenantSchema = z.object({
  businessName: z.string().min(2, "Nama WO wajib diisi"),
  ownerName: z.string().min(2, "Nama owner wajib diisi"),
  email: z.string().email("Email tidak valid"),
  phone: z.string().min(6, "Nomor telepon tidak valid"),
  city: z.string().min(2, "Kota wajib diisi"),
});

export type TenantFormValues = z.infer<typeof tenantSchema>;

export const tenantCreateSchema = tenantSchema
  .extend({
    planId: z.string().min(1, "Pilih paket langganan"),
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
