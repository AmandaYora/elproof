import { z } from "zod";

// Staff sign in with Google only (decision K5): the email is the Google
// account that must accept the invitation. No username or password here.
export const userSchema = z.object({
  name: z.string().trim().min(2, "Isi nama lengkap, minimal 2 karakter"),
  title: z.string().trim().min(2, "Isi jabatan, minimal 2 karakter"),
  // A workspace role (never the Owner role, which only the Owner holds).
  roleId: z.number({ invalid_type_error: "Pilih peran" }).int().positive("Pilih peran"),
  email: z.string().trim().email("Isi email akun Google orang ini, contoh: nama@gmail.com"),
  phone: z
    .string()
    .trim()
    .refine((v) => v === "" || v.replace(/\D/g, "").length >= 9, "Isi nomor WhatsApp minimal 9 angka, atau kosongkan"),
});

export type UserFormValues = z.infer<typeof userSchema>;
