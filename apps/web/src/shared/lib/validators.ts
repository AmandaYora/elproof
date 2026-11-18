import { z } from "zod";

export const usernameSchema = z
  .string()
  .min(4, "Nama pengguna minimal 4 karakter")
  .max(32, "Nama pengguna maksimal 32 karakter")
  .regex(/^[a-z0-9_.]+$/, "Hanya huruf kecil, angka, garis bawah, dan titik");
