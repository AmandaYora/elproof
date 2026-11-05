// Subscription lifecycle (backend domain.ComputeLifecycle). "expired" is a
// legacy value that is no longer produced.
export type TenantSubscriptionStatus =
  | "trialing"
  | "active"
  | "expiring_soon"
  | "past_due"
  | "read_only"
  | "suspended"
  | "pending_payment"
  | "expired";

export type TenantAccessLevel = "full" | "read_only" | "billing_only" | "blocked";

/** Computed at read time by the API; the source of truth for access. */
export interface TenantLifecycle {
  status: TenantSubscriptionStatus;
  access: TenantAccessLevel;
  isTrial: boolean;
  periodEndsAt: string | null;
  daysLeft: number;
  readOnlyAt: string | null;
  suspendAt: string | null;
  frozen: boolean;
}

export interface Tenant {
  id: string;
  businessName: string;
  ownerName: string;
  username: string;
  email: string;
  phone: string;
  city: string;
  joinedAt: string;
  planId: string | null;
  subscriptionStatus: TenantSubscriptionStatus;
  subscriptionExpiresAt: string | null;
  trialEndsAt: string | null;
  lifecycle: TenantLifecycle;
  isSuspended: boolean;
  lastCredentialResetAt: string | null;
  brandColorPreset: string;
  hasLogo: boolean;
  customDomain: string | null;
}

export type SubscriptionTransactionType = "new" | "renewal";
export type SubscriptionTransactionStatus = "unpaid" | "pending" | "paid" | "expired" | "granted" | "cancelled";

export interface SubscriptionTransaction {
  id: string;
  tenantId: string;
  type: SubscriptionTransactionType;
  amount: number;
  paymentMethod: string;
  paymentReference: string;
  createdAt: string;
  status: SubscriptionTransactionStatus;
  paidAt: string | null;
}

export type PlatformAdminRole = "Super Admin" | "Billing" | "Support" | "Read-only";

export interface PlatformAdmin {
  id: string;
  name: string;
  title: string;
  role: PlatformAdminRole;
  username: string;
  email: string;
  phone: string;
  isActive: boolean;
  mfaEnabled: boolean;
}
