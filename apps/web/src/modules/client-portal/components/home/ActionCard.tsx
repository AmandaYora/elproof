import { Link } from "react-router-dom";
import { AlertCircle, ChevronRight, ClipboardList, Clock, HeartHandshake } from "lucide-react";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import { formatCurrency } from "@/shared/lib/formatters";
import { daysUntil } from "@/modules/projects/lib/dates";
import { WhatsAppButton } from "@/modules/client-portal/components/WhatsAppButton";
import { CompleteTaskButton, taskDueText } from "@/modules/client-portal/components/home/ClientTasks";
import { askMessage, type PortalContact } from "@/modules/client-portal/lib/usePortalContact";
import type { ClientTask, PortalAction } from "@/modules/client-portal/types";

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

/** What the WhatsApp message asks about: "pembayaran Termin 2 Rp40.000.000", "kendala yang sedang ditangani". */
function subjectOf(a: PortalAction): string {
  if (a.kind === "open_issues") return "kendala yang sedang ditangani";
  const term = /^Pembayaran (.+?) (?:sudah lewat jatuh tempo|jatuh tempo)/.exec(a.title)?.[1];
  const what = term ? `pembayaran ${term}` : a.title.charAt(0).toLowerCase() + a.title.slice(1);
  return a.amount ? `${what} ${formatCurrency(a.amount)}` : what;
}

const TARGETS = new Set<string>(["pembayaran", "kendala"]);

function Item({ box, icon, title, detail, children }: { box: string; icon: React.ReactNode; title: React.ReactNode; detail: string; children: React.ReactNode }) {
  return (
    <li className={`flex gap-3 rounded-xl border px-4 py-3.5 ${box}`}>
      {icon}
      <div className="min-w-0 flex-1">
        <div className="text-[14px] font-semibold text-foreground">{title}</div>
        <p className="text-[13px] text-muted-foreground">{detail}</p>
        {children && <div className="mt-3 flex flex-wrap items-center gap-2">{children}</div>}
      </div>
    </li>
  );
}

/**
 * "Yang perlu Anda lakukan": the only place the portal uses red, and only when the client really has to act (a payment
 * past its due date). Which payments and kendala count, and in which order, is decided by the server (E5); the couple's
 * own tasks due within two weeks follow. Every item can be asked about on WhatsApp with the item already named.
 */
export function ActionCard({ actions, tasks, contact }: { actions: PortalAction[]; tasks: ClientTask[]; contact: PortalContact }) {
  if (actions.length === 0 && tasks.length === 0) return null;
  return (
    <section aria-labelledby="todo-heading" className="rounded-2xl border border-border bg-card p-5 sm:p-6">
      <h2 id="todo-heading" className="text-base font-semibold text-foreground">Yang perlu Anda lakukan</h2>
      <ul className="mt-3 flex flex-col gap-2.5">
        {actions.map((a) => {
          const { box, icon, Icon } = STYLE[urgency(a)];
          return (
            <Item
              key={`${a.kind}-${a.title}`}
              box={box}
              icon={<Icon className={`mt-0.5 size-4 shrink-0 ${icon}`} aria-hidden />}
              title={
                TARGETS.has(a.target) ? (
                  // The title opens the page that handles it; the WhatsApp button beside it is a separate control.
                  <Link
                    to={ROUTE_PATHS.portal(a.target)}
                    className="group inline-flex items-start gap-1 rounded-sm hover:underline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-solid focus-visible:outline-ring"
                  >
                    {a.title}
                    <ChevronRight className="mt-0.5 size-4 shrink-0 text-muted-foreground" aria-hidden />
                  </Link>
                ) : (
                  a.title
                )
              }
              detail={a.detail}
            >
              {contact.phone && <WhatsAppButton contact={contact} size="sm" label="Tanyakan lewat WhatsApp" message={askMessage(contact, subjectOf(a))} />}
            </Item>
          );
        })}
        {tasks.map((t) => {
          const late = daysUntil(t.targetDate) < 0;
          return (
            <Item
              key={`task-${t.id}`}
              box={late ? STYLE.soon.box : STYLE.info.box}
              icon={<ClipboardList className={`mt-0.5 size-4 shrink-0 ${late ? STYLE.soon.icon : "text-muted-foreground"}`} aria-hidden />}
              title={t.name}
              detail={`Tugas Anda · ${taskDueText(t)}`}
            >
              <CompleteTaskButton task={t} />
              {contact.phone && <WhatsAppButton contact={contact} size="sm" label="Tanyakan lewat WhatsApp" message={askMessage(contact, `tugas "${t.name}"`)} />}
            </Item>
          );
        })}
      </ul>
      {/* Without a WhatsApp number the "Hubungi kami" card on the same page says so once; no line here. */}
    </section>
  );
}
