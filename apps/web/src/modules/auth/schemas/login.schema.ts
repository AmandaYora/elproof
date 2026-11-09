import { z } from "zod";

export const loginSchema = z.object({
  username: z.string().min(1, "Isi nama pengguna atau email"),
  password: z.string().min(1, "Isi kata sandi").min(6, "Kata sandi minimal 6 karakter"),
});

export type LoginFormValues = z.infer<typeof loginSchema>;

/** Confirming the ElProof password once to link a Google account to an existing staff login. */
export const googleLinkSchema = z.object({
  password: z.string().min(1, "Isi kata sandi ElProof akun ini"),
});

/** Second login step (and Platform Console enrolment): the 6-digit authenticator code. */
export const mfaCodeSchema = z.object({
  code: z.string().trim().length(6, "Masukkan 6 digit kode dari aplikasi authenticator"),
});
