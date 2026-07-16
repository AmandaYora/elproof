import type { PlatformAdmin, SubscriptionTransaction, Tenant } from "@/modules/platform-admin/data/types";
import { daysBetween, todayISO } from "@/shared/lib/formatters";

export function getPlatformAdmin(admins: PlatformAdmin[], id: string): PlatformAdmin | null {
  return admins.find((a) => a.id === id) ?? null;
}

export function daysUntilExpiry(tenant: Tenant): number | null {
  if (!tenant.subscriptionExpiresAt) return null;
  return daysBetween(todayISO(), tenant.subscriptionExpiresAt);
}

export function getPlatformStats(tenantList: Tenant[], transactionList: SubscriptionTransaction[]) {
  const activeTenants = tenantList.filter((t) => t.subscriptionStatus === "active" || t.subscriptionStatus === "expiring_soon").length;
  const expiringSoonCount = tenantList.filter((t) => t.subscriptionStatus === "expiring_soon").length;
  const expiredCount = tenantList.filter((t) => t.subscriptionStatus === "expired").length;
  const unpaidCount = transactionList.filter((t) => t.status === "unpaid").length;
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
