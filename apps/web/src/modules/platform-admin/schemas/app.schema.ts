import { z } from "zod";

export const createAppSchema = z.object({
  name: z.string().trim().min(2, "Isi nama aplikasi, minimal 2 karakter"),
  callbackUrl: z.string().trim().url("Isi URL callback lengkap, misalnya https://aplikasi.example.com/webhook"),
});

export type CreateAppFormValues = z.infer<typeof createAppSchema>;
