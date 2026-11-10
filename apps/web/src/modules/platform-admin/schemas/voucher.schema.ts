import { z } from "zod";

const CODE = /^[A-Z0-9_-]{3,32}$/;
const DAY = /^\d{4}-\d{2}-\d{2}$/;

/**
 * The voucher form (ADR-0041). Money is whole rupiah. Empty optional fields mean "no limit": planId "" = every plan of
 * the App, maxDiscount 0 = no cap, quota "" = unlimited, dates "" = open-ended. The API checks everything again and its
 * 422 uses these same field names.
 */
export const voucherSchema = z
  .object({
    code: z
      .string()
      .trim()
      .transform((s) => s.toUpperCase())
      .pipe(z.string().min(1, "Isi kode voucher").regex(CODE, "Kode 3–32 karakter: huruf, angka, tanda hubung (-) atau garis bawah (_)")),
    appId: z.string().min(1, "Pilih aplikasi voucher ini"),
    planId: z.string(),
    discountType: z.enum(["percent", "amount"], { errorMap: () => ({ message: "Pilih jenis potongan" }) }),
    discountValue: z.number({ invalid_type_error: "Isi nilai potongan dengan angka" }).int("Nilai potongan harus bilangan bulat"),
    maxDiscount: z.number().int("Batas potongan harus bilangan bulat").min(0, "Batas potongan tidak boleh negatif"),
    validFrom: z.string().refine((s) => s === "" || DAY.test(s), "Pilih tanggal yang benar"),
    validUntil: z.string().refine((s) => s === "" || DAY.test(s), "Pilih tanggal yang benar"),
    quota: z.string().trim().refine((s) => s === "" || /^\d+$/.test(s), "Isi kuota dengan angka bulat, atau kosongkan"),
    oncePerCustomer: z.boolean(),
  })
  .superRefine((v, ctx) => {
    if (v.discountType === "percent" && (v.discountValue < 1 || v.discountValue > 99)) {
      ctx.addIssue({ code: z.ZodIssueCode.custom, path: ["discountValue"], message: "Potongan persen 1–99%" });
    }
    if (v.discountType === "amount" && v.discountValue < 1) {
      ctx.addIssue({ code: z.ZodIssueCode.custom, path: ["discountValue"], message: "Potongan minimal Rp1" });
    }
    if (v.validFrom && v.validUntil && v.validUntil < v.validFrom) {
      ctx.addIssue({ code: z.ZodIssueCode.custom, path: ["validUntil"], message: "Tanggal akhir tidak boleh sebelum tanggal mulai" });
    }
    if (v.quota !== "" && (Number(v.quota) < 1 || Number(v.quota) > 1_000_000_000)) {
      ctx.addIssue({ code: z.ZodIssueCode.custom, path: ["quota"], message: "Kuota 1–1.000.000.000, atau kosongkan untuk tanpa batas" });
    }
  });

export type VoucherFormValues = z.infer<typeof voucherSchema>;
