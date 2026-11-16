import { useEffect, useMemo, useState } from "react";
import { useLocation } from "react-router-dom";
import { usePermissionStore, hasPermission } from "@/shared/stores/usePermissionStore";
import { useAuthStore } from "@/shared/stores/useAuthStore";
import { AttentionQueue } from "@/modules/dashboard/components/AttentionQueue";
import { CashFlowChart } from "@/modules/dashboard/components/CashFlowChart";
import { DueList, type DueListRow } from "@/modules/dashboard/components/DueList";
import { RecentActivity } from "@/modules/dashboard/components/RecentActivity";
import { ProjectTrendChart } from "@/modules/dashboard/components/ProjectTrendChart";
import { RevenueTrendChart } from "@/modules/dashboard/components/RevenueTrendChart";
import { UpcomingEvents } from "@/modules/dashboard/components/UpcomingEvents";
import { buildAttentionItems } from "@/modules/dashboard/lib/attention";
import { greeting, todayLong } from "@/modules/dashboard/lib/greeting";
import { useDashboardStore } from "@/modules/dashboard/stores/useDashboardStore";
import type { DashboardStats } from "@/modules/dashboard/types";
import { daysUntil } from "@/modules/projects/lib/dates";
import { KpiMoney } from "@/modules/dashboard/components/KpiMoney";
import { formatDate, formatDayCountdown } from "@/shared/lib/formatters";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import { LoadError } from "@/shared/components/feedback/LoadError";
import { PageHeader } from "@/shared/components/ui/PageHeader";
import { StatCard } from "@/shared/components/ui/StatCard";
import { SkeletonRows } from "@/shared/components/ui/Skeleton";
import MyProjectsPage from "@/modules/dashboard/pages/MyProjectsPage";
import { OnboardingChecklist } from "@/modules/onboarding/components/OnboardingChecklist";

/**
 * A 12-month trend is shown only once at least three months have something in them: a new workspace would otherwise
 * see a row of zero bars that reads as broken (RISET WO Console §7.2). Until then the cash flow is the page's chart.
 */
const MIN_TREND_MONTHS = 3;
const monthsWithData = (values: number[]) => values.filter((v) => v > 0).length;

function Kpis({ stats, canFinance }: { stats: DashboardStats; canFinance: boolean }) {
  const overdueProjects = new Set(stats.overdueTerms.map((t) => t.projectId)).size;
  const overdue = stats.overdueTermsTotal;
  const dueSoon = stats.dueSoonTerms.reduce((n, t) => n + t.remaining, 0);
  const events = stats.nearDDayProjects.filter((p) => daysUntil(p.eventDate) >= 0).sort((a, b) => a.eventDate.localeCompare(b.eventDate));
  const { draft, preparation, ready } = stats.stageCounts;
  const active = preparation + ready;
  // Keuangan needs finance.record; without it the late-installment card opens the first late project instead.
  const overdueTo = canFinance ? ROUTE_PATHS.finance : stats.overdueTerms[0] ? ROUTE_PATHS.projectDetail(stats.overdueTerms[0].projectId, "keuangan/klien") : undefined;

  return (
    <section aria-label="Ringkasan angka" className="grid grid-cols-2 gap-3 sm:gap-4 xl:grid-cols-4">
      <StatCard
        label="Tagihan terlambat"
        value={<KpiMoney amount={overdue} />}
        tone={overdue > 0 ? "danger" : "neutral"}
        sublabel={overdue > 0 ? `${stats.overdueTerms.length} termin di ${overdueProjects} proyek` : "Tidak ada termin lewat jatuh tempo"}
        to={overdueTo}
      />
      <StatCard
        label="Tagihan jatuh tempo 30 hari"
        value={<KpiMoney amount={dueSoon} />}
        sublabel={stats.dueSoonTerms.length > 0 ? `${stats.dueSoonTerms.length} termin klien belum lunas` : "Belum ada termin yang jatuh tempo"}
        to={canFinance ? ROUTE_PATHS.finance : undefined}
      />
      <StatCard
        label="Acara 30 hari ke depan"
        value={events.length}
        sublabel={events[0] ? `Terdekat ${formatDayCountdown(daysUntil(events[0].eventDate))}, ${events[0].name}` : "Tidak ada acara dalam 30 hari"}
        to={ROUTE_PATHS.calendar}
      />
      <StatCard
        label="Proyek aktif"
        value={active}
        sublabel={active + draft > 0 ? `${preparation} persiapan, ${ready} siap · ${draft} draf` : "Belum ada proyek berjalan"}
        to={ROUTE_PATHS.projects}
      />
    </section>
  );
}

function FullDashboard() {
  const stats = useDashboardStore((s) => s.stats);
  const fetchDashboard = useDashboardStore((s) => s.fetchDashboard);
  const displayName = useAuthStore((s) => s.session?.displayName ?? "");
  const grants = usePermissionStore((s) => s.grants);
  // The "Mulai di sini" checklist belongs to whoever runs the workspace.
  const isOwner = Boolean(grants?.isOwner);
  const canFinance = hasPermission(grants, "finance.record");
  const canCreate = hasPermission(grants, "project.create");
  const welcome = Boolean((useLocation().state as { welcome?: boolean } | null)?.welcome);

  const [loadFailed, setLoadFailed] = useState(false);
  const [attempt, setAttempt] = useState(0);

  useEffect(() => {
    setLoadFailed(false);
    fetchDashboard().catch(() => setLoadFailed(true));
  }, [fetchDashboard, attempt]);

  const attention = useMemo(() => (stats ? buildAttentionItems(stats) : null), [stats]);

  const tasks = useMemo<DueListRow[]>(
    () =>
      (stats?.tasksDueThisWeek ?? []).slice(0, 5).map((t, i) => ({
        id: `${t.projectId}-${i}`,
        to: ROUTE_PATHS.projectDetail(t.projectId, "jadwal"),
        title: t.name,
        subtitle: t.projectName,
        trailing: daysUntil(t.targetDate) === 0 ? "Hari ini" : formatDate(t.targetDate).replace(/ \d{4}$/, ""),
      })),
    [stats],
  );

  const first = displayName.split(" ")[0];
  const header = <PageHeader title={`${greeting()}${first ? `, ${first}` : ""}`} description={todayLong()} />;

  if (!stats || !attention) {
    return (
      <div>
        {header}
        {loadFailed ? (
          <LoadError message="Beranda belum bisa dimuat. Periksa koneksi Anda, lalu coba lagi." onRetry={() => setAttempt((n) => n + 1)} />
        ) : (
          <div className="rounded-lg border border-border bg-surface p-5">
            <SkeletonRows rows={5} label="Memuat beranda" />
          </div>
        )}
      </div>
    );
  }

  const showRevenue = monthsWithData(stats.revenueTrend.map((p) => p.total)) >= MIN_TREND_MONTHS;
  const showProjects = monthsWithData(stats.projectTrend.map((p) => p.count)) >= MIN_TREND_MONTHS;
  const emptyAction = stats.totalProjects === 0 && canCreate ? { label: "Buat proyek", to: ROUTE_PATHS.projectNew } : { label: "Buka daftar proyek", to: ROUTE_PATHS.projects };

  return (
    <div>
      {header}
      <div className="flex flex-col gap-6">
        {isOwner && <OnboardingChecklist welcome={welcome} />}

        <Kpis stats={stats} canFinance={canFinance} />

        {stats.cashFlow ? (
          <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
            <div className="lg:col-span-2">
              <CashFlowChart weeks={stats.cashFlow} emptyAction={emptyAction} />
            </div>
            {/* The chart's empty state already offers the next step. */}
            <UpcomingEvents projects={stats.upcomingProjects} canCreate={false} />
          </div>
        ) : null}

        {/* Phone order: what needs doing, this week's steps, then history. Desktop: the side column spans both rows. */}
        <div className="grid grid-cols-1 items-start gap-6 lg:grid-cols-3">
          <div className="lg:col-span-2">
            <AttentionQueue items={attention.items} counts={attention.counts} hasProjects={stats.totalProjects > 0} />
          </div>
          <div className="flex flex-col gap-6 lg:col-start-3 lg:row-span-2 lg:row-start-1">
            {/* Without money figures there is no cash flow; the events list moves here. */}
            {!stats.cashFlow && <UpcomingEvents projects={stats.upcomingProjects} canCreate={canCreate} />}
            <DueList title="Tahapan 7 hari ke depan" rows={tasks} empty="Tidak ada tahapan proyek yang jatuh tempo minggu ini." />
          </div>
          <div className="lg:col-span-2">
            <RecentActivity activity={stats.recentActivity} />
          </div>
        </div>

        {/* Twelve narrow bars do not fit a phone; the trends are a desk-side look back. */}
        {(showRevenue || showProjects) && (
          <div className="hidden grid-cols-1 gap-6 md:grid lg:grid-cols-2">
            {showRevenue && <RevenueTrendChart points={stats.revenueTrend} />}
            {showProjects && <ProjectTrendChart points={stats.projectTrend} />}
          </div>
        )}
      </div>
    </div>
  );
}

/**
 * Beranda: the tenant-wide dashboard for roles that may see every project's
 * numbers; everyone else lands on "Proyek saya".
 */
export default function DashboardPage() {
  const grants = usePermissionStore((s) => s.grants);
  const seesEverything = grants?.permissions.includes("dashboard.view") === true && grants.projectScope === "all";
  return seesEverything ? <FullDashboard /> : <MyProjectsPage />;
}
