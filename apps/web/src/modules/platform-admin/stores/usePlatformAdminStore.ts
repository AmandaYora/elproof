import { create } from "zustand";
import type { AccessLink } from "@/shared/lib/access-link";
import { httpClient } from "@/shared/services/http-client";
import { API } from "@/shared/services/api-endpoints";
import type { PlatformAdmin, SubscriptionTransaction, Tenant } from "@/modules/platform-admin/data/types";
import type { TenantCreateFormValues, TenantFormValues } from "@/modules/platform-admin/schemas/tenant.schema";
import type { CompressedFilePayload } from "@/shared/lib/image-compression";
import type {
  PlatformAdminCreateFormValues,
  PlatformAdminFormValues,
} from "@/modules/platform-admin/schemas/platform-admin.schema";
import { toPaginationMeta, EMPTY_PAGINATION_META, type PaginationMeta, type RawPaginationMeta } from "@/shared/types/pagination";

export interface RegisterTenantResult {
  tenant: Tenant;
  /** The Owner's invitation (accepted with Google). */
  invitation: AccessLink;
  /** Google account that must accept it. */
  email: string;
}

interface OwnerInvitationRaw {
  token: string;
  expiresAt: string;
  email: string;
}

export interface OwnerInviteResult {
  link: AccessLink;
  email: string;
}

/** Manual activation: reason and proof are mandatory (backend validates). */
export interface ManualActivationInput {
  planId: string;
  reason: string;
  proofReference: string;
  note: string;
}

/** Filters beyond search and status on the tenant list (Fase 6). */
export interface TenantListExtraFilter {
  planId?: string;
  source?: string;
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

export interface RawTenant {
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
  trialEndsAt: string | null;
  signupSource: Tenant["signupSource"];
  lifecycle: Tenant["lifecycle"];
  isSuspended: boolean;
  lastCredentialResetAt: string | null;
  brandColorPreset: string;
  hasLogo: boolean;
}

/** Maps the API's tenant row; the Tenant 360° page maps its overview with this too. */
export function toTenant(raw: RawTenant): Tenant {
  return { ...raw, id: String(raw.id), planId: raw.planId !== null ? String(raw.planId) : null };
}

export interface RawTransaction {
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

export function toTransaction(raw: RawTransaction): SubscriptionTransaction {
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
  mfaEnabled: boolean;
}

function toPlatformAdmin(raw: RawPlatformAdmin): PlatformAdmin {
  return { ...raw, id: String(raw.id) };
}

interface PlatformAdminState {
  /** The signed-in admin only: the console reads its own profile, not the whole team. */
  platformAdmins: PlatformAdmin[];
  myTenant: Tenant | null;

  tenantPage: Tenant[];
  tenantPageMeta: PaginationMeta;
  platformAdminPage: PlatformAdmin[];
  platformAdminPageMeta: PaginationMeta;
  transactionPage: SubscriptionTransaction[];
  transactionPageMeta: PaginationMeta;

  fetchPlatformAdmin: (id: string) => Promise<void>;
  fetchMyTenant: () => Promise<void>;

  // Backs the respective Platform Console list-page tables — real
  // server-side pagination + search/filter, separate from the full-roster
  // caches above (used by dropdowns and the "Langganan" self-service page).
  fetchTenantPage: (page: number, search: string, status: string, extra?: TenantListExtraFilter) => Promise<void>;
  fetchPlatformAdminPage: (page: number, search: string, role: string) => Promise<void>;
  fetchTransactionPage: (page: number, status: string) => Promise<void>;

  registerTenant: (values: TenantCreateFormValues) => Promise<RegisterTenantResult>;
  updateTenant: (id: string, values: TenantFormValues) => Promise<void>;
  uploadTenantLogo: (id: string, file: CompressedFilePayload) => Promise<void>;
  toggleTenantSuspension: (id: string) => Promise<void>;
  /** New Owner invitation link (lost Google account, never accepted, …). */
  reinviteTenantOwner: (id: string) => Promise<OwnerInviteResult>;
  activateTenantSubscription: (tenantId: string, input: ManualActivationInput) => Promise<void>;
  // Tenant Owner's own self-service "Bayar Sekarang" — scoped to their own
  // tenant server-side via the JWT claim, not a request parameter. Returns
  // the created charge (Fase 9) — the subscription activates later, once
  // the gateway's webhook confirms payment, not synchronously here.
  paySubscription: (planId: string) => Promise<PaymentCharge>;
  fetchPendingCharge: () => Promise<PaymentCharge | null>;
  cancelPendingCharge: () => Promise<void>;

  registerPlatformAdmin: (values: PlatformAdminCreateFormValues) => Promise<RegisterPlatformAdminResult>;
  updatePlatformAdmin: (id: string, values: PlatformAdminFormValues) => Promise<void>;
  togglePlatformAdminActive: (id: string) => Promise<void>;
  resetPlatformAdminPassword: (id: string, password: string) => Promise<ResetPlatformAdminPasswordResult>;
  resetPlatformAdminMfa: (id: string) => Promise<void>;
}

// Backed by the real `platform` + `billing` modules (Fase 2) — fetch-then-set,
// no client cache (ADR-0009). Passwords are never stored client-side; they
// only ever flow back once, directly in a mutation's response.
export const usePlatformAdminStore = create<PlatformAdminState>((set, get) => ({
  platformAdmins: [],
  myTenant: null,

  tenantPage: [],
  tenantPageMeta: EMPTY_PAGINATION_META,
  platformAdminPage: [],
  platformAdminPageMeta: EMPTY_PAGINATION_META,
  transactionPage: [],
  transactionPageMeta: EMPTY_PAGINATION_META,

  fetchMyTenant: async () => {
    const res = await httpClient.get(API.platform.tenantMe);
    set({ myTenant: toTenant(res.data.data as RawTenant) });
  },

  fetchPlatformAdmin: async (id) => {
    const res = await httpClient.get(API.platform.platformAdmin(id));
    set({ platformAdmins: [toPlatformAdmin(res.data.data as RawPlatformAdmin)] });
  },

  fetchTenantPage: async (page, search, status, extra) => {
    const res = await httpClient.get(API.platform.tenants, {
      params: {
        page,
        search: search || undefined,
        status: status || undefined,
        planId: extra?.planId || undefined,
        source: extra?.source || undefined,
      },
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
    const res = await httpClient.post(API.platform.tenants, values);
    const { tenant, invitation } = res.data.data as { tenant: RawTenant; invitation: OwnerInvitationRaw };
    return { tenant: toTenant(tenant), invitation: { token: invitation.token, expiresAt: invitation.expiresAt, purpose: "staff_invite" }, email: invitation.email };
  },

  updateTenant: async (id, values) => {
    await httpClient.patch(API.platform.tenant(id), values);
  },

  uploadTenantLogo: async (id, file) => {
    await httpClient.put(API.platform.tenantLogo(id), file);
  },

  toggleTenantSuspension: async (id) => {
    await httpClient.post(API.platform.tenantToggleSuspension(id));
  },

  reinviteTenantOwner: async (id) => {
    const res = await httpClient.post(API.platform.tenantOwnerInvite(id));
    const inv = res.data.data as OwnerInvitationRaw;
    return { link: { token: inv.token, expiresAt: inv.expiresAt, purpose: "staff_invite" }, email: inv.email };
  },

  activateTenantSubscription: async (tenantId, input) => {
    await httpClient.post(API.platform.tenantActivateSubscription(tenantId), {
      planId: Number(input.planId),
      reason: input.reason,
      proofReference: input.proofReference,
      note: input.note,
    });
  },

  paySubscription: async (planId) => {
    const res = await httpClient.post(API.platform.subscriptionsPay, { planId: Number(planId) });
    return res.data.data as PaymentCharge;
  },

  // Lets the Owner re-view a still-pending charge (e.g. after accidentally
  // closing its QR modal) — `data` is null when there's nothing pending,
  // which is the common case, not an error.
  fetchPendingCharge: async () => {
    const res = await httpClient.get(API.platform.subscriptionsPendingCharge);
    return (res.data.data as PaymentCharge | null) ?? null;
  },

  cancelPendingCharge: async () => {
    await httpClient.post(API.platform.subscriptionsCancelPendingCharge);
    await get().fetchTransactionPage(1, "");
  },

  registerPlatformAdmin: async (values) => {
    const res = await httpClient.post(API.platform.platformAdmins, values);
    const { admin, username } = res.data.data as { admin: RawPlatformAdmin; username: string };
    return { admin: toPlatformAdmin(admin), username, password: values.password };
  },

  updatePlatformAdmin: async (id, values) => {
    await httpClient.patch(API.platform.platformAdmin(id), values);
  },

  togglePlatformAdminActive: async (id) => {
    await httpClient.post(API.platform.platformAdminToggleActive(id));
  },

  resetPlatformAdminPassword: async (id, password) => {
    const res = await httpClient.post(API.platform.platformAdminResetPassword(id), { password });
    const { username } = res.data.data as { username: string };
    return { username, password };
  },

  resetPlatformAdminMfa: async (id) => {
    await httpClient.post(API.platform.platformAdminResetMfa(id));
  },
}));
