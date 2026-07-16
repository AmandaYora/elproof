import { z } from "zod";

export const loginSchema = z.object({
  username: z.string().min(1, "Nama pengguna atau email wajib diisi"),
  password: z.string().min(6, "Kata sandi minimal 6 karakter"),
});

export type LoginFormValues = z.infer<typeof loginSchema>;
