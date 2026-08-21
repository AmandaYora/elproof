import type { BadgeTone } from "@/shared/components/ui/Badge";
import type {
  ProjectStatus,
  EngagementStatus,
  MilestoneStatus,
  IssueImpact,
  IssueStatus,
  ProjectCondition,
  ClientPaymentStatus,
} from "@/modules/projects/types";

export const PROJECT_STATUS_TONE: Record<ProjectStatus, BadgeTone> = {
  Draft: "neutral",
  Preparation: "info",
  Ready: "success",
  Completed: "navy",
  Cancelled: "danger",
};

export const ENGAGEMENT_STATUS_TONE: Record<EngagementStatus, BadgeTone> = {
  Planned: "neutral",
  Negotiation: "neutral",
  Booked: "info",
  "DP Paid": "info",
  "In Progress": "info",
  "Fully Paid": "success",
  Ready: "success",
  Completed: "navy",
  Cancelled: "danger",
};

export const MILESTONE_STATUS_TONE: Record<MilestoneStatus, BadgeTone> = {
  "Not Started": "neutral",
  "In Progress": "info",
  Completed: "success",
  Blocked: "danger",
  Cancelled: "neutral",
};

export const ISSUE_IMPACT_TONE: Record<IssueImpact, BadgeTone> = {
  Low: "neutral",
  Medium: "warning",
  High: "danger",
  Critical: "danger",
};

export const ISSUE_STATUS_TONE: Record<IssueStatus, BadgeTone> = {
  Open: "danger",
  "In Review": "warning",
  "In Resolution": "warning",
  Resolved: "success",
  Closed: "neutral",
};

export const CONDITION_TONE: Record<ProjectCondition, BadgeTone> = {
  "On Track": "success",
  Attention: "warning",
  "At Risk": "danger",
};

export const CONDITION_LABEL: Record<ProjectCondition, string> = {
  "On Track": "Sesuai Rencana",
  Attention: "Perlu Perhatian",
  "At Risk": "Berisiko",
};

// Kwitansi & Tagihan feature (PLAN.md) -- identifier stays English
// (matches the API value), label is Indonesian, same convention as
// Timeline/"Tanggal Booking" elsewhere in this codebase.
export const CLIENT_PAYMENT_STATUS_LABEL: Record<ClientPaymentStatus, string> = {
  Paid: "Lunas",
  PartiallyPaid: "Dibayar Sebagian",
  Unpaid: "Belum Dibayar",
  Overdue: "Jatuh Tempo",
};

export const CLIENT_PAYMENT_STATUS_TONE: Record<ClientPaymentStatus, BadgeTone> = {
  Paid: "success",
  PartiallyPaid: "info",
  Unpaid: "neutral",
  Overdue: "danger",
};
