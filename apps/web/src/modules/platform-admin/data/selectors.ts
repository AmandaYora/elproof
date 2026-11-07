import type { PlatformAdmin, Tenant } from "@/modules/platform-admin/data/types";
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
