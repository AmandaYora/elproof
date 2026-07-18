import { create } from "zustand";
import { httpClient } from "@/shared/services/http-client";
import { API } from "@/shared/services/api-endpoints";

export interface PaymentApp {
  appId: string;
  name: string;
  kind: "internal" | "external";
  callbackUrl: string;
  isActive: boolean;
  createdAt: string;
}

export interface CreateAppInput {
  name: string;
  callbackUrl: string;
}

export interface CreateAppResult {
  appId: string;
  name: string;
  secret: string;
}

export interface ResetAppSecretResult {
  appId: string;
  secret: string;
}

interface AppsState {
  apps: PaymentApp[];
  isLoading: boolean;
  fetchApps: () => Promise<void>;
  createApp: (input: CreateAppInput) => Promise<CreateAppResult>;
  resetAppSecret: (appId: string) => Promise<ResetAppSecretResult>;
  toggleAppActive: (appId: string) => Promise<void>;
}

// Backed by the real `payment` module (Fase 10 — App registry). Same
// fetch-then-set pattern as every other admin store (ADR-0009, no client
// cache) — see usePlatformAdminStore/usePaymentGatewayStore.
export const useAppsStore = create<AppsState>((set, get) => ({
  apps: [],
  isLoading: false,

  fetchApps: async () => {
    set({ isLoading: true });
    try {
      const res = await httpClient.get(API.payment.apps);
      set({ apps: res.data.data as PaymentApp[], isLoading: false });
    } catch (err) {
      set({ isLoading: false });
      throw err;
    }
  },

  createApp: async (input) => {
    const res = await httpClient.post(API.payment.apps, input);
    await get().fetchApps();
    return res.data.data as CreateAppResult;
  },

  resetAppSecret: async (appId) => {
    const res = await httpClient.post(API.payment.appResetSecret(appId));
    return res.data.data as ResetAppSecretResult;
  },

  toggleAppActive: async (appId) => {
    await httpClient.post(API.payment.appToggleActive(appId));
    await get().fetchApps();
  },
}));
