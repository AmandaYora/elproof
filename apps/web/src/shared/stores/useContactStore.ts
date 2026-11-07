import { create } from "zustand";
import { httpClient } from "@/shared/services/http-client";
import { API } from "@/shared/services/api-endpoints";

/** GET /tenants/me/contact — how to reach the WO and, for those who may pay, where to pay. */
export interface TenantContact {
  businessName: string;
  /** Digits with the country code, e.g. 62812… */
  whatsapp: string;
  city: string;
  address: string;
  bankName?: string;
  bankAccountNumber?: string;
  bankAccountHolder?: string;
}

interface ContactState {
  contact: TenantContact | null;
  /** The last load failed (so "no account shared" cannot be told from "could not load"). */
  failed: boolean;
  fetch: () => Promise<void>;
  reset: () => void;
}

export const useContactStore = create<ContactState>((set) => ({
  contact: null,
  failed: false,
  fetch: async () => {
    // Never show the previous person's WO while this one loads or fails.
    set({ contact: null, failed: false });
    try {
      const res = await httpClient.get(API.platform.tenantMeContact);
      set({ contact: res.data.data as TenantContact });
    } catch {
      set({ failed: true });
    }
  },
  reset: () => set({ contact: null, failed: false }),
}));
