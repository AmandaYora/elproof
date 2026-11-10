import { useCallback, useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { FolderKanban } from "lucide-react";
import { Card, CardContent } from "@/shared/components/ui/Card";
import { EmptyState } from "@/shared/components/feedback/EmptyState";
import { LoadError } from "@/shared/components/feedback/LoadError";
import { SkeletonRows } from "@/shared/components/ui/Skeleton";
import { ProgressMeter } from "@/shared/components/ui/ProgressMeter";
import { ProjectStatusBadge } from "@/modules/projects/components/StatusBadges";
import { httpClient } from "@/shared/services/http-client";
import { API } from "@/shared/services/api-endpoints";
import { useAuthStore } from "@/shared/stores/useAuthStore";
import { daysUntil } from "@/modules/projects/lib/dates";
import { formatCurrency, formatDate } from "@/shared/lib/formatters";
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

  return (
    <div className="flex flex-col gap-6">
      <div>
        <h1 className="text-xl font-bold text-text-primary">Proyek saya{name ? `, ${name.split(" ")[0]}` : ""}</h1>
        <p className="mt-1 text-[13px] text-text-secondary">Proyek yang Anda pegang, dengan yang perlu Anda tangani lebih dulu.</p>
      </div>

      {failed ? (
        <LoadError message="Proyek Anda belum bisa dimuat. Periksa koneksi, lalu coba lagi." onRetry={load} />
      ) : projects === null ? (
        <div className="rounded-lg border border-border bg-surface p-5"><SkeletonRows rows={3} label="Memuat proyek saya" /></div>
      ) : projects.length === 0 ? (
        <Card>
          <EmptyState
            icon={<FolderKanban className="h-7 w-7" />}
            title="Belum ada proyek yang Anda pegang"
            description="Proyek muncul di sini setelah Owner menugaskan Anda sebagai penanggung jawab."
          />
        </Card>
      ) : (
        <ul className="grid grid-cols-1 gap-4 lg:grid-cols-2">
          {projects.map((p) => {
            const days = daysUntil(p.eventDate);
            return (
              <li key={p.id}>
                <Card>
                  <CardContent className="flex flex-col gap-4 py-5">
                    <div className="flex items-start justify-between gap-3">
                      <div className="min-w-0">
                        <Link
                          to={ROUTE_PATHS.projectDetail(String(p.id))}
                          className="block truncate text-[15px] font-semibold text-text-primary underline-offset-2 hover:underline focus-visible:outline focus-visible:outline-2 focus-visible:outline-ring"
                        >
                          {p.name}
                        </Link>
                        <p className="text-[12.5px] text-text-secondary">
                          {p.brideName} &amp; {p.groomName} · {formatDate(p.eventDate)}
                        </p>
                      </div>
                      <div className="flex shrink-0 flex-col items-end gap-1.5">
                        <span className="text-[22px] font-extrabold leading-none tracking-[-0.02em] tabular-nums text-primary">{days >= 0 ? `H-${days}` : `H+${Math.abs(days)}`}</span>
                        <ProjectStatusBadge status={p.status} />
                      </div>
                    </div>

                    <ProgressMeter percent={p.progressPercent} segments={[]} caption={`${p.progressPercent}% tahapan selesai`} />

                    {p.overdueTerms > 0 && (
                      <p className="rounded-md bg-danger-soft px-3 py-2 text-[13px] font-medium text-danger-text">
                        {p.overdueTerms} termin klien terlambat · {formatCurrency(p.overdueTotal)}
                      </p>
                    )}
                    {p.overdueTerms === 0 && p.nextTermLabel && p.nextTermDueDate && (
                      <p className="text-[13px] text-text-secondary">
                        Termin berikutnya: {p.nextTermLabel}, {formatCurrency(p.nextTermRemaining ?? 0)}, jatuh tempo {formatDate(p.nextTermDueDate)}
                      </p>
                    )}
                    {p.openIssues > 0 && (
                      <Link to={ROUTE_PATHS.projectDetail(String(p.id), "kendala")} className="text-[13px] font-medium text-warning-strong underline-offset-2 hover:underline">
                        {p.openIssues} kendala masih terbuka
                      </Link>
                    )}
                    {p.tasksDueThisWeek.length > 0 && (
                      <div>
                        <p className="text-[12px] font-medium text-text-secondary">Jatuh tempo minggu ini</p>
                        <ul className="mt-1 flex flex-col gap-0.5">
                          {p.tasksDueThisWeek.slice(0, 3).map((t, i) => (
                            <li key={i} className="flex justify-between gap-3 text-[13px] text-text-primary">
                              <span className="truncate">{t.name}</span>
                              <span className="shrink-0 text-text-secondary">{daysUntil(t.targetDate) === 0 ? "Hari ini" : formatDate(t.targetDate)}</span>
                            </li>
                          ))}
                        </ul>
                      </div>
                    )}
                  </CardContent>
                </Card>
              </li>
            );
          })}
        </ul>
      )}
    </div>
  );
}
