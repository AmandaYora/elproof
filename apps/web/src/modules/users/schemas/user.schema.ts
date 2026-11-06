import { z } from "zod";

// Selectable when inviting/editing Admin, Wedding Planner and Sales — Owner
// is never an option (fixed at tenant registration), but an Owner editing
// their own row round-trips role="Owner" through the same form.
export const STAFF_ROLE_OPTIONS = ["Admin", "Staff", "Sales"] as const;

// Staff sign in with Google only (decision K5): the email is the Google
// account that must accept the invitation. No username or password here.
export const userSchema = z.object({
  name: z.string().trim().min(2, "Nama wajib diisi"),
  title: z.string().trim().min(2, "Jabatan wajib diisi"),
  role: z.enum(["Owner", "Admin", "Staff", "Sales"]),
  email: z.string().trim().email("Masukkan email akun Google orang ini"),
  phone: z
    .string()
    .trim()
    .refine((v) => v === "" || v.replace(/\D/g, "").length >= 9, "Nomor WhatsApp tidak valid"),
});

export type UserFormValues = z.infer<typeof userSchema>;
