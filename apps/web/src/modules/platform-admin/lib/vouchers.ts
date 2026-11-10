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

/** "3 / 10", "3 / tanpa batas", plus "· 1 menunggu" while payments that hold a use are still pending. */
export function usageLabel(v: Pick<Voucher, "usedCount" | "reservedCount" | "quota">): string {
  const base = `${v.usedCount} / ${v.quota ?? "tanpa batas"}`;
  return v.reservedCount > 0 ? `${base} · ${v.reservedCount} menunggu` : base;
}
