import { z } from "zod";

/** Keys match backend domain.ManualActivationReason. */
export const ACTIVATION_REASON_OPTIONS = [
  { value: "offline_payment", label: "Pembayaran di luar gateway" },
  { value: "sales_deal", label: "Kesepakatan sales" },
  { value: "compensation", label: "Kompensasi layanan" },
  { value: "trial_extension", label: "Perpanjangan masa coba" },
  { value: "internal", label: "Akun internal/demo" },
  { value: "other", label: "Lainnya" },
] as const;

export const activateSubscriptionSchema = z
  .object({
    planId: z.string().min(1, "Pilih paket langganan"),
    reason: z.string().min(1, "Pilih alasan aktivasi"),
    proofReference: z
      .string()
      .trim()
      .min(3, "Isi bukti aktivasi, misalnya no. transfer, no. invoice, atau tautan dokumen")
      .max(255, "Bukti maksimal 255 karakter"),
    note: z.string().trim().max(500, "Catatan maksimal 500 karakter"),
  })
  .refine((v) => v.reason !== "other" || v.note.length >= 5, {
    path: ["note"],
    message: "Jelaskan alasan aktivasi (minimal 5 karakter)",
  });

export type ActivateSubscriptionFormValues = z.infer<typeof activateSubscriptionSchema>;
