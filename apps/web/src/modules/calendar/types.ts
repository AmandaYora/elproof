import type { MilestoneStatus, PaymentTermStatus, ProjectStatus } from "@/modules/projects/types";

export type CalendarKind = "event" | "milestone" | "payment";

export const CALENDAR_KINDS: CalendarKind[] = ["event", "milestone", "payment"];

interface CalendarItemBase {
  /** YYYY-MM-DD */
  date: string;
  projectId: number;
  projectName: string;
  title: string;
}

/** One entry of GET /calendar (docs/API_CONTRACT.md › WO Console dashboard › /calendar). */
export type CalendarItem =
  | (CalendarItemBase & { kind: "event"; status: ProjectStatus })
  | (CalendarItemBase & { kind: "milestone"; status: MilestoneStatus })
  | (CalendarItemBase & { kind: "payment"; status: PaymentTermStatus; amount: number; remaining: number });

export type CalendarView = "bulan" | "agenda";
