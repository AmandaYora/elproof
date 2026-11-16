import type { EvidenceType, MilestoneStatus, PaymentTermStatus, ProjectStatus } from "@/modules/projects/types";

/**
 * GET /projects/me/home (PLAN.md E5): everything Beranda shows, in one call.
 * A section the client's portal role may not see is simply absent (never null),
 * so `home.finance === undefined` means "this role has no money view".
 */
export interface PortalHome {
  project: {
    id: string;
    name: string;
    brideName: string;
    groomName: string;
    eventDate: string;
    venue: string;
    packageName: string;
    status: ProjectStatus;
    picName: string;
    picPhone: string;
    /** Cover photo the WO uploaded for the hero; null or absent (older API) = the plain navy hero. */
    coverUrl?: string | null;
  };
  countdown: { daysToEvent: number };
  progress?: { overallPercent: number; condition: string };
  finance?: {
    contractValue: number;
    paid: number;
    remaining: number;
    nextTerm: { id: string; label: string; amount: number; dueDate: string; status: PaymentTermStatus; remaining: number } | null;
    overdueCount: number;
  };
  payTo?: { bankName: string; accountNumber: string; accountHolder: string };
  contact?: { whatsapp: string };
  actions: PortalAction[];
  openIssueCount?: number;
  recentUpdates: PortalUpdate[];
  pic: { name: string; phone: string };
  /** Steps the WO assigned to the couple; absent on an older API (the section then does not show). */
  clientTasks?: ClientTask[];
}

export interface ClientTask {
  id: string;
  name: string;
  targetDate: string;
  status: MilestoneStatus;
  completedDate: string | null;
}

export interface PortalAction {
  kind: "pay_term" | "open_issues";
  title: string;
  detail: string;
  dueDate?: string;
  amount?: number;
  /** The portal page that handles it. */
  target: "pembayaran" | "kendala";
}

export interface PortalUpdate {
  kind: "milestone_done" | "payment_verified" | "issue_resolved";
  title: string;
  at: string;
  proof: { evidenceId: string; type: EvidenceType; fileName: string } | null;
  verifiedBy?: string;
}
