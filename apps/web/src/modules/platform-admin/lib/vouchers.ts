import { formatCurrency, formatDate } from "@/shared/lib/formatters";
import type { Voucher } from "@/modules/platform-admin/stores/useVoucherStore";

/** "10% · maks Rp50.000" or "Rp100.000". */
export function discountLabel(v: Pick<Voucher, "discountType" | "discountValue" | "maxDiscount">): string {
  if (v.discountType === "amount") return formatCurrency(v.discountValue);
  return v.maxDiscount ? `${v.discountValue}% · maks ${formatCurrency(v.maxDiscount)}` : `${v.discountValue}%`;
}

/** "1 Nov 2026 – 31 Des 2026", "Mulai 1 Nov 2026", "Sampai 31 Des 2026" or "Tanpa batas". */
export function validityLabel(v: Pick<Voucher, "validFrom" | "validUntil">): string {
  if (v.validFrom && v.validUntil) return `${formatDate(v.validFrom)} – ${formatDate(v.validUntil)}`;
  if (v.validFrom) return `Mulai ${formatDate(v.validFrom)}`;
  if (v.validUntil) return `Sampai ${formatDate(v.validUntil)}`;
  return "Tanpa batas";
}

/** From this share of the price a voucher counts as a big discount (EP-17). */
export const BIG_DISCOUNT_PERCENT = 50;

/**
 * Why a voucher about to be saved needs an explicit "yes, on purpose" (EP-17): a big discount (≥ 50%, or a fixed amount of at
 * least half the chosen plan's price) with no end date or no total quota can be used without limit. null = nothing to confirm.
 */
export function bigOpenEndedDiscount(
  v: { discountType: "percent" | "amount"; discountValue: number; validUntil: string; quota: string },
  planPrice?: number
): string | null {
  const big =
    v.discountType === "percent"
      ? v.discountValue >= BIG_DISCOUNT_PERCENT
      : planPrice !== undefined && planPrice > 0 && v.discountValue * 100 >= planPrice * BIG_DISCOUNT_PERCENT;
  if (!big) return null;
  const open = [v.validUntil === "" && "tanpa tanggal akhir", v.quota.trim() === "" && "tanpa kuota total"].filter(Boolean);
  if (open.length === 0) return null;
  const size = v.discountType === "percent" ? `${v.discountValue}%` : `${formatCurrency(v.discountValue)} (setengah harga paket atau lebih)`;
  return `Potongan ${size} ${open.join(" dan ")} bisa dipakai tanpa batas.`;
}

/** "3 / 10", "3 / tanpa batas", plus "· 1 menunggu" while payments that hold a use are still pending. */
export function usageLabel(v: Pick<Voucher, "usedCount" | "reservedCount" | "quota">): string {
  const base = `${v.usedCount} / ${v.quota ?? "tanpa batas"}`;
  return v.reservedCount > 0 ? `${base} · ${v.reservedCount} menunggu` : base;
}
