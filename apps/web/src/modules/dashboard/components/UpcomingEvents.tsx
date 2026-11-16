import { Link } from "react-router-dom";
import { CalendarDays } from "lucide-react";
import { Card, CardContent, CardHeader } from "@/shared/components/ui/Card";
import { daysUntil } from "@/modules/projects/lib/dates";
import { formatDayCountdown } from "@/shared/lib/formatters";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import type { Project } from "@/modules/projects/types";

const MONTHS = ["Jan", "Feb", "Mar", "Apr", "Mei", "Jun", "Jul", "Agu", "Sep", "Okt", "Nov", "Des"];

const linkClass =
  "text-[13px] font-semibold text-primary underline-offset-2 hover:underline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-solid focus-visible:outline-ring";

/** The next events across the workspace, each as a small date tile: the day a WO plans everything towards. */
export function UpcomingEvents({ projects, canCreate }: { projects: Project[]; canCreate: boolean }) {
  return (
    <Card className="flex h-full flex-col">
      <CardHeader
        title="Acara berikutnya"
        action={
          <Link to={ROUTE_PATHS.calendar} className={linkClass}>
            Buka kalender
          </Link>
        }
      />
      <CardContent className="flex-1 p-0">
        {projects.length === 0 ? (
          <div className="flex h-full flex-col items-center justify-center gap-2 px-5 py-10 text-center">
            <CalendarDays aria-hidden className="size-7 text-muted-foreground" />
            <p className="text-[14px] font-semibold text-foreground">Belum ada acara yang akan datang</p>
            <p className="max-w-xs text-[13px] text-muted-foreground">Tanggal acara diambil dari setiap proyek yang masih berjalan.</p>
            {canCreate && (
              <Link to={ROUTE_PATHS.projectNew} className={`mt-1 ${linkClass}`}>
                Buat proyek
              </Link>
            )}
          </div>
        ) : (
          <ul className="divide-y divide-border-light">
            {projects.map((p) => {
              const [, m, d] = p.eventDate.split("-").map(Number);
              const days = daysUntil(p.eventDate);
              return (
                <li key={p.id}>
                  <Link
                    to={ROUTE_PATHS.projectDetail(p.id)}
                    className="flex items-center gap-3 px-5 py-2.5 hover:bg-muted focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-solid focus-visible:outline-ring"
                  >
                    <span aria-hidden className="flex w-11 shrink-0 flex-col items-center rounded-md border border-border bg-card py-1 leading-none">
                      <span className="text-[17px] font-bold tabular-nums text-foreground">{d}</span>
                      <span className="mt-0.5 text-[12px] font-medium text-muted-foreground">{MONTHS[m - 1]}</span>
                    </span>
                    <span className="min-w-0 flex-1">
                      <span className="block truncate text-[14px] font-medium text-foreground">{p.name}</span>
                      <span className="block truncate text-[12px] text-muted-foreground">
                        <span className="sr-only">{`${d} ${MONTHS[m - 1]}, `}</span>
                        {p.venue || "Venue belum diisi"}
                      </span>
                    </span>
                    <span className="shrink-0 text-[13px] font-semibold tabular-nums text-highlight-text">
                      {formatDayCountdown(days)}
                    </span>
                  </Link>
                </li>
              );
            })}
          </ul>
        )}
      </CardContent>
    </Card>
  );
}
