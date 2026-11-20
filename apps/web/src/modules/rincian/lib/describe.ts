import { formatCurrency } from "@/shared/lib/formatters";
import type { ExtraGuestsInfo } from "@/modules/rincian/types";

const count = (n: number) => n.toLocaleString("id-ID");

/**
 * How a "Tambahan tamu" line was worked out, written on its row (DESAIN D5): "650 tamu − 500 termasuk = 150 → 3 × 50
 * tamu × Rp4.000.000". The numbers are the server's; this only puts them in words.
 */
export function extraGuestsFormula(x: ExtraGuestsInfo): string {
  if (x.noGuests) return `Belum ada tamu di ${x.eventKindName || "acara paket ini"}`;
  if (x.over <= 0) return `${count(x.guests)} tamu, masih dalam ${count(x.includedGuests)} tamu paket`;
  const price = x.stepPrice !== undefined ? ` × ${formatCurrency(x.stepPrice)}` : "";
  return `${count(x.guests)} tamu − ${count(x.includedGuests)} termasuk = ${count(x.over)} → ${count(x.steps)} × ${count(x.step)} tamu${price}`;
}

/** The same for the couple (DESAIN §6.11): "150 tamu di atas 500 yang termasuk paket". */
export function extraGuestsPlain(x: ExtraGuestsInfo): string {
  if (x.noGuests) return "Jumlah tamu belum diisi";
  if (x.over <= 0) return `${count(x.guests)} tamu, masih termasuk paket`;
  return `${count(x.over)} tamu di atas ${count(x.includedGuests)} yang termasuk paket`;
}

/** "300 pcs × Rp12.000" (no price when it is hidden). */
export function itemFormula(quantity: number, unit: string, unitPrice?: number): string {
  const qty = `${count(quantity)}${unit ? ` ${unit}` : ""}`;
  return unitPrice !== undefined ? `${qty} × ${formatCurrency(unitPrice)}` : qty;
}
