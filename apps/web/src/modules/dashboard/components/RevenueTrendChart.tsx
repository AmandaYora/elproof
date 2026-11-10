import { TrendBarChart } from "@/shared/components/ui/TrendBarChart";
import type { RevenueTrendPoint } from "@/modules/dashboard/types";
import { formatCurrency, formatCurrencyCompact } from "@/shared/lib/formatters";

export function RevenueTrendChart({ points }: { points: RevenueTrendPoint[] }) {
  return (
    <TrendBarChart
      title="Omzet 12 bulan terakhir"
      subtitle="Total nilai kontrak proyek berdasarkan bulan pelaksanaan acara."
      points={points.map((p) => ({ key: p.key, label: p.label, value: p.total }))}
      formatTotal={formatCurrencyCompact}
      totalLabel="total omzet dalam 12 bulan terakhir"
      formatBarLabel={(value) => formatCurrencyCompact(value).replace(/^Rp\s*/, "")}
      formatValue={formatCurrency}
      valueHeader="Omzet"
      periodHeader="Bulan"
    />
  );
}
