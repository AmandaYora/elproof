import { z } from "zod";

export const usernameSchema = z
  .string()
  .min(4, "Username minimal 4 karakter")
  .max(32, "Username maksimal 32 karakter")
  .regex(/^[a-z0-9_.]+$/, "Hanya huruf kecil, angka, underscore, dan titik");
