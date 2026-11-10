import { create } from "zustand";
import { httpClient } from "@/shared/services/http-client";
import { API } from "@/shared/services/api-endpoints";
import type { SubscriptionPlan } from "@/shared/data/subscriptionPlans";

export interface SubscriptionPlanInput {
  appId: string;
  name: string;
  durationMonths: number;
  price: number;
  features: string[];
}

interface RawPlan {
  id: number;
  appId: string;
  name: string;
  durationMonths: number;
  price: number;
  features: string[] | null;
  isActive: boolean;
}

function toPlan(raw: RawPlan): SubscriptionPlan {
  return {
    id: String(raw.id),
    appId: raw.appId,
    name: raw.name,
    durationMonths: raw.durationMonths,
    price: raw.price,
    features: raw.features ?? [],
    isActive: raw.isActive,
  };
}

async function requestPlans(appId?: string): Promise<SubscriptionPlan[]> {
  const params: Record<string, string | boolean> = { all: true };
  if (appId) params.appId = appId;
  const res = await httpClient.get(API.billing.plans, { params });
  return (res.data.data as RawPlan[]).map(toPlan);
}

interface SubscriptionPlanState {
  plans: SubscriptionPlan[];
  /** The App `plans` was last loaded for ("" = every App the caller may see); writes reload the same view. */
  appFilter: string;
  isLoading: boolean;
  /** Loads the catalog. A Platform admin may narrow it to one App; every other caller always gets ElProof's own. */
  fetchPlans: (appId?: string) => Promise<void>;
  /** One App's plans, without touching the shared `plans` list (the App detail dialog). */
  fetchPlansForApp: (appId: string) => Promise<SubscriptionPlan[]>;
  createPlan: (values: SubscriptionPlanInput) => Promise<void>;
  updatePlan: (id: string, values: SubscriptionPlanInput) => Promise<void>;
  togglePlanActive: (id: string) => Promise<void>;
}

// Single source of truth for the plan catalog — read by both the Platform Console
// (where a super admin manages it) and the WO Console's own Langganan card, so the
// two are never allowed to drift out of sync the way two hardcoded constants would.
// Backed by the real `billing` module — fetch-then-set, no client cache. Which App's
// plans come back is decided by the server (ADR-0040).
export const useSubscriptionPlanStore = create<SubscriptionPlanState>((set, get) => ({
  plans: [],
  appFilter: "",
  isLoading: false,

  fetchPlans: async (appId = "") => {
    set({ isLoading: true, appFilter: appId });
    try {
      const plans = await requestPlans(appId);
      // A slower, older request must not overwrite the view the person switched to since.
      if (get().appFilter === appId) set({ plans, isLoading: false });
    } catch (err) {
      set({ isLoading: false });
      throw err;
    }
  },

  fetchPlansForApp: (appId) => requestPlans(appId),

  createPlan: async (values) => {
    await httpClient.post(API.billing.plans, values);
    await get().fetchPlans(get().appFilter);
  },

  updatePlan: async (id, values) => {
    await httpClient.patch(API.billing.plan(id), values);
    await get().fetchPlans(get().appFilter);
  },

  togglePlanActive: async (id) => {
    await httpClient.post(API.billing.planToggleActive(id));
    await get().fetchPlans(get().appFilter);
  },
}));
