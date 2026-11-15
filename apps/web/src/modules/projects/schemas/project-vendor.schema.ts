import { z } from "zod";
import { formatCurrency } from "@/shared/lib/formatters";

export const ENGAGEMENT_STATUS_OPTIONS = [
  "Planned",
  "Negotiation",
  "Booked",
  "DP Paid",
  "In Progress",
  "Fully Paid",
  "Ready",
  "Completed",
  "Cancelled",
] as const;

export const PRICING_TIER_OPTIONS = ["Akad", "AkadResepsi", "Resepsi"] as const;

// Fixed presets for "Jam Acara" (PLAN.md revisi-timeline-vendor-role-sales) —
// selecting one fills both time fields; "Custom" leaves them for manual
// entry instead.
export const EVENT_HOURS_PRESETS = [
  { label: "08.00 - 13.00", start: "08:00", end: "13:00" },
  { label: "16.00 - 21.00", start: "16:00", end: "21:00" },
  { label: "06.00 - 21.00", start: "06:00", end: "21:00" },
  { label: "10.45 - 13.00", start: "10:45", end: "13:00" },
  { label: "18.45 - 21.00", start: "18:45", end: "21:00" },
] as const;

export const projectVendorSchema = z
  .object({
    vendorId: z.string().min(1, "Pilih vendor"),
    // Optional — memilih Vendor tetap menyinkronkan field ini ke kategori
    // vendor tersebut; field ini sendiri hanya memfilter dropdown Vendor.
    categoryId: z.string().optional().default(""),
    scope: z.string().min(3, "Isi layanan atau scope pekerjaan, minimal 3 karakter"),
    contractValue: z.coerce.number().min(0, "Nilai kerja sama tidak boleh negatif"),
    pricingTier: z.enum(PRICING_TIER_OPTIONS),
    engagementStatus: z.enum(ENGAGEMENT_STATUS_OPTIONS),
    bookingDate: z.string().optional().default(""),
    // Opsional (keputusan analis, dikonfirmasi pengguna) — "HH:MM" atau "".
    eventStartTime: z.string().optional().default(""),
    eventEndTime: z.string().optional().default(""),
    dpAmount: z.coerce.number().min(0, "Jumlah DP tidak boleh negatif"),
    dueDate: z.string().optional().default(""),
    picStaffId: z.string().min(1, "Pilih PIC internal WO"),
    notes: z.string().optional().default(""),
  })
  // Same rule as the server (QA EP-04): the DP stays within the agreed price. A price of 0 is "not set yet" and
  // limits nothing, like the payment ledger. Lowering the price under what was already paid is refused by the
  // server and shown on contractValue.
  .superRefine((v, ctx) => {
    if (v.contractValue > 0 && v.dpAmount > v.contractValue) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        path: ["dpAmount"],
        message: `Jumlah DP (${formatCurrency(v.dpAmount)}) melebihi nilai kerja sama (${formatCurrency(v.contractValue)}). Turunkan DP atau naikkan nilai kerja sama.`,
      });
    }
  });

export type ProjectVendorFormValues = z.infer<typeof projectVendorSchema>;
