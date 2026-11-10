import { useCallback, useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { AlertTriangle, CheckCircle2, Clock } from "lucide-react";
import { StatCard } from "@/shared/components/ui/StatCard";
import { Card, CardHeader, CardContent } from "@/shared/components/ui/Card";
import { EmptyState } from "@/shared/components/feedback/EmptyState";
import { LoadError } from "@/shared/components/feedback/LoadError";
import { TrendBarChart } from "@/shared/components/ui/TrendBarChart";
import { PeriodToggle } from "@/modules/platform-admin/components/PeriodToggle";
import { TENANT_STATUS_LABEL } from "@/modules/platform-admin/lib/status";
import type { AttentionTenant } from "@/modules/platform-admin/data/console.types";
import { usePlatformCan } from "@/modules/platform-admin/lib/capabilities";
import { useConsoleStore } from "@/modules/platform-admin/stores/useConsoleStore";
import type { DashboardPeriod } from "@/modules/platform-admin/lib/trend";
import { formatCurrency, formatCurrencyCompact } from "@/shared/lib/formatters";
import { ROUTE_PATHS } from "@/app/routes/route-paths";

function pct(value: number | null): string {
  return value === null ? "-" : `${value.toLocaleString("id-ID", { maximumFractionDigits: 1 })}%`;
}

/** "+12%" / "-8%" against the 30 days before; null when there is no base to compare to. */
function change(now: number, before: number): string | null {
  if (before <= 0) return null;
  const delta = ((now - before) / before) * 100;
  return `${delta >= 0 ? "+" : ""}${delta.toLocaleString("id-ID", { maximumFractionDigits: 0 })}%`;
}

function row(a: AttentionTenant, tone: "warning" | "danger", describe: (a: AttentionTenant) => string) {
  return (
    <>
      {tone === "danger" ? <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0 text-danger-text" /> : <Clock className="mt-0.5 h-4 w-4 shrink-0 text-warning-text" />}
      <span className="min-w-0 flex-1">
        <span className="block text-[13.5px] font-semibold text-text-primary">{a.name}</span>
        <span className="block truncate text-[12.5px] text-text-secondary">{describe(a)}</span>
      </span>
    </>
  );
}

function AttentionList({ items, empty, tone, describe, canOpen }: { items: AttentionTenant[]; empty: { title: string; description: string }; tone: "warning" | "danger"; describe: (a: AttentionTenant) => string; canOpen: boolean }) {
  if (items.length === 0) {
    return <EmptyState icon={<CheckCircle2 className="h-8 w-8 text-success-text" />} title={empty.title} description={empty.description} />;
  }
  return (
    <ul className="divide-y divide-border-light">
      {items.map((a) => (
        <li key={a.tenantId}>
          {canOpen ? (
            <Link to={ROUTE_PATHS.platformTenantDetail(String(a.tenantId), "langganan")} className="flex items-start gap-3 px-5 py-3.5 transition-colors hover:bg-surface-muted">
              {row(a, tone, describe)}
            </Link>
          ) : (
            <div className="flex items-start gap-3 px-5 py-3.5">{row(a, tone, describe)}</div>
          )}
        </li>
      ))}
    </ul>
  );
}

export default function PlatformDashboardPage() {
  const metrics = useConsoleStore((s) => s.metrics);
  const fetchMetrics = useConsoleStore((s) => s.fetchMetrics);
  const trends = useConsoleStore((s) => s.trends);
  const fetchTrends = useConsoleStore((s) => s.fetchTrends);
  const canOpen = usePlatformCan("internal-read");
  const [failed, setFailed] = useState(false);
  const [period, setPeriod] = useState<DashboardPeriod>("month");

  const load = useCallback(() => {
    setFailed(false);
    fetchMetrics().catch(() => setFailed(true));
  }, [fetchMetrics]);

  useEffect(() => {
    load();
  }, [load]);

  // The charts are bucketed by the API, one small request per period.
  useEffect(() => {
    void fetchTrends(period).catch(() => undefined);
  }, [fetchTrends, period]);

  const periodNoun = period === "month" ? "hari" : "bulan";
  const periodLabel = period === "month" ? "Bulan Ini" : "Tahun Ini";
  // Keep showing the previous period's bars until the new ones arrive.
  const revenuePoints = trends?.revenue ?? [];
  const tenantGrowthPoints = trends?.newTenants ?? [];

  if (failed && !metrics) return <LoadError onRetry={load} />;
  if (!metrics) return <div className="py-20 text-center text-sm text-text-secondary">Memuat ringkasan...</div>;

  const revenueChange = change(metrics.revenue30d, metrics.revenuePrev30d);
  const statusRow = [
    { key: "trialing", label: TENANT_STATUS_LABEL.trialing, value: metrics.trialing },
    { key: "active", label: TENANT_STATUS_LABEL.active, value: metrics.active },
    { key: "past_due", label: TENANT_STATUS_LABEL.past_due, value: metrics.pastDue },
    { key: "read_only", label: TENANT_STATUS_LABEL.read_only, value: metrics.readOnly },
    { key: "suspended", label: TENANT_STATUS_LABEL.suspended, value: metrics.suspended },
  ];

  return (
    <div className="flex flex-col gap-6">
      <div>
        <h1 className="text-xl font-bold text-text-primary">Ringkasan</h1>
        <p className="mt-1 text-[13px] text-text-secondary">Kondisi bisnis ElProof: pendapatan berulang, trial yang jadi berlangganan, dan tenant yang perlu ditindaklanjuti.</p>
      </div>

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <StatCard
          label="Pendapatan bulanan (MRR)"
          value={formatCurrencyCompact(metrics.mrr)}
          tone="info"
          sublabel={metrics.paidTenants > 0 ? `${metrics.paidTenants} tenant berbayar, rata-rata ${formatCurrencyCompact(metrics.arpa)}` : "Belum ada tenant berbayar"}
        />
        <StatCard
          label="Trial jadi berlangganan"
          value={pct(metrics.trialConversionPct)}
          tone={metrics.trialConversionPct !== null && metrics.trialConversionPct < 20 ? "warning" : "success"}
          sublabel={metrics.trialsEnded30d > 0 ? `${metrics.trialsConverted30d} dari ${metrics.trialsEnded30d} trial yang berakhir, 30 hari` : "Belum ada trial berakhir dalam 30 hari"}
        />
        <StatCard
          label="Tenant berhenti (churn)"
          value={pct(metrics.churnPct)}
          tone={metrics.churned30d > 0 ? "danger" : "success"}
          sublabel={metrics.churned30d > 0 ? `${metrics.churned30d} tenant, ${formatCurrencyCompact(metrics.churnedMrr)}/bulan hilang, 30 hari` : "Tidak ada yang berhenti dalam 30 hari"}
        />
        <StatCard
          label="Tagihan tertunda"
          value={formatCurrencyCompact(metrics.overdueValue)}
          tone={metrics.overdueValue > 0 ? "warning" : "success"}
          sublabel="Lihat tagihan"
          to={ROUTE_PATHS.platformTransactions}
        />
      </div>

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
        <Card className="lg:col-span-2">
          <CardHeader title="Pendapatan 30 hari terakhir" subtitle="Tagihan yang dibayar lewat sistem pembayaran. Pembayaran di luar sistem dan aktivasi gratis dari tim ikut MRR bila berbayar, tetapi tidak masuk angka ini." />
          <CardContent className="flex flex-wrap items-end gap-x-10 gap-y-3">
            <div>
              <p className="text-[28px] font-bold leading-none tabular-nums text-text-primary">{formatCurrency(metrics.revenue30d)}</p>
              <p className="mt-2 text-[12.5px] text-text-secondary">
                {revenueChange
                  ? `${revenueChange} dibanding 30 hari sebelumnya (${formatCurrency(metrics.revenuePrev30d)})`
                  : metrics.revenue30d > 0
                    ? "30 hari sebelumnya belum ada pendapatan sebagai pembanding"
                    : "Belum ada pembayaran masuk dalam 30 hari"}
              </p>
            </div>
            <div>
              <p className="text-lg font-semibold tabular-nums text-text-primary">{metrics.newPaid30d} tenant baru berbayar</p>
              <p className="mt-1 text-[12.5px] text-text-secondary">menambah {formatCurrencyCompact(metrics.newMrr30d)}/bulan</p>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader title="Tenant per status" subtitle={`${metrics.totalTenants} tenant terdaftar`} />
          <ul className="divide-y divide-border-light">
            {statusRow.map((r) => (
              <li key={r.key}>
                <Link to={`${ROUTE_PATHS.platformTenants}?status=${r.key}`} className="flex items-center justify-between px-5 py-2.5 text-[13.5px] transition-colors hover:bg-surface-muted">
                  <span className="text-text-secondary">{r.label}</span>
                  <span className="font-semibold tabular-nums text-text-primary">{r.value}</span>
                </Link>
              </li>
            ))}
          </ul>
        </Card>
      </div>

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
        <Card>
          <CardHeader title="Trial segera berakhir" subtitle="Hubungi sebelum trial habis, ini peluang konversi." />
          <CardContent className="p-0">
            <AttentionList
              canOpen={canOpen}
              items={metrics.trialsEndingSoon}
              tone="warning"
              empty={{ title: "Tidak ada trial yang segera berakhir", description: "Belum ada trial yang berakhir dalam waktu dekat." }}
              describe={(a) => (a.daysLeft <= 0 ? "Trial berakhir hari ini" : `Trial berakhir dalam ${a.daysLeft} hari`)}
            />
          </CardContent>
        </Card>
        <Card>
          <CardHeader title="Menunggak" subtitle="Langganan berakhir dan belum diperpanjang." />
          <CardContent className="p-0">
            <AttentionList
              canOpen={canOpen}
              items={metrics.overdueTenants}
              tone="danger"
              empty={{ title: "Tidak ada tunggakan", description: "Semua langganan yang berakhir sudah diperpanjang atau masih berjalan." }}
              describe={(a) => `${TENANT_STATUS_LABEL[a.status]}, berakhir ${Math.abs(a.daysLeft)} hari lalu`}
            />
          </CardContent>
        </Card>
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
          formatValue={formatCurrency}
          valueHeader="Omzet"
          periodHeader={periodNoun === "hari" ? "Hari" : "Bulan"}
        />
        <TrendBarChart
          title={`Tenant Baru ${periodLabel}`}
          subtitle={`Jumlah tenant baru bergabung, per ${periodNoun}.`}
          points={tenantGrowthPoints}
          totalLabel={`tenant baru ${periodLabel.toLowerCase()}`}
          formatTotal={(value) => String(value)}
          formatBarLabel={(value) => String(value)}
          formatValue={(value) => `${value} tenant`}
          valueHeader="Tenant baru"
          periodHeader={periodNoun === "hari" ? "Hari" : "Bulan"}
        />
      </div>
    </div>
  );
}
