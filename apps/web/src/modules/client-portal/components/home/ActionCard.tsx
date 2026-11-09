import { Link } from "react-router-dom";
import { AlertCircle, ChevronRight, Clock, HeartHandshake } from "lucide-react";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import { daysUntil } from "@/modules/projects/lib/dates";
import type { PortalAction } from "@/modules/client-portal/types";

/** The server hands over a due date for payments: past it is red, within a week amber. */
function urgency(a: PortalAction): "late" | "soon" | "info" {
  if (a.kind !== "pay_term" || !a.dueDate) return "info";
  return daysUntil(a.dueDate) < 0 ? "late" : "soon";
}

const STYLE = {
  late: { box: "border-danger/30 bg-danger-soft", icon: "text-danger-text", Icon: AlertCircle },
  soon: { box: "border-warning/30 bg-warning-soft", icon: "text-warning-text", Icon: Clock },
  info: { box: "border-border bg-card", icon: "text-muted-foreground", Icon: HeartHandshake },
} as const;

/**
 * "Yang perlu Anda lakukan": the only place the portal uses red, and only when
 * the client really has to act (a payment past its due date). What counts, and
 * in which order, is decided by the server (E5); this only draws it.
 */
export function ActionCard({ actions }: { actions: PortalAction[] }) {
  if (actions.length === 0) return null;
  return (
    <section aria-labelledby="todo-heading" className="rounded-2xl border border-border bg-card p-5 sm:p-6">
      <h2 id="todo-heading" className="text-[15px] font-semibold text-foreground">Yang perlu Anda lakukan</h2>
      <ul className="mt-3 flex flex-col gap-2.5">
        {actions.map((a) => {
          const { box, icon, Icon } = STYLE[urgency(a)];
          return (
            <li key={`${a.kind}-${a.title}`}>
              <Link
                to={ROUTE_PATHS.portal(a.target)}
                className={`flex min-h-14 items-start gap-3 rounded-xl border px-4 py-3 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-foreground ${box}`}
              >
                <Icon className={`mt-0.5 size-4 shrink-0 ${icon}`} aria-hidden />
                <span className="min-w-0 flex-1">
                  <span className="block text-[14px] font-semibold text-foreground">{a.title}</span>
                  <span className="block text-[13px] text-muted-foreground">{a.detail}</span>
                </span>
                <ChevronRight className="mt-0.5 size-4 shrink-0 text-muted-foreground" aria-hidden />
              </Link>
            </li>
          );
        })}
      </ul>
    </section>
  );
}
