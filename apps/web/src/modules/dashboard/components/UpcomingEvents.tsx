import { Link } from "react-router-dom";
import { Card, CardHeader, CardContent } from "@/shared/components/ui/Card";
import { EmptyState } from "@/shared/components/feedback/EmptyState";
import type { Project } from "@/modules/projects/types";
import { daysUntil } from "@/modules/projects/lib/dates";
import { formatDate } from "@/shared/lib/formatters";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import { CalendarClock } from "lucide-react";

export function UpcomingEvents({ projects }: { projects: Project[] }) {
  return (
    <Card>
      <CardHeader title="Acara Terdekat" subtitle="Diurutkan berdasarkan tanggal acara." />
      <CardContent className="p-0">
        {projects.length === 0 ? (
          <EmptyState icon={<CalendarClock className="h-7 w-7" />} title="Belum ada acara mendatang" />
        ) : (
          <ul className="divide-y divide-border-light">
            {projects.map((p) => (
              <li key={p.id}>
                <Link to={ROUTE_PATHS.projectDetail(p.id)} className="flex items-center justify-between gap-3 px-5 py-3 hover:bg-surface-muted">
                  <span className="min-w-0">
                    <span className="block truncate text-[13.5px] font-semibold text-text-primary">{p.name}</span>
                    <span className="block text-[12.5px] text-text-secondary">{formatDate(p.eventDate)} · {p.venue}</span>
                  </span>
                  <span className="shrink-0 rounded-full bg-navy-900/10 px-2.5 py-1 text-[11px] font-bold tabular-nums text-navy-900">
                    H-{daysUntil(p.eventDate)}
                  </span>
                </Link>
              </li>
            ))}
          </ul>
        )}
      </CardContent>
    </Card>
  );
}
