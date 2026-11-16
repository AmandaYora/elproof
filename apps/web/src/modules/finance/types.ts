import type { PaymentTermStatus } from "@/modules/projects/types";

/** One week of the 8-week cash flow (the same weeks GET /dashboard computes). */
export interface CashFlowWeek {
  /** Monday of the week, YYYY-MM-DD. */
  weekStart: string;
  expectedIn: number;
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
  payables: { amount: number; count: number; overdueAmount: number; overdueCount: number };
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
export type PayableStatus = "paid" | "unpaid" | "overdue";

/** One amount the WO owes a vendor or a venue (GET /finance/payables). */
export interface Payable {
  kind: PayableKind;
  id: string;
  projectId: string;
  projectName: string;
  brideName: string;
  groomName: string;
  payeeName: string;
  categoryName: string;
  contractValue: number;
  paid: number;
  remaining: number;
  dueDate: string | null;
  status: PayableStatus;
  daysLate: number;
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
}

/** The status tabs above the installment list; each is a `status` view of the API. */
export type ReceivableTab = "overdue" | "due_30d" | "all" | "paid";
