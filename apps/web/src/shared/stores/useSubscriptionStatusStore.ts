import { create } from "zustand";
import { httpClient } from "@/shared/services/http-client";
import { API } from "@/shared/services/api-endpoints";

export type SubscriptionLifecycleStatus =
  | "trialing"
  | "active"
  | "expiring_soon"
  | "past_due"
  | "read_only"
  | "suspended"
  | "pending_payment"
  | "expired";

export type TenantAccess = "full" | "read_only" | "billing_only" | "blocked";

/** GET /tenants/me/subscription — computed by the API for every staff role. */
export interface SubscriptionLifecycle {
  status: SubscriptionLifecycleStatus;
  access: TenantAccess;
  isTrial: boolean;
  periodEndsAt: string | null;
  daysLeft: number;
  readOnlyAt: string | null;
  suspendAt: string | null;
  frozen: boolean;
}

interface SubscriptionStatusState {
  lifecycle: SubscriptionLifecycle | null;
  fetch: () => Promise<void>;
  reset: () => void;
}

// Shared by the WO Console banner, the locked screen and the HTTP client
// (which refreshes it whenever the API answers with a subscription code).
export const useSubscriptionStatusStore = create<SubscriptionStatusState>((set) => ({
  lifecycle: null,
  fetch: async () => {
    try {
      const res = await httpClient.get(API.platform.tenantMeSubscription);
      set({ lifecycle: res.data.data as SubscriptionLifecycle });
    } catch {
      // Keep the last known state; the banner is informational and the API
      // enforces access regardless.
    }
  },
  reset: () => set({ lifecycle: null }),
}));
