import { z } from "zod";
import { usernameSchema } from "@/shared/lib/validators";

export const PLATFORM_ADMIN_ROLE_OPTIONS = ["Super Admin", "Support"] as const;

export const platformAdminSchema = z.object({
  name: z.string().min(2, "Nama wajib diisi"),
  title: z.string().min(2, "Jabatan wajib diisi"),
  role: z.enum(PLATFORM_ADMIN_ROLE_OPTIONS),
  email: z.string().email("Email tidak valid"),
  phone: z.string().min(6, "Nomor telepon tidak valid"),
});

export type PlatformAdminFormValues = z.infer<typeof platformAdminSchema>;

export const platformAdminCreateSchema = platformAdminSchema
  .extend({
    username: usernameSchema,
    password: z.string().min(8, "Password minimal 8 karakter"),
    confirmPassword: z.string().min(1, "Konfirmasi password wajib diisi"),
  })
  .refine((values) => values.password === values.confirmPassword, {
    message: "Konfirmasi password tidak sama",
    path: ["confirmPassword"],
  });

export type PlatformAdminCreateFormValues = z.infer<typeof platformAdminCreateSchema>;

export const resetPlatformAdminPasswordSchema = z
  .object({
    password: z.string().min(8, "Password minimal 8 karakter"),
    confirmPassword: z.string().min(1, "Konfirmasi password wajib diisi"),
  })
  .refine((values) => values.password === values.confirmPassword, {
    message: "Konfirmasi password tidak sama",
    path: ["confirmPassword"],
  });

export type ResetPlatformAdminPasswordFormValues = z.infer<typeof resetPlatformAdminPasswordSchema>;
