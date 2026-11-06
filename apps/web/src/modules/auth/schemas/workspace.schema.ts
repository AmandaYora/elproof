import { z } from "zod";

export const workspaceSchema = z.object({
  businessName: z
    .string()
    .trim()
    .min(3, "Nama WO minimal 3 karakter")
    .max(100, "Nama WO maksimal 100 karakter"),
  acceptTerms: z
    .boolean()
    .refine((v) => v, "Setujui Syarat & Ketentuan dan Perjanjian Pemrosesan Data untuk melanjutkan"),
  marketingOptIn: z.boolean(),
});

export type WorkspaceFormValues = z.infer<typeof workspaceSchema>;
