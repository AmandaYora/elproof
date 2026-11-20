import { z } from "zod";

/** Digits only ("" = not filled in): guest counts are typed as text so a field can be left empty. */
const count = z.string().regex(/^\d*$/, "Isi dengan angka");

export const packageComponentSchema = z.object({
  /** Stable React key for a row (a new row has no id yet). */
  key: z.string(),
  /** The saved row's id; "" for a row added in this form. */
  id: z.string(),
  description: z.string().trim().min(1, "Isi keterangan isi paket").max(255, "Keterangan maksimal 255 karakter"),
  categoryId: z.string(),
  vendorId: z.string(),
  /** 0 = not filled in. Only sent by a role that may see costs. */
  estimatedCost: z.number().min(0, "Perkiraan biaya tidak boleh kurang dari 0"),
});

/**
 * The package form (DESAIN.md §6.6). The guest rule mirrors the server's own checks (offers/domain ValidateRule) with
 * the same sentences, so a person sees the same message whether the browser or the server caught it.
 */
export const packageSchema = z
  .object({
    name: z.string().trim().min(1, "Isi nama paket").max(150, "Nama paket maksimal 150 karakter"),
    price: z.number().min(0, "Harga paket tidak boleh kurang dari 0"),
    eventKindIds: z.array(z.string()).min(1, "Pilih minimal satu acara yang dicakup"),
    includedGuests: count,
    extraGuestStep: count,
    /** 0 = not filled in. */
    extraGuestStepPrice: z.number().min(0, "Harga tambahan tidak boleh kurang dari 0"),
    defaultVenueId: z.string(),
    components: z.array(packageComponentSchema).max(50, "Isi paket paling banyak 50 baris"),
  })
  .superRefine((v, ctx) => {
    const included = v.includedGuests === "" ? null : Number(v.includedGuests);
    const step = v.extraGuestStep === "" ? null : Number(v.extraGuestStep);
    const hasPrice = v.extraGuestStepPrice > 0;
    if (included !== null && included < 1) {
      ctx.addIssue({ code: "custom", path: ["includedGuests"], message: "Tamu termasuk minimal 1. Kosongkan bila harga tidak bergantung tamu." });
    }
    if ((step !== null && step < 1) || (hasPrice && step === null)) {
      ctx.addIssue({ code: "custom", path: ["extraGuestStep"], message: "Isi jumlah tamu per tambahan, minimal 1" });
    }
    if (step !== null && !hasPrice) {
      ctx.addIssue({ code: "custom", path: ["extraGuestStepPrice"], message: "Isi harga untuk setiap tambahan tamu" });
    }
    if ((step !== null || hasPrice) && included === null) {
      ctx.addIssue({ code: "custom", path: ["includedGuests"], message: "Isi tamu termasuk dulu, baru tambahan tamu bisa dihitung" });
    }
  });

export type PackageFormValues = z.infer<typeof packageSchema>;
export type PackageComponentFormValues = z.infer<typeof packageComponentSchema>;
