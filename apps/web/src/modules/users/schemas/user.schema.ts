import { z } from "zod";
import { usernameSchema } from "@/shared/lib/validators";

// Selectable when creating/editing Admin & Staff accounts — Owner is never an
// option here (its role is fixed at tenant registration, see staff module's
// backend guard), but an Owner editing their own account still round-trips
// role="Owner" through this same form, hence the wider enum below.
export const STAFF_ROLE_OPTIONS = ["Admin", "Staff"] as const;

export const userSchema = z.object({
  name: z.string().min(2, "Nama wajib diisi"),
  title: z.string().min(2, "Jabatan wajib diisi"),
  role: z.enum(["Owner", "Admin", "Staff"]),
  email: z.string().email("Email tidak valid"),
  phone: z.string().min(6, "Nomor telepon tidak valid"),
});

export type UserFormValues = z.infer<typeof userSchema>;

export const userCreateSchema = userSchema
  .extend({
    username: usernameSchema,
    password: z.string().min(8, "Password minimal 8 karakter"),
    confirmPassword: z.string().min(1, "Konfirmasi password wajib diisi"),
  })
  .refine((values) => values.password === values.confirmPassword, {
    message: "Konfirmasi password tidak sama",
    path: ["confirmPassword"],
  });

export type UserCreateFormValues = z.infer<typeof userCreateSchema>;
