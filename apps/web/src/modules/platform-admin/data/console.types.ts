import type { TenantSubscriptionStatus } from "@/modules/platform-admin/data/types";

/** One tenant that needs attention (trial ending, overdue payment). */
export interface AttentionTenant {
  tenantId: number;
  name: string;
  status: TenantSubscriptionStatus;
  daysLeft: number;
}

/** GET /platform/metrics. Percentages are null while there is nothing to divide by. */
export interface PlatformMetrics {
  totalTenants: number;
  trialing: number;
  active: number;
  pastDue: number;
  readOnly: number;
  suspended: number;
  frozen: number;
  paidTenants: number;
  mrr: number;
  arpa: number;
  trialsEnded30d: number;
  trialsConverted30d: number;
  trialConversionPct: number | null;
  churned30d: number;
  churnedMrr: number;
  churnPct: number | null;
  newPaid30d: number;
  newMrr30d: number;
  revenue30d: number;
  revenuePrev30d: number;
  overdueValue: number;
  trialsEndingSoon: AttentionTenant[];
  overdueTenants: AttentionTenant[];
}

export interface UsageItem {
  used: number;
  /** null: no limit (paid plan). */
  max: number | null;
}

export interface TenantMember {
  id: number;
  name: string;
  email: string;
  role: string;
  isActive: boolean;
}

export interface ManualActivation {
  id: number;
  planId: number;
  admin: string;
  reason: string;
  reasonLabel: string;
  proofReference: string;
  note: string;
  expiresAt: string;
  createdAt: string;
}

export interface SignupConsent {
  termsVersion: string;
  termsAcceptedAt: string;
  marketingOptIn: boolean;
  ipAddress: string;
}

export interface TenantOverview {
  planName: string;
  consent: SignupConsent | null;
  activations: ManualActivation[];
  staff: UsageItem;
  activeProjects: UsageItem;
  clients: UsageItem;
  members: TenantMember[];
  openDataRequests: number;
  supportAccessUntil: string | null;
  /** Raw tenant row; the page maps it with the store's toTenant. */
  tenant: unknown;
}

export interface TenantNote {
  id: number;
  admin: string;
  body: string;
  createdAt: string;
}

export type DataRequestKind = "export" | "delete";
export type DataRequestStatus = "open" | "done" | "rejected";

export interface DataRequest {
  id: number;
  tenantId: number;
  tenantName: string;
  kind: DataRequestKind;
  status: DataRequestStatus;
  requestedBy: string;
  note: string;
  dueAt: string;
  daysUntilDue: number;
  createdAt: string;
  completedAt: string | null;
  completedBy: string;
  completionNote: string;
}

export type AnnouncementKind = "info" | "warning" | "maintenance";
export type AnnouncementAudience = "all" | "plan" | "status";

export interface Announcement {
  id: number;
  title: string;
  body: string;
  kind: AnnouncementKind;
  audience: AnnouncementAudience;
  audienceValue: string;
  startsAt: string;
  endsAt: string | null;
  active: boolean;
}

export interface SubscriptionPolicy {
  trialDays: number;
  expiringSoonDays: number;
  graceDays: number;
  readOnlyDays: number;
}

export interface SupportGrant {
  grantedAt: string;
  expiresAt: string;
}

export interface SupportSession {
  id: number;
  admin: string;
  reason: string;
  startedAt: string;
  expiresAt: string;
  endedAt: string | null;
}

export interface SupportStatus {
  grant: SupportGrant | null;
  sessions: SupportSession[];
}

export interface ImpersonationResult {
  sessionId: number;
  accessToken: string;
  expiresAt: string;
  tenantId: number;
  role: string;
}

export interface AuditEntry {
  id: number;
  occurredAt: string;
  actorType: string;
  actorId: string;
  actorRole: string;
  tenantId: number | null;
  action: string;
  method: string;
  path: string;
  statusCode: number;
  succeeded: boolean;
  ip: string;
}

export interface AuditFilter {
  actorType?: string;
  actorId?: string;
  tenantId?: string;
  outcome?: "ok" | "failed" | "";
  q?: string;
  from?: string;
  to?: string;
}
