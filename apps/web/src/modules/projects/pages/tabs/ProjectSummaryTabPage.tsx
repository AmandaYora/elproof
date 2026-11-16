import { lazy, Suspense, useEffect, useState, type ReactNode } from "react";
import { Link, useOutletContext } from "react-router-dom";
import { CircleCheck, UserPlus } from "lucide-react";
import { Card, CardHeader, CardContent } from "@/shared/components/ui/Card";
import { Badge } from "@/shared/components/ui/Badge";
import { Button } from "@/shared/components/ui/Button";
import { Drawer } from "@/shared/components/ui/Drawer";
import { StatCard } from "@/shared/components/ui/StatCard";
import { ProgressMeter, type ProgressMeterStatus } from "@/shared/components/ui/ProgressMeter";
import { SkeletonRows } from "@/shared/components/ui/Skeleton";
import { EmptyState } from "@/shared/components/feedback/EmptyState";
import { useCan } from "@/shared/stores/usePermissionStore";
import { ConditionBadge, MilestoneStatusBadge } from "@/modules/projects/components/StatusBadges";
import { ActivityList } from "@/modules/projects/components/detail/ActivityList";
import { ProjectCoverCard } from "@/modules/projects/components/detail/ProjectCoverCard";
import { useProjectStore } from "@/modules/projects/stores/useProjectStore";
import { TERM_STATUS_TONE } from "@/modules/projects/lib/status";
import type { ActivityLogEntry, ProjectProgress } from "@/modules/projects/types";
import { TERM_STATUS_LABELS } from "@/shared/lib/labels";
import { formatCurrency, formatDate, formatDaysFromToday, formatEventOffset } from "@/shared/lib/formatters";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import type { ProjectDetailContext } from "@/modules/projects/pages/ProjectDetailLayout";

// Opened from the two panels on the right; neither is needed until someone clicks.
const ProjectClientsSection = lazy(() => import("@/modules/projects/components/detail/ProjectClientsSection").then((m) => ({ default: m.ProjectClientsSection })));
const ProjectActivitySection = lazy(() => import("@/modules/projects/components/detail/ProjectActivitySection").then((m) => ({ default: m.ProjectActivitySection })));

const RECENT_ACTIVITY = 5;

const CLIENT_ROLE_LABEL: Record<string, string> = {
  Bride: "Pengantin wanita",
  Groom: "Pengantin pria",
  "Family Representative": "Perwakilan keluarga",
};

/** One notch per timeline item, done first, then in progress, blocked, not started (the counts come with the project). */
function progressSegments(progress: ProjectProgress): Array<{ status: ProgressMeterStatus }> {
  const keys = { Completed: "completed", "In Progress": "inProgress", Blocked: "blocked", "Not Started": "notStarted" } as const;
  return (["Completed", "In Progress", "Blocked", "Not Started"] as const).flatMap((status) =>
    Array.from({ length: progress.projectMilestoneStats[keys[status]] + progress.vendorMilestoneStats[keys[status]] }, () => ({ status })),
  );
}

/**
 * First tab of a project (RISET WO Console §7.4): three numbers that matter (contract, what the client still owes,
 * how far the timeline is), then what needs doing, what comes next, and on the right who the clients are and what
 * changed lately. Everything but the history comes from the overview (E1), computed by the server.
 */
export default function ProjectSummaryTabPage() {
  const { projectId } = useOutletContext<ProjectDetailContext>();
  const project = useProjectStore((s) => s.currentProject);
  const overview = useProjectStore((s) => s.overview);
  if (!overview || !project) return null;

  const { actions, nextMilestones, finance } = overview;
  const progress = project.progress;
  const nextTerm = finance.nextTerm;
  const hasTerms = finance.terms.length > 0;
  const overdue = finance.overdueTotal > 0;
  const total = (progress?.projectMilestoneStats.total ?? 0) + (progress?.vendorMilestoneStats.total ?? 0);
  const done = (progress?.projectMilestoneStats.completed ?? 0) + (progress?.vendorMilestoneStats.completed ?? 0);

  return (
    <div className="flex flex-col gap-5">
      <section aria-label="Angka utama" className="grid grid-cols-1 gap-3 sm:grid-cols-3">
        <StatCard
          label="Nilai kontrak"
          value={formatCurrency(finance.contractValue)}
          sublabel={finance.margin !== undefined ? `Margin ${formatCurrency(finance.margin)}` : `Sudah dibayar ${formatCurrency(finance.paid)}`}
        />
        <StatCard
          label="Sisa tagihan"
          value={formatCurrency(finance.remaining)}
          tone={overdue ? "warning" : "neutral"}
          to={ROUTE_PATHS.projectDetail(projectId, "keuangan/klien")}
          sublabel={
            overdue
              ? `${formatCurrency(finance.overdueTotal)} sudah lewat jatuh tempo`
              : finance.remaining === 0 && hasTerms
                ? "Lunas"
                : nextTerm
                  ? `${nextTerm.label} jatuh tempo ${formatDate(nextTerm.dueDate)}`
                  : "Lihat pembayaran klien"
          }
        />
        <div className="rounded-lg border border-border bg-surface px-5 py-4">
          <div className="flex items-center justify-between gap-2">
            <span className="text-[13px] font-medium text-text-secondary">Progres tahapan</span>
            {progress && <ConditionBadge condition={progress.condition} />}
          </div>
          <div className="mt-2.5 text-[28px] leading-none font-bold tracking-[-0.02em] tabular-nums text-text-primary">{progress?.overallPercent ?? 0}%</div>
          {progress && total > 0 ? (
            <ProgressMeter className="mt-3" percent={progress.overallPercent} segments={progressSegments(progress)} />
          ) : null}
          <p className="mt-2 text-[12.5px] text-text-secondary">
            {total > 0 ? `${done} dari ${total} tahapan selesai` : "Belum ada tahapan"}
            {progress && progress.overdueMilestoneCount > 0 && <span className="font-semibold text-danger-text"> · {progress.overdueMilestoneCount} lewat tenggat</span>}
          </p>
        </div>
      </section>

      <div className="grid grid-cols-1 gap-5 xl:grid-cols-[minmax(0,1fr)_20rem]">
        <div className="flex min-w-0 flex-col gap-5">
          <Card>
            <CardHeader title="Perlu tindakan" />
            <CardContent>
              {actions.length === 0 ? (
                <EmptyState
                  icon={<CircleCheck className="h-6 w-6 text-success-text" />}
                  title="Tidak ada yang perlu ditindak"
                  description="Termin, kendala, vendor, dan jadwal proyek ini sejauh ini berjalan sesuai rencana."
                />
              ) : (
                <ul className="-my-1 flex flex-col divide-y divide-border-light">
                  {actions.map((a) => (
                    <li key={`${a.kind}-${a.id}`}>
                      <Link
                        to={ROUTE_PATHS.projectDetail(projectId, a.tab)}
                        className="-mx-2 flex items-start gap-3 rounded-md px-2 py-2.5 hover:bg-surface-muted/60 focus-visible:outline-2 focus-visible:outline-solid focus-visible:outline-ring"
                      >
                        <span aria-hidden className={`mt-1.5 h-2 w-2 shrink-0 rounded-full ${a.tone === "danger" ? "bg-danger" : "bg-warning"}`} />
                        <span className="min-w-0">
                          <span className="block text-[13.5px] font-semibold text-text-primary">{a.title}</span>
                          <span className="block text-[12.5px] text-text-secondary">{a.detail}</span>
                        </span>
                      </Link>
                    </li>
                  ))}
                </ul>
              )}
            </CardContent>
          </Card>

          <div className="grid grid-cols-1 gap-5 lg:grid-cols-2">
            <Card>
              <CardHeader
                title="Jadwal berikutnya"
                action={<PanelLink to={ROUTE_PATHS.projectDetail(projectId, "jadwal")}>Lihat jadwal</PanelLink>}
              />
              <CardContent>
                {nextMilestones.length === 0 ? (
                  <p className="text-[13px] text-text-secondary">{total > 0 ? "Semua tahapan sudah selesai." : "Belum ada tahapan."}</p>
                ) : (
                  <ul className="-my-1 flex flex-col divide-y divide-border-light">
                    {nextMilestones.map((m) => (
                      <li key={m.id} className="flex items-center justify-between gap-3 py-2.5">
                        <div className="min-w-0">
                          <p className="truncate text-[13.5px] font-medium text-text-primary">{m.name}</p>
                          {/* H-n counts to the event day like a wedding checklist (Gladi resik = H-1); how soon it is from today comes after (QA RT-02). */}
                          <p className="flex flex-wrap gap-x-1.5 text-[12.5px] text-text-secondary">
                            <span className="whitespace-nowrap">
                              {formatDate(m.targetDate)} · <span className="font-medium text-text-primary">{formatEventOffset(m.targetDate, project.eventDate)}</span>
                            </span>
                            <span className="whitespace-nowrap">({formatDaysFromToday(m.targetDate)})</span>
                          </p>
                        </div>
                        <MilestoneStatusBadge status={m.status} />
                      </li>
                    ))}
                  </ul>
                )}
              </CardContent>
            </Card>

            <Card>
              <CardHeader
                title="Termin berikutnya"
                action={hasTerms ? <PanelLink to={ROUTE_PATHS.projectDetail(projectId, "keuangan/klien")}>Lihat termin</PanelLink> : undefined}
              />
              <CardContent>
                {nextTerm ? (
                  <div className="flex flex-wrap items-start justify-between gap-3">
                    <div className="min-w-0">
                      <p className="text-[14px] font-semibold text-text-primary">{nextTerm.label}</p>
                      <p className="text-[12.5px] text-text-secondary">
                        Jatuh tempo {formatDate(nextTerm.dueDate)} · {formatDaysFromToday(nextTerm.dueDate)}
                      </p>
                      <p className="mt-2 text-[20px] leading-tight font-bold tabular-nums text-text-primary">{formatCurrency(nextTerm.remaining)}</p>
                    </div>
                    <Badge tone={TERM_STATUS_TONE[nextTerm.status]}>{TERM_STATUS_LABELS.staff[nextTerm.status]}</Badge>
                  </div>
                ) : !hasTerms ? (
                  <p className="text-[13px] text-text-secondary">
                    Belum ada jadwal termin.{" "}
                    <Link to={ROUTE_PATHS.projectDetail(projectId, "keuangan/klien")} className="font-medium text-primary underline-offset-2 hover:underline">
                      Buat jadwal termin
                    </Link>
                  </p>
                ) : (
                  <p className="text-[13px] text-text-secondary">Semua termin sudah lunas.</p>
                )}
              </CardContent>
            </Card>
          </div>

          <ProjectFacts />
        </div>

        <aside aria-label="Klien, foto sampul, dan riwayat" className="flex min-w-0 flex-col gap-5">
          <ClientsPanel projectId={projectId} />
          <ProjectCoverCard projectId={projectId} />
          <HistoryPanel projectId={projectId} />
        </aside>
      </div>
    </div>
  );
}

function PanelLink({ to, children }: { to: string; children: ReactNode }) {
  return (
    <Link to={to} className="text-[13px] font-medium text-primary underline-offset-2 hover:underline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-solid focus-visible:outline-ring">
      {children}
    </Link>
  );
}

/** The facts that used to fill the header: who, when booked, which package, the team, the notes. */
function ProjectFacts() {
  const project = useProjectStore((s) => s.currentProject);
  const team = useProjectStore((s) => s.overview?.team);
  if (!project) return null;
  const couple = [project.brideName, project.groomName].filter(Boolean).join(" & ");
  return (
    <Card>
      <CardHeader title="Detail proyek" />
      <CardContent className="flex flex-col gap-4">
        <dl className="grid grid-cols-2 gap-x-6 gap-y-3 lg:grid-cols-3">
          <Fact label="Pasangan" value={couple || "-"} />
          <Fact label="Tanggal acara" value={formatDate(project.eventDate)} />
          <Fact label="Tanggal booking" value={formatDate(project.prepStartDate)} />
          <Fact label="Paket" value={project.packageName || "-"} />
          <Fact label="Venue" value={project.venue || "-"} />
          <Fact label="PIC wedding planner" value={team?.picName || project.picName || "Belum ditugaskan"} />
          <Fact label="PIC sales" value={team?.picSalesName || "Belum ditugaskan"} />
        </dl>
        {project.description && (
          <div className="border-t border-border-light pt-4">
            <p className="mb-1 text-[12.5px] text-text-secondary">Catatan</p>
            <p className="max-w-[70ch] text-[13.5px] whitespace-pre-line text-text-primary">{project.description}</p>
          </div>
        )}
      </CardContent>
    </Card>
  );
}

function Fact({ label, value }: { label: string; value: string }) {
  return (
    <div className="min-w-0">
      <dt className="text-[12.5px] text-text-secondary">{label}</dt>
      <dd className="mt-0.5 truncate text-[13.5px] font-medium text-text-primary" title={value}>
        {value}
      </dd>
    </div>
  );
}

/** "Klien & akses portal": who of the couple and family can sign in; managing them opens in a drawer. */
function ClientsPanel({ projectId }: { projectId: string }) {
  const clients = useProjectStore((s) => s.overview?.clients) ?? [];
  const fetchOverview = useProjectStore((s) => s.fetchOverview);
  const canManage = useCan("client.manage");
  const [open, setOpen] = useState(false);

  function close() {
    setOpen(false);
    // Someone may have been added, invited or removed: the panel reads the overview, so refresh it.
    void fetchOverview(projectId).catch(() => undefined);
  }

  return (
    <Card>
      <CardHeader
        title="Klien & akses portal"
        action={
          clients.length > 0 ? (
            <Button variant="ghost" size="sm" onClick={() => setOpen(true)}>
              {canManage ? "Kelola" : "Lihat"}
            </Button>
          ) : undefined
        }
      />
      <CardContent>
        {clients.length === 0 ? (
          <div className="flex flex-col items-start gap-3">
            <p className="text-[13px] text-text-secondary">Belum ada klien. Undang pengantin agar mereka bisa melihat jadwal, pembayaran, dan bukti di portal.</p>
            {canManage && (
              <Button size="sm" variant="secondary" icon={<UserPlus className="h-3.5 w-3.5" />} onClick={() => setOpen(true)}>
                Undang klien
              </Button>
            )}
          </div>
        ) : (
          <ul className="-my-1 flex flex-col divide-y divide-border-light">
            {clients.map((c) => (
              <li key={c.id} className="flex items-center justify-between gap-3 py-2.5">
                <div className="min-w-0">
                  <p className="truncate text-[13.5px] font-medium text-text-primary">{c.name}</p>
                  <p className="text-[12.5px] text-text-secondary">{CLIENT_ROLE_LABEL[c.role] ?? c.role}</p>
                </div>
                {c.hasLogin ? <Badge tone="success">Bisa masuk</Badge> : <Badge tone="neutral">Belum masuk</Badge>}
              </li>
            ))}
          </ul>
        )}
      </CardContent>
      <Drawer open={open} onClose={close} title="Klien & akses portal" description="Kontak pengantin dan perwakilan keluarga, dan tautan masuk ke portal klien." size="lg">
        {open && (
          <Suspense fallback={<SkeletonRows rows={3} label="Memuat klien" />}>
            <ProjectClientsSection projectId={projectId} embedded />
          </Suspense>
        )}
      </Drawer>
    </Card>
  );
}

/** "Riwayat": the five latest changes; the whole history opens in a drawer. */
function HistoryPanel({ projectId }: { projectId: string }) {
  const fetchRecentActivity = useProjectStore((s) => s.fetchRecentActivity);
  const [entries, setEntries] = useState<ActivityLogEntry[] | null>(null);
  const [failed, setFailed] = useState(false);
  const [open, setOpen] = useState(false);

  useEffect(() => {
    let cancelled = false;
    setFailed(false);
    fetchRecentActivity(projectId, RECENT_ACTIVITY)
      .then((list) => !cancelled && setEntries(list))
      .catch(() => !cancelled && setFailed(true));
    return () => {
      cancelled = true;
    };
  }, [projectId, fetchRecentActivity]);

  return (
    <Card>
      <CardHeader
        title="Riwayat"
        action={
          entries && entries.length > 0 ? (
            <Button variant="ghost" size="sm" onClick={() => setOpen(true)}>
              Lihat semua
            </Button>
          ) : undefined
        }
      />
      <CardContent>
        {failed ? (
          <p className="text-[13px] text-text-secondary">Riwayat belum bisa dimuat.</p>
        ) : entries === null ? (
          <SkeletonRows rows={3} label="Memuat riwayat" />
        ) : entries.length === 0 ? (
          <p className="text-[13px] text-text-secondary">Perubahan pada proyek ini akan tercatat di sini.</p>
        ) : (
          <ActivityList entries={entries} />
        )}
      </CardContent>
      <Drawer open={open} onClose={() => setOpen(false)} title="Riwayat proyek" description="Semua perubahan yang tercatat otomatis, terbaru di atas." size="lg">
        {open && (
          <Suspense fallback={<SkeletonRows rows={5} label="Memuat riwayat" />}>
            <ProjectActivitySection projectId={projectId} />
          </Suspense>
        )}
      </Drawer>
    </Card>
  );
}
