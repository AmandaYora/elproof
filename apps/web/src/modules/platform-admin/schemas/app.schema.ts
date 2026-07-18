import { z } from "zod";

export const createAppSchema = z.object({
  name: z.string().min(2, "Nama aplikasi wajib diisi"),
  callbackUrl: z.string().url("URL callback tidak valid"),
});

export type CreateAppFormValues = z.infer<typeof createAppSchema>;
