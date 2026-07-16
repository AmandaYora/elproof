import { create } from "zustand";
import { httpClient } from "@/shared/services/http-client";
import { API } from "@/shared/services/api-endpoints";
import type { SubscriptionPlan } from "@/shared/data/subscriptionPlans";

export interface SubscriptionPlanInput {
  name: string;
  durationMonths: number;
  price: number;
  features: string[];
}

interface RawPlan {
  id: number;
  name: string;
  durationMonths: number;
  price: number;
  features: string[] | null;
  isActive: boolean;
}

function toPlan(raw: RawPlan): SubscriptionPlan {
  return {
    id: String(raw.id),
    name: raw.name,
    durationMonths: raw.durationMonths,
    price: raw.price,
    features: raw.features ?? [],
    isActive: raw.isActive,
  };
}

interface SubscriptionPlanState {
  plans: SubscriptionPlan[];
  isLoading: boolean;
  fetchPlans: () => Promise<void>;
  createPlan: (values: SubscriptionPlanInput) => Promise<void>;
  updatePlan: (id: string, values: SubscriptionPlanInput) => Promise<void>;
  togglePlanActive: (id: string) => Promise<void>;
}

// Single source of truth for the plan catalog — read by both the Platform Console
// (where a super admin manages it) and the WO Console's own Langganan card, so the
// two are never allowed to drift out of sync the way two hardcoded constants would.
// Backed by the real `billing` module (Fase 2) — fetch-then-set, no client cache.
export const useSubscriptionPlanStore = create<SubscriptionPlanState>((set, get) => ({
  plans: [],
  isLoading: false,

  fetchPlans: async () => {
    set({ isLoading: true });
    try {
      const res = await httpClient.get(API.billing.plans, { params: { all: true } });
      set({ plans: (res.data.data as RawPlan[]).map(toPlan), isLoading: false });
    } catch (err) {
      set({ isLoading: false });
      throw err;
    }
  },

  createPlan: async (values) => {
    await httpClient.post(API.billing.plans, values);
    await get().fetchPlans();
  },

  updatePlan: async (id, values) => {
    await httpClient.patch(API.billing.plan(id), values);
    await get().fetchPlans();
  },

  togglePlanActive: async (id) => {
    await httpClient.post(API.billing.planToggleActive(id));
    await get().fetchPlans();
  },
}));
