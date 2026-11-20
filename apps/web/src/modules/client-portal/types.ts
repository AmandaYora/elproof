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
    /** The main event's date (the project's Hari H). */
    eventDate: string;
    /** The main event's kind ("Resepsi"); "" without portal.timeline.view or before an event exists. */
    mainEventKind: string;
    /** Where the events happen ("Gedung Melati", or "Akad: Masjid Al-Ikhlas · Resepsi: Gedung Melati"); "" before a venue is chosen. */
    location: string;
    /** "2 acara: Akad 13 Mar · Resepsi 14 Mar"; "" with one event. */
    eventsLine: string;
    /** The packages on the rincian, in its order; [] before the rincian has one. */
    packageNames: string[];
    status: ProjectStatus;
    picName: string;
    picPhone: string;
    /** Cover photo the WO uploaded for the hero; null or absent (older API) = the plain navy hero. */
    coverUrl?: string | null;
  };
  countdown: { daysToEvent: number };
  progress?: { overallPercent: number; condition: string };
  finance?: {
    /** Nilai kontrak: the rincian's subtotal plus PPN; 0 before the rincian has a line. */
    contractValue: number;
    /** The PPN inside contractValue; 0 when the WO is not PKP. */
    contractTax: number;
    paid: number;
    remaining: number;
    nextTerm: { id: string; label: string; amount: number; dueDate: string; status: PaymentTermStatus; remaining: number } | null;
    overdueCount: number;
    /**
     * The next unpaid bill of a vendor or venue the couple pays directly (Paket WO Tahap 4, "Tagihan vendor"), earliest
     * due first; null when there is none. Never part of the contract above.
     */
    nextSupplierBill: NextSupplierBill | null;
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

export interface NextSupplierBill {
  billTermId: string;
  projectSupplierId: string;
  supplierName: string;
  label: string;
  amount: number;
  dueDate: string;
  status: PaymentTermStatus;
  remaining: number;
  /** What the couple sent for it that the WO has not checked yet. */
  pending: number;
}

export interface PortalAction {
  /** pay_supplier: a bill of a vendor the couple pays directly is due or late (Paket WO Tahap 4). */
  kind: "pay_term" | "pay_supplier" | "open_issues";
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
