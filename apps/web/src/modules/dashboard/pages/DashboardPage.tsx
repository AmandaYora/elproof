import { useEffect } from "react";
import { StatCard } from "@/shared/components/ui/StatCard";
import { AttentionQueue } from "@/modules/dashboard/components/AttentionQueue";
import { UpcomingEvents } from "@/modules/dashboard/components/UpcomingEvents";
import { RecentActivity } from "@/modules/dashboard/components/RecentActivity";
import { ProjectTrendChart } from "@/modules/dashboard/components/ProjectTrendChart";
import { RevenueTrendChart } from "@/modules/dashboard/components/RevenueTrendChart";
import { buildAttentionItems } from "@/modules/dashboard/lib/attention";
import { useDashboardStore } from "@/modules/dashboard/stores/useDashboardStore";
import { useVendorStore } from "@/modules/vendors/stores/useVendorStore";
import { formatCurrencyCompact } from "@/shared/lib/formatters";
import { ROUTE_PATHS } from "@/app/routes/route-paths";

export default function DashboardPage() {
  const stats = useDashboardStore((s) => s.stats);
  const fetchDashboard = useDashboardStore((s) => s.fetchDashboard);
  const vendors = useVendorStore((s) => s.vendors);
  const fetchVendors = useVendorStore((s) => s.fetchVendors);

  useEffect(() => {
    void fetchDashboard();
    void fetchVendors();
  }, [fetchDashboard, fetchVendors]);

  if (!stats) {
    return <div className="py-16 text-center text-sm text-text-secondary">Memuat...</div>;
  }

  const { items, counts } = buildAttentionItems(stats, vendors);
  const revenue = stats.revenue;
  const revenueSublabel =
    revenue.deltaPercent === null
      ? "Nilai kontrak event yang berlangsung bulan ini"
      : `${revenue.deltaPercent >= 0 ? "+" : ""}${Math.round(revenue.deltaPercent)}% dari bulan lalu`;

  return (
    <div className="flex flex-col gap-6">
      <div>
        <h1 className="text-xl font-bold text-text-primary">Dashboard</h1>
        <p className="mt-1 text-[13px] text-text-secondary">Ringkasan kondisi operasional seluruh project pernikahan.</p>
      </div>

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-5">
        <StatCard
          label="Omzet Bulan Ini"
          value={formatCurrencyCompact(revenue.total)}
          tone={revenue.deltaPercent === null ? "info" : revenue.deltaPercent >= 0 ? "success" : "warning"}
          sublabel={revenueSublabel}
        />
        <StatCard label="Seluruh Project" value={stats.totalProjects} sublabel="Lihat semua project" to={ROUTE_PATHS.projects} />
        <StatCard label="Project Aktif" value={stats.activeProjects} sublabel="Preparation & Ready" tone="info" to={ROUTE_PATHS.projects} />
        <StatCard label="Vendor Aktif" value={stats.activeVendorCount} sublabel="Terlibat di project berjalan" to={ROUTE_PATHS.vendors} />
        <StatCard
          label="Kendala Terbuka"
          value={stats.openIssues.length}
          tone={stats.openIssues.length > 0 ? "danger" : "success"}
          sublabel="Belum diselesaikan"
        />
      </div>

      <RevenueTrendChart points={stats.revenueTrend} />

      <ProjectTrendChart points={stats.projectTrend} />

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
        <div className="lg:col-span-2">
          <AttentionQueue items={items} counts={counts} />
        </div>
        <div className="flex flex-col gap-6">
          <UpcomingEvents projects={stats.upcomingProjects} />
          <RecentActivity activity={stats.recentActivity} />
        </div>
      </div>
    </div>
  );
}
