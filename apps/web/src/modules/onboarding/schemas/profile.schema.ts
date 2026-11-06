import { z } from "zod";

/** Indonesian mobile number in any common spelling (0812…, +62 812…, 62812…). */
function normalizeWhatsApp(v: string) {
  let d = v.replace(/\D/g, "");
  if (d.startsWith("0")) d = "62" + d.slice(1);
  else if (d.startsWith("8")) d = "62" + d;
  return d;
}

export const profileSchema = z.object({
  businessName: z.string().trim().min(3, "Nama WO minimal 3 karakter").max(100, "Nama WO maksimal 100 karakter"),
  phone: z
    .string()
    .trim()
    .refine((v) => /^62[1-9]\d{7,12}$/.test(normalizeWhatsApp(v)), "Masukkan nomor WhatsApp yang valid, contoh 0812 3456 7890"),
  city: z.string().trim().min(2, "Kota minimal 2 karakter").max(100, "Kota maksimal 100 karakter"),
});

export type ProfileFormValues = z.infer<typeof profileSchema>;
