import { z } from "zod";

// Same bounds the API enforces (role_service: 2–60 characters for the name,
// 255 for the description); the permission matrix and scope are not text
// fields and stay outside the schema.
export const roleSchema = z.object({
  name: z.string().trim().min(2, "Isi nama peran, minimal 2 karakter").max(60, "Nama peran maksimal 60 karakter"),
  description: z.string().max(255, "Deskripsi maksimal 255 karakter"),
});

export type RoleFormValues = z.infer<typeof roleSchema>;
