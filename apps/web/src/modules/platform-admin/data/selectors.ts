import type { PlatformAdmin, SubscriptionTransaction, Tenant } from "@/modules/platform-admin/data/types";
import { daysBetween, todayISO } from "@/shared/lib/formatters";

export function getPlatformAdmin(admins: PlatformAdmin[], id: string): PlatformAdmin | null {
  return admins.find((a) => a.id === id) ?? null;
}

/** End of the current paid period or trial (computed by the API). */
export function periodEndsAt(tenant: Tenant): string | null {
  return tenant.lifecycle?.periodEndsAt ?? tenant.subscriptionExpiresAt;
}

export function daysUntilExpiry(tenant: Tenant): number | null {
  const end = periodEndsAt(tenant);
  if (!end) return null;
  return daysBetween(todayISO(), end);
}

export function getPlatformStats(tenantList: Tenant[], transactionList: SubscriptionTransaction[]) {
  // Paying tenants: a paid period that is running or within its grace
  // period, and not frozen by the platform team.
  const activeTenants = tenantList.filter(
    (t) => !t.isSuspended && (t.subscriptionStatus === "active" || t.subscriptionStatus === "expiring_soon" || t.subscriptionStatus === "past_due")
  ).length;
  const expiringSoonCount = tenantList.filter((t) => t.subscriptionStatus === "expiring_soon").length;
  const expiredCount = tenantList.filter((t) => t.subscriptionStatus === "read_only" || t.subscriptionStatus === "suspended").length;
  // New charges are created as "pending" (waiting for the QR to be paid);
  // "unpaid" is a legacy status.
  const unpaidCount = transactionList.filter((t) => t.status === "pending" || t.status === "unpaid").length;
  const paidRevenue = transactionList.filter((t) => t.status === "paid").reduce((sum, t) => sum + t.amount, 0);
  return {
    totalTenants: tenantList.length,
    activeTenants,
    expiringSoonCount,
    expiredCount,
    unpaidCount,
    paidRevenue,
  };
}
