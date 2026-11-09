import { z } from "zod";
import { usernameSchema } from "@/shared/lib/validators";

export const PLATFORM_ADMIN_ROLE_OPTIONS = ["Super Admin", "Billing", "Support", "Read-only"] as const;

/** What each role may do — shown next to the role picker. */
export const PLATFORM_ADMIN_ROLE_DESCRIPTIONS: Record<(typeof PLATFORM_ADMIN_ROLE_OPTIONS)[number], string> = {
  "Super Admin": "Semua akses, termasuk admin, gateway pembayaran, dan pembekuan tenant.",
  Billing: "Kelola paket dan aktivasi langganan manual.",
  Support: "Daftarkan dan ubah tenant, serta reset kredensial Owner.",
  "Read-only": "Hanya melihat data, tanpa mengubah apa pun.",
};

export const platformAdminSchema = z.object({
  name: z.string().min(2, "Isi nama lengkap, minimal 2 karakter"),
  title: z.string().min(2, "Isi jabatan, minimal 2 karakter"),
  role: z.enum(PLATFORM_ADMIN_ROLE_OPTIONS),
  email: z.string().trim().email("Isi email lengkap, misalnya nama@elproof.id"),
  phone: z.string().min(6, "Isi nomor telepon, minimal 6 digit"),
});

export type PlatformAdminFormValues = z.infer<typeof platformAdminSchema>;

export const platformAdminCreateSchema = platformAdminSchema
  .extend({
    username: usernameSchema,
    password: z.string().min(8, "Password minimal 8 karakter"),
    confirmPassword: z.string().min(1, "Ulangi password"),
  })
  .refine((values) => values.password === values.confirmPassword, {
    message: "Kedua password belum sama",
    path: ["confirmPassword"],
  });

export type PlatformAdminCreateFormValues = z.infer<typeof platformAdminCreateSchema>;

export const resetPlatformAdminPasswordSchema = z
  .object({
    password: z.string().min(8, "Password minimal 8 karakter"),
    confirmPassword: z.string().min(1, "Ulangi password"),
  })
  .refine((values) => values.password === values.confirmPassword, {
    message: "Kedua password belum sama",
    path: ["confirmPassword"],
  });

export type ResetPlatformAdminPasswordFormValues = z.infer<typeof resetPlatformAdminPasswordSchema>;
