import { CircleAlert, CircleCheck, CircleDashed, Clock, Gem, ListChecks, Wallet, type LucideIcon } from "lucide-react";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import { MILESTONE_STATUS_LABELS, PROJECT_STATUS_LABELS, TERM_STATUS_LABELS } from "@/shared/lib/labels";
import { formatCurrency } from "@/shared/lib/formatters";
import type { PaymentTermStatus } from "@/modules/projects/types";
import type { CalendarItem, CalendarKind } from "@/modules/calendar/types";

/** The words a person reads for each kind (DESIGN.md §7: Jadwal/tahapan, Termin). */
export const KIND_LABELS: Record<CalendarKind, string> = {
  event: "Hari acara",
  milestone: "Tahapan",
  payment: "Termin",
};

export const KIND_ICON: Record<CalendarKind, LucideIcon> = {
  event: Gem,
  milestone: ListChecks,
  payment: Wallet,
};

/**
 * Where an entry opens: the wedding day → the project's summary, a step → its schedule, an installment → its money.
 * The schedule tab is addressed as "/jadwal" (the project-detail revamp renames ".../milestone" with a redirect).
 */
export function itemHref(item: CalendarItem): string {
  const id = String(item.projectId);
  switch (item.kind) {
    case "event":
      return ROUTE_PATHS.projectDetail(id);
    case "milestone":
      return `${ROUTE_PATHS.projects}/${id}/jadwal`;
    case "payment":
      return ROUTE_PATHS.projectDetail(id, "pembayaran/client");
  }
}

/** A step still open after its target day is late (ADR-0042: late only from the day after, WIB). */
export function isLateMilestone(item: CalendarItem, today: string): boolean {
  return item.kind === "milestone" && item.date < today;
}

// Payment status → its tone, icon and the short word the month chip shows. Never colour alone (DESIGN.md T5).
export const PAYMENT_LOOK: Record<PaymentTermStatus, { icon: LucideIcon; chip: string; mark: string; short: string }> = {
  overdue: { icon: CircleAlert, chip: "bg-danger-soft text-danger-text border-danger/30", mark: "bg-danger-soft text-danger-text", short: "Terlambat" },
  due_soon: { icon: Clock, chip: "bg-warning-soft text-warning-text border-warning/30", mark: "bg-warning-soft text-warning-text", short: "Jatuh tempo" },
  partial: { icon: CircleDashed, chip: "bg-warning-soft text-warning-text border-warning/30", mark: "bg-warning-soft text-warning-text", short: "Sebagian" },
  paid: { icon: CircleCheck, chip: "bg-success-soft text-success-text border-success/30", mark: "bg-success-soft text-success-text", short: "Lunas" },
  upcoming: { icon: Wallet, chip: "bg-card text-foreground border-border", mark: "bg-muted text-muted-foreground", short: "Termin" },
};

export interface ItemLook {
  icon: LucideIcon;
  /** Chip colours (semantic tokens only). */
  chip: string;
  /** The round icon mark of an agenda row. */
  mark: string;
  /** The month chip's one line. */
  chipText: string;
  /** Primary line of an agenda row. */
  heading: string;
  /** Secondary line of an agenda row. */
  meta: string;
  /** The status in words (agenda badge, tooltips, screen readers). */
  statusText: string;
  /** Everything at once: the chip's tooltip. */
  full: string;
  /** Read after the chip's visible text, so its accessible name starts with what is on screen (WCAG 2.5.3). */
  srExtra: string;
  late: boolean;
}

export function lookOf(item: CalendarItem, today: string): ItemLook {
  switch (item.kind) {
    case "event": {
      const status = PROJECT_STATUS_LABELS.staff[item.status];
      return {
        icon: Gem,
        chip: "bg-primary text-primary-foreground border-primary",
        mark: "bg-primary text-primary-foreground",
        chipText: item.projectName,
        heading: item.projectName,
        meta: "Hari acara",
        statusText: status,
        full: `Hari acara ${item.projectName} (${status})`,
        srExtra: `, hari acara, ${status}`,
        late: false,
      };
    }
    case "milestone": {
      const late = isLateMilestone(item, today);
      const status = late ? "Terlambat" : MILESTONE_STATUS_LABELS.staff[item.status];
      return {
        icon: late ? CircleAlert : ListChecks,
        chip: late ? "bg-card text-danger-text border-danger/40" : "bg-card text-foreground border-border",
        mark: late ? "bg-danger-soft text-danger-text" : "bg-muted text-muted-foreground",
        chipText: item.title,
        heading: item.title,
        meta: `Tahapan · ${item.projectName}`,
        statusText: status,
        full: `Tahapan ${item.title}, ${item.projectName} (${status})`,
        srExtra: `, tahapan ${item.projectName}, ${status}`,
        late,
      };
    }
    case "payment": {
      const look = PAYMENT_LOOK[item.status];
      const status = TERM_STATUS_LABELS.staff[item.status];
      const owed = item.status === "paid" ? item.amount : item.remaining;
      return {
        icon: look.icon,
        chip: look.chip,
        mark: look.mark,
        chipText: `${look.short} · ${item.projectName}`,
        heading: `${item.title} · ${formatCurrency(owed)}`,
        meta: `Termin klien · ${item.projectName}`,
        statusText: status,
        full: `Termin ${item.title} ${item.projectName}, ${formatCurrency(owed)} (${status})`,
        srExtra: `, termin ${item.title}, ${formatCurrency(owed)}, ${status}`,
        late: item.status === "overdue",
      };
    }
  }
}
