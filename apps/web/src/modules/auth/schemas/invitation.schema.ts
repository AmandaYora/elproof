import { z } from "zod";

export const MIN_PASSWORD = 8;

/** A client choosing their own password from an invitation or reset link. */
export const invitationPasswordSchema = z
  .object({
    password: z.string().min(1, "Isi kata sandi baru").min(MIN_PASSWORD, `Kata sandi minimal ${MIN_PASSWORD} karakter`),
    confirm: z.string().min(1, "Ulangi kata sandi baru"),
  })
  .refine((v) => v.password === v.confirm, { path: ["confirm"], message: "Kedua kata sandi belum sama" });
