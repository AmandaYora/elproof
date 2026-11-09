import { z } from "zod";

export const CLIENT_ROLE_OPTIONS = ["Bride", "Groom", "Family Representative"] as const;

const optionalEmail = z
  .string()
  .trim()
  .refine((v) => v === "" || z.string().email().safeParse(v).success, "Periksa email, contoh: nama@email.com");

export const clientContactSchema = z.object({
  name: z.string().trim().min(2, "Isi nama, minimal 2 karakter"),
  // WhatsApp: the invitation and reset links are sent there.
  phone: z
    .string()
    .trim()
    .refine((v) => v.replace(/\D/g, "").length >= 9, "Isi nomor WhatsApp, minimal 9 angka"),
  email: optionalEmail,
  /** Portal role (ADR-0026); 0 / absent = keep / default. */
  accessRoleId: z.number().int().nonnegative().optional(),
});

export type ClientContactFormValues = z.infer<typeof clientContactSchema>;

// No password: the client sets their own through the invitation link
// (PLAN.md Fase 3). Username is optional — derived from the name when empty.
export const clientCreateSchema = clientContactSchema.extend({
  relationNote: z.string().trim().optional().default(""),
  username: z
    .string()
    .trim()
    .refine((v) => v === "" || /^[a-z0-9_.]{4,32}$/.test(v), "Username 4–32 karakter: huruf kecil, angka, titik, atau garis bawah"),
});

export type ClientCreateFormValues = z.infer<typeof clientCreateSchema>;

export const representativeSchema = clientContactSchema.extend({
  relationNote: z.string().trim().min(2, "Isi hubungan dengan pengantin"),
});

export type RepresentativeFormValues = z.infer<typeof representativeSchema>;
