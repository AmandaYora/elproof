import { useEffect, useMemo, useState } from "react";
import { Link, useLocation } from "react-router-dom";
import { usePermissionStore } from "@/shared/stores/usePermissionStore";
import { useAuthStore } from "@/shared/stores/useAuthStore";
import { AttentionQueue } from "@/modules/dashboard/components/AttentionQueue";
import { OverdueHero } from "@/modules/dashboard/components/OverdueHero";
import { DueList, type DueListRow } from "@/modules/dashboard/components/DueList";
import { RecentActivity } from "@/modules/dashboard/components/RecentActivity";
import { ProjectTrendChart } from "@/modules/dashboard/components/ProjectTrendChart";
import { CashFlowCard } from "@/modules/dashboard/components/CashFlowCard";
import { RevenueTrendChart } from "@/modules/dashboard/components/RevenueTrendChart";
import { buildAttentionItems, type AttentionItem } from "@/modules/dashboard/lib/attention";
import { useDashboardStore } from "@/modules/dashboard/stores/useDashboardStore";
import { daysUntil } from "@/modules/projects/lib/dates";
import { formatCurrency, formatDate } from "@/shared/lib/formatters";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import { LoadError } from "@/shared/components/feedback/LoadError";
import { SkeletonRows } from "@/shared/components/ui/Skeleton";
import MyProjectsPage from "@/modules/dashboard/pages/MyProjectsPage";
import { OnboardingChecklist } from "@/modules/onboarding/components/OnboardingChecklist";

function greeting(): string {
  const h = new Date().getHours();
  if (h < 11) return "Selamat pagi";
  if (h < 15) return "Selamat siang";
  if (h < 18) return "Selamat sore";
  return "Selamat malam";
}

const TODAY_LABEL = () =>
  new Date().toLocaleDateString("id-ID", { weekday: "long", day: "numeric", month: "long", year: "numeric" });

function FullDashboard() {
  const stats = useDashboardStore((s) => s.stats);
  const fetchDashboard = useDashboardStore((s) => s.fetchDashboard);
  const displayName = useAuthStore((s) => s.session?.displayName ?? "");
  // The "Mulai di sini" checklist belongs to whoever runs the workspace.
  const isOwner = usePermissionStore((s) => Boolean(s.grants?.isOwner));
  const welcome = Boolean((useLocation().state as { welcome?: boolean } | null)?.welcome);

  const [loadFailed, setLoadFailed] = useState(false);
  const [attempt, setAttempt] = useState(0);

  useEffect(() => {
    setLoadFailed(false);
    fetchDashboard().catch(() => setLoadFailed(true));
  }, [fetchDashboard, attempt]);

  const { items, counts } = useMemo(
    () => (stats ? buildAttentionItems(stats) : { items: [] as AttentionItem[], counts: {} as Record<string, number> }),
    [stats],
  );

  const events = useMemo<DueListRow[]>(
    () =>
      (stats?.upcomingProjects ?? [])
        .filter((p) => daysUntil(p.eventDate) <= 14)
        .map((p) => ({
          id: p.id,
          to: ROUTE_PATHS.projectDetail(p.id),
          title: p.name,
          subtitle: `${formatDate(p.eventDate)} · ${p.venue}`,
          trailing: `H-${daysUntil(p.eventDate)}`,
        })),
    [stats],
  );
  const tasks = useMemo<DueListRow[]>(
    () =>
      (stats?.tasksDueThisWeek ?? []).slice(0, 6).map((t, i) => ({
        id: `${t.projectId}-${i}`,
        to: ROUTE_PATHS.projectDetail(t.projectId, "milestone"),
        title: t.name,
        subtitle: t.projectName,
        trailing: daysUntil(t.targetDate) === 0 ? "Hari ini" : formatDate(t.targetDate),
      })),
    [stats],
  );
  const payments = useMemo<DueListRow[]>(
    () =>
      (stats?.dueSoonTerms ?? []).slice(0, 6).map((t, i) => ({
        id: `${t.projectId}-${i}`,
        to: ROUTE_PATHS.projectDetail(t.projectId, "pembayaran/client"),
        title: `${t.projectName} · ${t.label}`,
        subtitle: `Jatuh tempo ${formatDate(t.dueDate)}`,
        trailing: formatCurrency(t.remaining),
      })),
    [stats],
  );

  const vendorDue = useMemo<DueListRow[]>(
    () =>
      (stats?.vendorPaymentsDue ?? []).slice(0, 6).map((v, i) => ({
        id: `${v.projectId}-${i}`,
        to: ROUTE_PATHS.projectDetail(v.projectId, "pembayaran/vendor"),
        title: `${v.projectName} · ${v.vendorName}`,
        subtitle: daysUntil(v.dueDate) < 0 ? `Terlambat ${Math.abs(daysUntil(v.dueDate))} hari` : `Jatuh tempo ${formatDate(v.dueDate)}`,
        trailing: formatCurrency(v.remaining),
      })),
    [stats],
  );

  if (!stats) {
    return loadFailed ? (
      <LoadError message="Beranda belum bisa dimuat. Periksa koneksi Anda, lalu coba lagi." onRetry={() => setAttempt((n) => n + 1)} />
    ) : (
      <div className="rounded-lg border border-border bg-surface p-5"><SkeletonRows rows={5} label="Memuat beranda" /></div>
    );
  }

  const first = displayName.split(" ")[0];
  const stage = stats.stageCounts;

  return (
    <div className="flex flex-col gap-6">
      <div>
        <h1 className="text-xl font-bold text-text-primary">
          {greeting()}
          {first ? `, ${first}` : ""}
        </h1>
        <p className="mt-1 text-[13px] text-text-secondary">{TODAY_LABEL()}</p>
      </div>

      {/* Money that is late comes first; with none, the setup guide leads. */}
      {stats.overdueTermsTotal > 0 && <OverdueHero total={stats.overdueTermsTotal} terms={stats.overdueTerms} />}
      {isOwner && <OnboardingChecklist welcome={welcome} />}
      {stats.overdueTermsTotal === 0 && <OverdueHero total={0} terms={[]} />}

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
        <div className="lg:col-span-2">
          <AttentionQueue items={items} counts={counts} />
        </div>
        <div className="flex flex-col gap-6">
          <DueList title="Acara 14 hari ke depan" subtitle="Hitung mundur per proyek." rows={events} empty="Tidak ada acara dalam 14 hari ke depan." />
          <DueList title="Tugas minggu ini" subtitle="Tahapan proyek yang jatuh tempo 7 hari ke depan." rows={tasks} empty="Tidak ada tahapan yang jatuh tempo minggu ini." />
        </div>
      </div>

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
        <div className="lg:col-span-2">
          <DueList title="Pembayaran masuk 30 hari ke depan" subtitle="Termin klien yang belum lunas." rows={payments} empty="Belum ada termin jatuh tempo dalam 30 hari. Atur jadwal termin di tab Keuangan proyek." />
        </div>
        <section aria-labelledby="stage-heading" className="rounded-lg border border-border bg-surface px-5 py-4">
          <h2 id="stage-heading" className="text-[15px] font-semibold text-text-primary">Proyek per tahap</h2>
          <dl className="mt-3 grid grid-cols-3 gap-2 text-center">
            {[
              ["Draf", stage.draft],
              ["Persiapan", stage.preparation],
              ["Siap", stage.ready],
            ].map(([label, n]) => (
              <div key={label as string} className="rounded-md bg-surface-muted px-2 py-3">
                <dd className="text-xl font-bold tabular-nums text-navy-900">{n}</dd>
                <dt className="text-[12px] text-text-secondary">{label}</dt>
              </div>
            ))}
          </dl>
          <Link to={ROUTE_PATHS.projects} className="mt-3 inline-block text-[13px] font-medium text-navy-900 underline-offset-2 hover:underline">
            Lihat semua proyek
          </Link>
        </section>
      </div>

      {stats.cashFlow && (
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
          <div className="lg:col-span-2">
            <CashFlowCard weeks={stats.cashFlow} />
          </div>
          <DueList title="Tagihan vendor 30 hari ke depan" subtitle="Termasuk yang sudah lewat." rows={vendorDue} empty="Tidak ada tagihan vendor yang jatuh tempo dalam 30 hari." />
        </div>
      )}

      <RevenueTrendChart points={stats.revenueTrend} />
      <ProjectTrendChart points={stats.projectTrend} />

      <RecentActivity activity={stats.recentActivity} />
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
