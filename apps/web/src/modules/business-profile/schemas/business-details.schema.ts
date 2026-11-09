import { z } from "zod";

// Mirrors the API's own rules (onboarding_service SaveBusinessDetails): all
// optional, but a bank account is all-or-nothing, so a number never reaches
// clients without its bank and holder.
const ACCOUNT_NUMBER = /^[0-9][0-9 .-]{3,38}[0-9]$/;

export const businessDetailsSchema = z
  .object({
    address: z.string().max(255, "Alamat maksimal 255 karakter"),
    bankName: z.string(),
    bankAccountNumber: z.string(),
    bankAccountHolder: z.string(),
  })
  .superRefine((v, ctx) => {
    const name = v.bankName.trim();
    const number = v.bankAccountNumber.trim();
    const holder = v.bankAccountHolder.trim();
    if (!name && !number && !holder) return;
    if (name.length < 2 || name.length > 60) ctx.addIssue({ code: "custom", path: ["bankName"], message: "Isi nama bank, 2–60 karakter" });
    if (!ACCOUNT_NUMBER.test(number)) ctx.addIssue({ code: "custom", path: ["bankAccountNumber"], message: "Isi nomor rekening, 5–40 angka" });
    if (holder.length < 2 || holder.length > 100) ctx.addIssue({ code: "custom", path: ["bankAccountHolder"], message: "Isi nama pemilik rekening, 2–100 karakter" });
  });

export type BusinessDetailsFormValues = z.infer<typeof businessDetailsSchema>;
