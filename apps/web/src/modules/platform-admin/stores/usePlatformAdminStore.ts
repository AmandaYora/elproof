import { create } from "zustand";
import { httpClient } from "@/shared/services/http-client";
import { API } from "@/shared/services/api-endpoints";
import type { PlatformAdmin, SubscriptionTransaction, Tenant } from "@/modules/platform-admin/data/types";
import type { TenantCreateFormValues, TenantFormValues } from "@/modules/platform-admin/schemas/tenant.schema";
import type {
  PlatformAdminCreateFormValues,
  PlatformAdminFormValues,
} from "@/modules/platform-admin/schemas/platform-admin.schema";
import { toPaginationMeta, EMPTY_PAGINATION_META, type PaginationMeta, type RawPaginationMeta } from "@/shared/types/pagination";

export interface RegisterTenantResult {
  tenant: Tenant;
  username: string;
  password: string;
}

export interface ResetTenantCredentialResult {
  username: string;
  password: string;
}

export interface RegisterPlatformAdminResult {
  admin: PlatformAdmin;
  username: string;
  password: string;
}

export interface ResetPlatformAdminPasswordResult {
  username: string;
  password: string;
}

// Returned by Pay (Fase 9) — a real charge created at the configured
// gateway (QRIS by default). The subscription is NOT active yet at this
// point; it only activates once the gateway's webhook confirms payment.
export interface PaymentCharge {
  orderRef: string;
  providerRef: string;
  channel: string;
  qrImageUrl: string;
  payCode: string;
  checkoutUrl: string;
  amount: number;
  feeAmount: number;
  expiresAt: string;
  status: string;
}

interface RawTenant {
  id: number;
  businessName: string;
  ownerName: string;
  username: string;
  email: string;
  phone: string;
  city: string;
  joinedAt: string;
  planId: number | null;
  subscriptionStatus: Tenant["subscriptionStatus"];
  subscriptionExpiresAt: string | null;
  isSuspended: boolean;
  lastCredentialResetAt: string | null;
}

function toTenant(raw: RawTenant): Tenant {
  return { ...raw, id: String(raw.id), planId: raw.planId !== null ? String(raw.planId) : null };
}

interface RawTransaction {
  id: number;
  tenantId: number;
  type: SubscriptionTransaction["type"];
  amount: number;
  paymentMethod: string;
  paymentReference: string;
  status: SubscriptionTransaction["status"];
  createdAt: string;
  paidAt: string | null;
}

function toTransaction(raw: RawTransaction): SubscriptionTransaction {
  return { ...raw, id: String(raw.id), tenantId: String(raw.tenantId) };
}

interface RawPlatformAdmin {
  id: number;
  name: string;
  title: string;
  role: PlatformAdmin["role"];
  username: string;
  email: string;
  phone: string;
  isActive: boolean;
}

function toPlatformAdmin(raw: RawPlatformAdmin): PlatformAdmin {
  return { ...raw, id: String(raw.id) };
}

interface PlatformAdminState {
  tenants: Tenant[];
  subscriptionTransactions: SubscriptionTransaction[];
  platformAdmins: PlatformAdmin[];
  myTenant: Tenant | null;

  tenantPage: Tenant[];
  tenantPageMeta: PaginationMeta;
  platformAdminPage: PlatformAdmin[];
  platformAdminPageMeta: PaginationMeta;
  transactionPage: SubscriptionTransaction[];
  transactionPageMeta: PaginationMeta;

  fetchTenants: () => Promise<void>;
  fetchPlatformAdmins: () => Promise<void>;
  fetchTransactions: () => Promise<void>;
  fetchMyTenant: () => Promise<void>;

  // Backs the respective Platform Console list-page tables — real
  // server-side pagination + search/filter, separate from the full-roster
  // caches above (used by dropdowns and the "Langganan" self-service page).
  fetchTenantPage: (page: number, search: string, status: string) => Promise<void>;
  fetchPlatformAdminPage: (page: number, search: string, role: string) => Promise<void>;
  fetchTransactionPage: (page: number, status: string) => Promise<void>;

  registerTenant: (values: TenantCreateFormValues) => Promise<RegisterTenantResult>;
  updateTenant: (id: string, values: TenantFormValues) => Promise<void>;
  toggleTenantSuspension: (id: string) => Promise<void>;
  resetTenantCredential: (id: string, password: string) => Promise<ResetTenantCredentialResult>;
  activateTenantSubscription: (tenantId: string, planId: string) => Promise<void>;
  // Tenant Owner's own self-service "Bayar Sekarang" — scoped to their own
  // tenant server-side via the JWT claim, not a request parameter. Returns
  // the created charge (Fase 9) — the subscription activates later, once
  // the gateway's webhook confirms payment, not synchronously here.
  paySubscription: (planId: string) => Promise<PaymentCharge>;

  registerPlatformAdmin: (values: PlatformAdminCreateFormValues) => Promise<RegisterPlatformAdminResult>;
  updatePlatformAdmin: (id: string, values: PlatformAdminFormValues) => Promise<void>;
  togglePlatformAdminActive: (id: string) => Promise<void>;
  resetPlatformAdminPassword: (id: string, password: string) => Promise<ResetPlatformAdminPasswordResult>;
}

// Backed by the real `platform` + `billing` modules (Fase 2) — fetch-then-set,
// no client cache (ADR-0009). Passwords are never stored client-side; they
// only ever flow back once, directly in a mutation's response.
export const usePlatformAdminStore = create<PlatformAdminState>((set, get) => ({
  tenants: [],
  subscriptionTransactions: [],
  platformAdmins: [],
  myTenant: null,

  tenantPage: [],
  tenantPageMeta: EMPTY_PAGINATION_META,
  platformAdminPage: [],
  platformAdminPageMeta: EMPTY_PAGINATION_META,
  transactionPage: [],
  transactionPageMeta: EMPTY_PAGINATION_META,

  fetchTenants: async () => {
    const res = await httpClient.get(API.platform.tenants, { params: { all: true } });
    set({ tenants: (res.data.data as RawTenant[]).map(toTenant) });
  },

  fetchMyTenant: async () => {
    const res = await httpClient.get(API.platform.tenantMe);
    set({ myTenant: toTenant(res.data.data as RawTenant) });
  },

  fetchPlatformAdmins: async () => {
    const res = await httpClient.get(API.platform.platformAdmins, { params: { all: true } });
    set({ platformAdmins: (res.data.data as RawPlatformAdmin[]).map(toPlatformAdmin) });
  },

  fetchTransactions: async () => {
    const res = await httpClient.get(API.billing.transactions, { params: { all: true } });
    set({ subscriptionTransactions: (res.data.data as RawTransaction[]).map(toTransaction) });
  },

  fetchTenantPage: async (page, search, status) => {
    const res = await httpClient.get(API.platform.tenants, {
      params: { page, search: search || undefined, status: status || undefined },
    });
    set({
      tenantPage: (res.data.data as RawTenant[]).map(toTenant),
      tenantPageMeta: toPaginationMeta(res.data.meta as RawPaginationMeta),
    });
  },

  fetchPlatformAdminPage: async (page, search, role) => {
    const res = await httpClient.get(API.platform.platformAdmins, {
      params: { page, search: search || undefined, role: role || undefined },
    });
    set({
      platformAdminPage: (res.data.data as RawPlatformAdmin[]).map(toPlatformAdmin),
      platformAdminPageMeta: toPaginationMeta(res.data.meta as RawPaginationMeta),
    });
  },

  fetchTransactionPage: async (page, status) => {
    const res = await httpClient.get(API.billing.transactions, { params: { page, status: status || undefined } });
    set({
      transactionPage: (res.data.data as RawTransaction[]).map(toTransaction),
      transactionPageMeta: toPaginationMeta(res.data.meta as RawPaginationMeta),
    });
  },

  registerTenant: async (values) => {
    const res = await httpClient.post(API.platform.tenants, { ...values, planId: Number(values.planId) });
    const { tenant, username } = res.data.data as { tenant: RawTenant; username: string };
    await Promise.all([get().fetchTenants(), get().fetchTransactions()]);
    return { tenant: toTenant(tenant), username, password: values.password };
  },

  updateTenant: async (id, values) => {
    await httpClient.patch(API.platform.tenant(id), values);
    await get().fetchTenants();
  },

  toggleTenantSuspension: async (id) => {
    await httpClient.post(API.platform.tenantToggleSuspension(id));
    await get().fetchTenants();
  },

  resetTenantCredential: async (id, password) => {
    const res = await httpClient.post(API.platform.tenantResetCredential(id), { password });
    await get().fetchTenants();
    return res.data.data as ResetTenantCredentialResult;
  },

  activateTenantSubscription: async (tenantId, planId) => {
    await httpClient.post(API.platform.tenantActivateSubscription(tenantId), { planId: Number(planId) });
    await Promise.all([get().fetchTenants(), get().fetchTransactions()]);
  },

  paySubscription: async (planId) => {
    const res = await httpClient.post(API.platform.subscriptionsPay, { planId: Number(planId) });
    return res.data.data as PaymentCharge;
  },

  registerPlatformAdmin: async (values) => {
    const res = await httpClient.post(API.platform.platformAdmins, values);
    const { admin, username } = res.data.data as { admin: RawPlatformAdmin; username: string };
    await get().fetchPlatformAdmins();
    return { admin: toPlatformAdmin(admin), username, password: values.password };
  },

  updatePlatformAdmin: async (id, values) => {
    await httpClient.patch(API.platform.platformAdmin(id), values);
    await get().fetchPlatformAdmins();
  },

  togglePlatformAdminActive: async (id) => {
    await httpClient.post(API.platform.platformAdminToggleActive(id));
    await get().fetchPlatformAdmins();
  },

  resetPlatformAdminPassword: async (id, password) => {
    const res = await httpClient.post(API.platform.platformAdminResetPassword(id), { password });
    return res.data.data as ResetPlatformAdminPasswordResult;
  },
}));
