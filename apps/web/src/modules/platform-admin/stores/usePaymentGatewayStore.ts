import { create } from "zustand";
import { httpClient } from "@/shared/services/http-client";
import { API } from "@/shared/services/api-endpoints";

export interface GatewayConfig {
  activeProvider: string;
  isSandbox: boolean;
  tripayMerchantCode: string;
  hasTripayApiKey: boolean;
  hasTripayPrivateKey: boolean;
}

export interface UpdateGatewayConfigInput {
  activeProvider: string;
  isSandbox: boolean;
  tripayMerchantCode: string;
  // Empty string means "leave the existing stored value unchanged" — the
  // backend is write-only for these (see MODULE_PAYMENT.md §8).
  tripayApiKey: string;
  tripayPrivateKey: string;
}

interface PaymentGatewayState {
  config: GatewayConfig | null;
  isLoading: boolean;
  fetchConfig: () => Promise<void>;
  updateConfig: (input: UpdateGatewayConfigInput) => Promise<void>;
}

// Backed by the real `payment` module (Fase 9) — fetch-then-set, no client
// cache (ADR-0009), same pattern as every other admin-config store.
export const usePaymentGatewayStore = create<PaymentGatewayState>((set, get) => ({
  config: null,
  isLoading: false,

  fetchConfig: async () => {
    set({ isLoading: true });
    try {
      const res = await httpClient.get(API.payment.gatewayConfig);
      set({ config: res.data.data as GatewayConfig, isLoading: false });
    } catch (err) {
      set({ isLoading: false });
      throw err;
    }
  },

  updateConfig: async (input) => {
    await httpClient.patch(API.payment.gatewayConfig, input);
    await get().fetchConfig();
  },
}));
