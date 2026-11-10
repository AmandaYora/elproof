import { TrendBarChart } from "@/shared/components/ui/TrendBarChart";
import type { ProjectTrendPoint } from "@/modules/dashboard/types";

export function ProjectTrendChart({ points }: { points: ProjectTrendPoint[] }) {
  return (
    <TrendBarChart
      title="Tren proyek baru"
      subtitle="Jumlah proyek baru per bulan, berdasarkan tanggal proyek dibuat."
      points={points.map((p) => ({ key: p.key, label: p.label, value: p.count }))}
      formatTotal={(total) => String(total)}
      totalLabel="proyek baru dalam 12 bulan terakhir"
      formatBarLabel={(value) => String(value)}
      formatValue={(value) => `${value} proyek`}
      valueHeader="Proyek baru"
      periodHeader="Bulan"
    />
  );
}
