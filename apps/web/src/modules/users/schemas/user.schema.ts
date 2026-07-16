import { z } from "zod";

export const STAFF_ROLE_OPTIONS = ["Admin", "Staff"] as const;

export const userSchema = z.object({
  name: z.string().min(2, "Nama wajib diisi"),
  title: z.string().min(2, "Jabatan wajib diisi"),
  role: z.enum(STAFF_ROLE_OPTIONS),
  email: z.string().email("Email tidak valid"),
  phone: z.string().min(6, "Nomor telepon tidak valid"),
});

export type UserFormValues = z.infer<typeof userSchema>;
