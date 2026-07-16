export type TenantSubscriptionStatus = "active" | "expiring_soon" | "expired" | "pending_payment";

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
  isSuspended: boolean;
  lastCredentialResetAt: string | null;
}

export type SubscriptionTransactionType = "new" | "renewal";
export type SubscriptionTransactionStatus = "unpaid" | "pending" | "paid" | "expired" | "granted";

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

export type PlatformAdminRole = "Super Admin" | "Support";

export interface PlatformAdmin {
  id: string;
  name: string;
  title: string;
  role: PlatformAdminRole;
  username: string;
  email: string;
  phone: string;
  isActive: boolean;
}
