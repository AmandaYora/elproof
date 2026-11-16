import { formatCurrency, formatCurrencyCompact } from "@/shared/lib/formatters";

/**
 * A money figure for a KPI card: short ("Rp55 jt"), whole millions from Rp100 jt up so it stays on one line in a
 * phone's half-width card, and the exact amount for assistive technology and on hover.
 */
export function KpiMoney({ amount }: { amount: number }) {
  const short = Math.abs(amount) >= 100_000_000 && Math.abs(amount) < 1_000_000_000 ? `Rp${Math.round(amount / 1_000_000).toLocaleString("id-ID")} jt` : formatCurrencyCompact(amount);
  const exact = formatCurrency(amount);
  return (
    <span title={exact} className="whitespace-nowrap max-sm:text-[24px]">
      <span aria-hidden>{short}</span>
      <span className="sr-only">{exact}</span>
    </span>
  );
}
