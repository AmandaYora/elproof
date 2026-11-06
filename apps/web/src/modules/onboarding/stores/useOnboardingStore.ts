import { create } from "zustand";
import { httpClient } from "@/shared/services/http-client";
import { API } from "@/shared/services/api-endpoints";

export interface TrialLimits {
  maxStaff: number;
  maxActiveProjects: number;
  maxClients: number;
}

/** GET /tenants/me/onboarding (Owner only). */
export interface OnboardingStatus {
  dismissed: boolean;
  profileComplete: boolean;
  hasLogo: boolean;
  hasPaidPlan: boolean;
  usage: { staff: number; activeProjects: number; clients: number };
  /** Set while the workspace is on its free trial. */
  trialLimits: TrialLimits | null;
}

interface OnboardingState {
  status: OnboardingStatus | null;
  fetch: () => Promise<void>;
  dismiss: () => Promise<void>;
}

export const useOnboardingStore = create<OnboardingState>((set) => ({
  status: null,
  fetch: async () => {
    try {
      const res = await httpClient.get(API.platform.tenantMeOnboarding);
      set({ status: res.data.data as OnboardingStatus });
    } catch {
      // Not critical: the dashboard simply shows no checklist.
    }
  },
  dismiss: async () => {
    set((s) => (s.status ? { status: { ...s.status, dismissed: true } } : s));
    try {
      await httpClient.post(API.platform.tenantMeOnboardingDismiss);
    } catch {
      // Hidden for this visit anyway; it will reappear next load.
    }
  },
}));
