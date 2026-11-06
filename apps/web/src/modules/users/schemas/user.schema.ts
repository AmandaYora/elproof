import { z } from "zod";

// Staff sign in with Google only (decision K5): the email is the Google
// account that must accept the invitation. No username or password here.
export const userSchema = z.object({
  name: z.string().trim().min(2, "Nama wajib diisi"),
  title: z.string().trim().min(2, "Jabatan wajib diisi"),
  // A workspace role (never the Owner role, which only the Owner holds).
  roleId: z.number({ invalid_type_error: "Pilih peran" }).int().positive("Pilih peran"),
  email: z.string().trim().email("Masukkan email akun Google orang ini"),
  phone: z
    .string()
    .trim()
    .refine((v) => v === "" || v.replace(/\D/g, "").length >= 9, "Nomor WhatsApp tidak valid"),
});

export type UserFormValues = z.infer<typeof userSchema>;
