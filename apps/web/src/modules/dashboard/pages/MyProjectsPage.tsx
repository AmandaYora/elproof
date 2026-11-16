import { useCallback, useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { AlertCircle, AlertTriangle, FolderKanban } from "lucide-react";
import { Card } from "@/shared/components/ui/Card";
import { EmptyState } from "@/shared/components/feedback/EmptyState";
import { LoadError } from "@/shared/components/feedback/LoadError";
import { PageHeader } from "@/shared/components/ui/PageHeader";
import { StatCard } from "@/shared/components/ui/StatCard";
import { SkeletonRows } from "@/shared/components/ui/Skeleton";
import { ProjectStatusBadge } from "@/modules/projects/components/StatusBadges";
import { KpiMoney } from "@/modules/dashboard/components/KpiMoney";
import { greeting } from "@/modules/dashboard/lib/greeting";
import { httpClient } from "@/shared/services/http-client";
import { API } from "@/shared/services/api-endpoints";
import { useAuthStore } from "@/shared/stores/useAuthStore";
import { daysUntil } from "@/modules/projects/lib/dates";
import { formatCurrency, formatDate, formatDayCountdown } from "@/shared/lib/formatters";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import type { ProjectStatus } from "@/modules/projects/types";

interface MyProject {
  id: number;
  name: string;
  brideName: string;
  groomName: string;
  eventDate: string;
  status: ProjectStatus;
  progressPercent: number;
  openIssues: number;
  overdueTerms: number;
  overdueTotal: number;
  nextTermLabel?: string;
  nextTermDueDate?: string;
  nextTermRemaining?: number;
  tasksDueThisWeek: { name: string; targetDate: string }[];
}

function MyKpis({ projects }: { projects: MyProject[] }) {
  const overdueTerms = projects.reduce((n, p) => n + p.overdueTerms, 0);
  const overdueTotal = projects.reduce((n, p) => n + p.overdueTotal, 0);
  const overdueProjects = projects.filter((p) => p.overdueTerms > 0).length;
  const events = projects.filter((p) => {
    const d = daysUntil(p.eventDate);
    return d >= 0 && d <= 30;
  });
  const issues = projects.reduce((n, p) => n + p.openIssues, 0);
  return (
    <section aria-label="Ringkasan angka" className="grid grid-cols-2 gap-3 sm:gap-4 xl:grid-cols-4">
      <StatCard label="Proyek yang Anda pegang" value={projects.length} sublabel="Belum selesai, acara terdekat di atas" />
      <StatCard
        label="Acara 30 hari ke depan"
        value={events.length}
        sublabel={events[0] ? `Terdekat ${formatDayCountdown(daysUntil(events[0].eventDate))}, ${events[0].name}` : "Tidak ada acara dalam 30 hari"}
        to={ROUTE_PATHS.calendar}
      />
      <StatCard
        label="Tagihan terlambat"
        value={<KpiMoney amount={overdueTotal} />}
        tone={overdueTotal > 0 ? "danger" : "neutral"}
        sublabel={overdueTerms > 0 ? `${overdueTerms} termin di ${overdueProjects} proyek` : "Tidak ada termin lewat jatuh tempo"}
      />
      <StatCard label="Kendala terbuka" value={issues} sublabel={issues > 0 ? "Di proyek yang Anda pegang" : "Tidak ada kendala terbuka"} />
    </section>
  );
}

function ProjectCard({ p }: { p: MyProject }) {
  const days = daysUntil(p.eventDate);
  const progress = Math.min(100, Math.max(0, p.progressPercent));
  return (
    <Card className="relative flex h-full flex-col gap-3 px-5 py-4 hover:border-primary/40">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <Link
            to={ROUTE_PATHS.projectDetail(String(p.id))}
            className="block truncate text-[16px] font-semibold text-foreground outline-none after:absolute after:inset-0 after:rounded-[inherit] after:content-[''] focus-visible:after:outline-2 focus-visible:after:outline-offset-2 focus-visible:after:outline-solid focus-visible:after:outline-ring"
          >
            {p.name}
          </Link>
          <p className="truncate text-[13px] text-muted-foreground">
            {formatDate(p.eventDate)} · <span className="font-semibold text-highlight-text">{formatDayCountdown(days)}</span>
          </p>
        </div>
        <ProjectStatusBadge status={p.status} />
      </div>

      <div className="flex items-center gap-3">
        <div aria-hidden className="h-1.5 flex-1 overflow-hidden rounded-full bg-muted">
          <div className="h-full rounded-full bg-primary" style={{ width: `${progress}%` }} />
        </div>
        <span className="shrink-0 text-[12px] tabular-nums text-muted-foreground">{progress}% tahapan selesai</span>
      </div>

      <ul className="flex flex-col gap-1 text-[13px]">
        {p.overdueTerms > 0 ? (
          <li className="flex items-center gap-2 font-medium text-danger-text">
            <AlertTriangle aria-hidden className="size-4 shrink-0" />
            {p.overdueTerms} termin klien terlambat · {formatCurrency(p.overdueTotal)}
          </li>
        ) : (
          p.nextTermLabel &&
          p.nextTermDueDate && (
            <li className="truncate text-muted-foreground">
              Termin berikutnya: {p.nextTermLabel}, {formatCurrency(p.nextTermRemaining ?? 0)} pada {formatDate(p.nextTermDueDate)}
            </li>
          )
        )}
        {p.openIssues > 0 && (
          <li className="flex items-center gap-2 font-medium text-warning-text">
            <AlertCircle aria-hidden className="size-4 shrink-0" />
            {p.openIssues} kendala masih terbuka
          </li>
        )}
        {p.tasksDueThisWeek.slice(0, 2).map((t, i) => (
          <li key={i} className="flex justify-between gap-3 text-foreground">
            <span className="truncate">{t.name}</span>
            <span className="shrink-0 text-muted-foreground">{daysUntil(t.targetDate) === 0 ? "Hari ini" : formatDate(t.targetDate).replace(/ \d{4}$/, "")}</span>
          </li>
        ))}
        {p.tasksDueThisWeek.length > 2 && <li className="text-muted-foreground">dan {p.tasksDueThisWeek.length - 2} tahapan lain minggu ini</li>}
      </ul>
    </Card>
  );
}

/**
 * The landing page for a role limited to its own projects (Wedding Planner,
 * Sales): the projects they hold, each with what needs them now. The tenant-wide
 * dashboard is not theirs (it adds up every project).
 */
export default function MyProjectsPage() {
  const name = useAuthStore((s) => s.session?.displayName ?? "");
  const [projects, setProjects] = useState<MyProject[] | null>(null);
  const [failed, setFailed] = useState(false);

  const load = useCallback(() => {
    setFailed(false);
    httpClient
      .get(API.dashboardMine)
      .then((res) => setProjects(res.data.data as MyProject[]))
      .catch(() => setFailed(true));
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  const first = name.split(" ")[0];

  return (
    <div>
      <PageHeader title="Proyek saya" description={`${greeting()}${first ? `, ${first}` : ""}. Proyek yang Anda pegang, acara terdekat lebih dulu.`} />

      {failed ? (
        <LoadError message="Proyek Anda belum bisa dimuat. Periksa koneksi, lalu coba lagi." onRetry={load} />
      ) : projects === null ? (
        <div className="rounded-lg border border-border bg-surface p-5">
          <SkeletonRows rows={3} label="Memuat proyek saya" />
        </div>
      ) : projects.length === 0 ? (
        <Card>
          <EmptyState
            icon={<FolderKanban className="h-7 w-7" />}
            title="Belum ada proyek yang Anda pegang"
            description="Proyek muncul di sini setelah Owner menugaskan Anda sebagai penanggung jawab."
          />
        </Card>
      ) : (
        <div className="flex flex-col gap-6">
          <MyKpis projects={projects} />
          <section aria-labelledby="my-projects-heading">
            <h2 id="my-projects-heading" className="mb-3 text-[16px] font-semibold text-foreground">
              Daftar proyek
            </h2>
            <ul className="grid grid-cols-1 gap-4 lg:grid-cols-2 2xl:grid-cols-3">
              {projects.map((p) => (
                <li key={p.id}>
                  <ProjectCard p={p} />
                </li>
              ))}
            </ul>
          </section>
        </div>
      )}
    </div>
  );
}
