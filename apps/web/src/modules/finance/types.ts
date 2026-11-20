import type { PaymentTermStatus } from "@/modules/projects/types";

/** One week of the 8-week cash flow (the same weeks GET /dashboard computes). */
export interface CashFlowWeek {
  /** Monday of the week, YYYY-MM-DD. */
  weekStart: string;
  expectedIn: number;
  /** What the WO still owes its vendors and venues; always 0 without finance.margin.view (summary.payables absent). */
  expectedOut: number;
}

/** An amount with how many items (and projects) make it up. */
export interface MoneyCount {
  amount: number;
  count: number;
  projects: number;
}

/** GET /finance/summary. */
export interface FinanceSummary {
  overdue: MoneyCount;
  dueSoon: MoneyCount;
  received: MoneyCount;
  /** What the WO still owes its vendors and venues; absent without finance.margin.view. */
  payables?: { amount: number; count: number; overdueAmount: number; overdueCount: number; incompleteCount: number };
  pendingVerification: MoneyCount;
  cashFlow: CashFlowWeek[];
}

/** One client installment across projects (GET /finance/receivables). */
export interface Receivable {
  id: string;
  projectId: string;
  projectName: string;
  brideName: string;
  groomName: string;
  label: string;
  dueDate: string;
  amount: number;
  paid: number;
  remaining: number;
  status: PaymentTermStatus;
  daysLate: number;
  picName: string;
}

export type PayableKind = "vendor" | "venue";
/** "incomplete": the WO pays it but no "Biaya rekanan" is filled in yet, so what is owed is unknown. */
export type PayableStatus = "paid" | "unpaid" | "overdue" | "incomplete";

/** One vendor or venue the WO pays, with what it still owes (GET /finance/payables, finance.margin.view only). */
export interface Payable {
  kind: PayableKind;
  id: string;
  projectId: string;
  projectName: string;
  brideName: string;
  groomName: string;
  payeeName: string;
  categoryName: string;
  /** "Biaya rekanan": what the WO agreed to pay. */
  agreedCost: number;
  paid: number;
  remaining: number;
  dueDate: string | null;
  status: PayableStatus;
  daysLate: number;
  /** No "Biaya rekanan" yet: agreedCost and remaining are 0 and mean nothing. */
  incomplete: boolean;
}

/** A client payment with proof that nobody verified yet (GET /finance/verifications). */
export interface PendingVerification {
  id: string;
  projectId: string;
  projectName: string;
  brideName: string;
  groomName: string;
  type: string;
  amount: number;
  paymentDate: string;
  method: string;
  referenceNumber: string;
  recordedByName: string;
  /** "client" when the couple sent the proof from the portal (ADR-0047). */
  source: "client" | "staff";
  /**
   * What is waiting: a client payment toward the contract, or the couple's proof of paying a vendor they pay directly
   * (Paket WO Tahap 4). Ids of the two can be the same number, so a row is keyed by both (`verificationRowId`).
   */
  kind: "clientPayment" | "supplierPayment";
  /** A supplierPayment row only: the vendor or venue and the bill it pays (null for a clientPayment row). */
  projectSupplierId: string | null;
  supplierName: string | null;
  billTermId: string | null;
  billTermLabel: string | null;
}

/** The status tabs above the installment list; each is a `status` view of the API. */
export type ReceivableTab = "overdue" | "due_30d" | "all" | "paid";
