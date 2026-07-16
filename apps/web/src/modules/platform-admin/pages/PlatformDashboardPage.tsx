import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { AlertTriangle, Clock, CheckCircle2 } from "lucide-react";
import { StatCard } from "@/shared/components/ui/StatCard";
import { Card, CardHeader, CardContent } from "@/shared/components/ui/Card";
import { EmptyState } from "@/shared/components/feedback/EmptyState";
import { TrendBarChart } from "@/modules/platform-admin/components/TrendBarChart";
import { PeriodToggle } from "@/modules/platform-admin/components/PeriodToggle";
import { getPlatformStats, daysUntilExpiry } from "@/modules/platform-admin/data";
import { usePlatformAdminStore } from "@/modules/platform-admin/stores/usePlatformAdminStore";
import { buildRevenueTrend, buildTenantGrowthTrend, type DashboardPeriod } from "@/modules/platform-admin/lib/trend";
import { formatCurrency, formatCurrencyCompact, formatDate } from "@/shared/lib/formatters";
import { TRANSACTION_TYPE_LABEL } from "@/modules/platform-admin/lib/status";
import { ROUTE_PATHS } from "@/app/routes/route-paths";

export default function PlatformDashboardPage() {
  const tenants = usePlatformAdminStore((s) => s.tenants);
  const transactions = usePlatformAdminStore((s) => s.subscriptionTransactions);
  const fetchTenants = usePlatformAdminStore((s) => s.fetchTenants);
  const fetchTransactions = usePlatformAdminStore((s) => s.fetchTransactions);

  useEffect(() => {
    void fetchTenants();
    void fetchTransactions();
  }, [fetchTenants, fetchTransactions]);

  const stats = getPlatformStats(tenants, transactions);
  const unpaidTransactions = transactions.filter((t) => t.status === "unpaid");
  const attentionTenants = tenants
    .filter((t) => t.subscriptionStatus === "expiring_soon" || t.subscriptionStatus === "expired")
    .sort((a, b) => (daysUntilExpiry(a) ?? 0) - (daysUntilExpiry(b) ?? 0));

  const [period, setPeriod] = useState<DashboardPeriod>("month");
  const periodNoun = period === "month" ? "hari" : "bulan";
  const periodLabel = period === "month" ? "Bulan Ini" : "Tahun Ini";
  const revenuePoints = buildRevenueTrend(transactions, period);
  const tenantGrowthPoints = buildTenantGrowthTrend(tenants, period);

  return (
    <div className="flex flex-col gap-6">
      <div>
        <h1 className="text-xl font-bold text-text-primary">Dashboard</h1>
        <p className="mt-1 text-[13px] text-text-secondary">Ringkasan langganan seluruh tenant WO di platform ElProof.</p>
      </div>

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <StatCard label="Total Tenant" value={stats.totalTenants} sublabel="Lihat seluruh tenant" to={ROUTE_PATHS.platformTenants} />
        <StatCard label="Tenant Aktif" value={stats.activeTenants} tone="success" sublabel="Langganan berjalan" />
        <StatCard
          label="Menunggu Pembayaran"
          value={stats.unpaidCount}
          tone={stats.unpaidCount > 0 ? "warning" : "success"}
          sublabel="Lihat transaksi"
          to={ROUTE_PATHS.platformTransactions}
        />
        <StatCard label="Pendapatan Terbayar" value={formatCurrencyCompact(stats.paidRevenue)} tone="info" sublabel="Total sepanjang berjalan" />
      </div>

      <div className="flex items-center justify-end">
        <PeriodToggle value={period} onChange={setPeriod} />
      </div>

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
        <TrendBarChart
          title={`Omzet ${periodLabel}`}
          subtitle={`Pendapatan yang berhasil dibayar tenant, per ${periodNoun}.`}
          points={revenuePoints}
          totalLabel={`total omzet ${periodLabel.toLowerCase()}`}
          formatTotal={formatCurrencyCompact}
          formatBarLabel={(value) => formatCurrencyCompact(value).replace(/^Rp\s*/, "")}
          formatTooltip={(label, value) => `${label}: ${formatCurrency(value)}`}
          emptyTooltip={(label) => `${label}: tidak ada omzet`}
        />
        <TrendBarChart
          title={`Tenant Baru ${periodLabel}`}
          subtitle={`Jumlah tenant baru bergabung, per ${periodNoun}.`}
          points={tenantGrowthPoints}
          totalLabel={`tenant baru ${periodLabel.toLowerCase()}`}
          formatTotal={(value) => String(value)}
          formatBarLabel={(value) => String(value)}
          formatTooltip={(label, value) => `${label}: ${value} tenant baru`}
          emptyTooltip={(label) => `${label}: tidak ada tenant baru`}
        />
      </div>

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
        <Card>
          <CardHeader title="Menunggu Pembayaran" subtitle="Invoice yang belum diselesaikan tenant." />
          <CardContent className="p-0">
            {unpaidTransactions.length === 0 ? (
              <EmptyState
                icon={<CheckCircle2 className="h-8 w-8 text-success" />}
                title="Tidak ada invoice tertunda"
                description="Seluruh transaksi tenant sudah selesai diproses."
              />
            ) : (
              <ul className="divide-y divide-border-light">
                {unpaidTransactions.map((tx) => {
                  const tenant = tenants.find((t) => t.id === tx.tenantId);
                  return (
                    <li key={tx.id}>
                      <Link
                        to={ROUTE_PATHS.platformTransactions}
                        className="flex items-start gap-3 px-5 py-3.5 transition-colors hover:bg-surface-muted"
                      >
                        <Clock className="mt-0.5 h-4 w-4 shrink-0 text-warning" />
                        <span className="min-w-0 flex-1">
                          <span className="block text-[13.5px] font-semibold text-text-primary">{tenant?.businessName ?? "-"}</span>
                          <span className="block truncate text-[12.5px] text-text-secondary">
                            {TRANSACTION_TYPE_LABEL[tx.type]} · dibuat {formatDate(tx.createdAt)}
                          </span>
                        </span>
                      </Link>
                    </li>
                  );
                })}
              </ul>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader title="Tenant Perlu Diperhatikan" subtitle="Segera berakhir atau sudah berakhir masa langganannya." />
          <CardContent className="p-0">
            {attentionTenants.length === 0 ? (
              <EmptyState
                icon={<CheckCircle2 className="h-8 w-8 text-success" />}
                title="Semua tenant aman"
                description="Tidak ada langganan tenant yang segera atau sudah berakhir."
              />
            ) : (
              <ul className="divide-y divide-border-light">
                {attentionTenants.map((tenant) => {
                  const d = daysUntilExpiry(tenant);
                  return (
                    <li key={tenant.id}>
                      <Link
                        to={ROUTE_PATHS.platformTenants}
                        className="flex items-start gap-3 px-5 py-3.5 transition-colors hover:bg-surface-muted"
                      >
                        <AlertTriangle className={`mt-0.5 h-4 w-4 shrink-0 ${tenant.subscriptionStatus === "expired" ? "text-danger" : "text-warning"}`} />
                        <span className="min-w-0 flex-1">
                          <span className="block text-[13.5px] font-semibold text-text-primary">{tenant.businessName}</span>
                          <span className="block truncate text-[12.5px] text-text-secondary">
                            {d !== null && d >= 0 ? `Berakhir dalam H-${d}` : `Sudah berakhir H+${Math.abs(d ?? 0)}`}
                          </span>
                        </span>
                      </Link>
                    </li>
                  );
                })}
              </ul>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
