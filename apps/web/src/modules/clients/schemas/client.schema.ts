import { z } from "zod";

export const CLIENT_ROLE_OPTIONS = ["Bride", "Groom", "Family Representative"] as const;

export const clientContactSchema = z.object({
  name: z.string().min(2, "Nama wajib diisi"),
  phone: z.string().min(6, "Nomor telepon tidak valid"),
  email: z.string().email("Email tidak valid"),
});

export type ClientContactFormValues = z.infer<typeof clientContactSchema>;

export const clientCreateSchema = clientContactSchema.extend({
  relationNote: z.string().optional().default(""),
  password: z.string().min(6, "Password minimal 6 karakter"),
});

export type ClientCreateFormValues = z.infer<typeof clientCreateSchema>;

export const representativeSchema = clientContactSchema.extend({
  relationNote: z.string().min(2, "Keterangan hubungan wajib diisi"),
});

export type RepresentativeFormValues = z.infer<typeof representativeSchema>;
