import { create } from "zustand";
import { httpClient } from "@/shared/services/http-client";
import { API } from "@/shared/services/api-endpoints";
import { toPaginationMeta, EMPTY_PAGINATION_META, type PaginationMeta, type RawPaginationMeta } from "@/shared/types/pagination";
import { toTransaction, type RawTransaction } from "@/modules/platform-admin/stores/usePlatformAdminStore";
import type { SubscriptionTransaction } from "@/modules/platform-admin/data/types";
import type {
  Announcement,
  AuditEntry,
  AuditFilter,
  DataRequest,
  DataRequestKind,
  ImpersonationResult,
  PlatformMetrics,
  SubscriptionPolicy,
  SupportStatus,
  TenantNote,
  TenantOverview,
} from "@/modules/platform-admin/data/console.types";

export interface AnnouncementInput {
  title: string;
  body: string;
  kind: Announcement["kind"];
  audience: Announcement["audience"];
  audienceValue: string;
  startsAt: string;
  endsAt: string | null;
}

interface ConsoleState {
  metrics: PlatformMetrics | null;
  announcements: Announcement[];
  policy: SubscriptionPolicy | null;
  dataRequests: DataRequest[];
  auditPage: AuditEntry[];
  auditMeta: PaginationMeta;

  fetchMetrics: () => Promise<void>;
  fetchAnnouncements: () => Promise<void>;
  saveAnnouncement: (id: number | null, input: AnnouncementInput) => Promise<void>;
  deleteAnnouncement: (id: number) => Promise<void>;
  fetchPolicy: () => Promise<void>;
  savePolicy: (policy: SubscriptionPolicy) => Promise<void>;
  fetchDataRequests: (openOnly: boolean) => Promise<void>;
  completeDataRequest: (id: number, status: "done" | "rejected", note: string) => Promise<void>;
  fetchAuditPage: (page: number, filter: AuditFilter) => Promise<void>;

  // Per-tenant calls return their data instead of caching it: the 360° page
  // owns that state and always reads fresh (ADR-0009, no client cache).
  fetchTenantOverview: (tenantId: string) => Promise<TenantOverview>;
  fetchTenantNotes: (tenantId: string) => Promise<TenantNote[]>;
  addTenantNote: (tenantId: string, body: string) => Promise<void>;
  fetchTenantDataRequests: (tenantId: string) => Promise<DataRequest[]>;
  addTenantDataRequest: (tenantId: string, input: { kind: DataRequestKind; requestedBy: string; note: string }) => Promise<void>;
  fetchTenantSupport: (tenantId: string) => Promise<SupportStatus>;
  startSupportSession: (tenantId: string, reason: string) => Promise<ImpersonationResult>;
  fetchTenantTransactions: (tenantId: string) => Promise<SubscriptionTransaction[]>;
  fetchTenantAudit: (tenantId: string) => Promise<AuditEntry[]>;
}

function auditParams(filter: AuditFilter): Record<string, string | undefined> {
  return {
    actorType: filter.actorType || undefined,
    actorId: filter.actorId || undefined,
    tenantId: filter.tenantId || undefined,
    outcome: filter.outcome || undefined,
    q: filter.q || undefined,
    from: filter.from || undefined,
    to: filter.to || undefined,
  };
}

export const useConsoleStore = create<ConsoleState>((set, get) => ({
  metrics: null,
  announcements: [],
  policy: null,
  dataRequests: [],
  auditPage: [],
  auditMeta: EMPTY_PAGINATION_META,

  fetchMetrics: async () => {
    const res = await httpClient.get(API.console.metrics);
    set({ metrics: res.data.data as PlatformMetrics });
  },

  fetchAnnouncements: async () => {
    const res = await httpClient.get(API.console.announcements);
    set({ announcements: res.data.data as Announcement[] });
  },

  saveAnnouncement: async (id, input) => {
    if (id === null) await httpClient.post(API.console.announcements, input);
    else await httpClient.patch(API.console.announcement(id), input);
    await get().fetchAnnouncements();
  },

  deleteAnnouncement: async (id) => {
    await httpClient.delete(API.console.announcement(id));
    await get().fetchAnnouncements();
  },

  fetchPolicy: async () => {
    const res = await httpClient.get(API.console.subscriptionPolicy);
    set({ policy: res.data.data as SubscriptionPolicy });
  },

  savePolicy: async (policy) => {
    await httpClient.put(API.console.subscriptionPolicy, policy);
    await get().fetchPolicy();
  },

  fetchDataRequests: async (openOnly) => {
    const res = await httpClient.get(API.console.dataRequests, { params: { open: openOnly ? true : undefined } });
    set({ dataRequests: res.data.data as DataRequest[] });
  },

  completeDataRequest: async (id, status, note) => {
    await httpClient.post(API.console.dataRequestComplete(id), { status, note });
  },

  fetchAuditPage: async (page, filter) => {
    const res = await httpClient.get(API.console.auditLog, { params: { page, ...auditParams(filter) } });
    set({
      auditPage: res.data.data as AuditEntry[],
      auditMeta: toPaginationMeta(res.data.meta as RawPaginationMeta),
    });
  },

  fetchTenantOverview: async (tenantId) => {
    const res = await httpClient.get(API.console.tenantOverview(tenantId));
    return res.data.data as TenantOverview;
  },

  fetchTenantNotes: async (tenantId) => {
    const res = await httpClient.get(API.console.tenantNotes(tenantId));
    return res.data.data as TenantNote[];
  },

  addTenantNote: async (tenantId, body) => {
    await httpClient.post(API.console.tenantNotes(tenantId), { body });
  },

  fetchTenantDataRequests: async (tenantId) => {
    const res = await httpClient.get(API.console.tenantDataRequests(tenantId));
    return res.data.data as DataRequest[];
  },

  addTenantDataRequest: async (tenantId, input) => {
    await httpClient.post(API.console.tenantDataRequests(tenantId), input);
  },

  fetchTenantSupport: async (tenantId) => {
    const res = await httpClient.get(API.console.tenantSupport(tenantId));
    return res.data.data as SupportStatus;
  },

  startSupportSession: async (tenantId, reason) => {
    const res = await httpClient.post(API.console.tenantImpersonate(tenantId), { reason });
    return res.data.data as ImpersonationResult;
  },

  fetchTenantTransactions: async (tenantId) => {
    const res = await httpClient.get(API.billing.transactions, { params: { tenantId, all: true } });
    return (res.data.data as RawTransaction[]).map(toTransaction);
  },

  fetchTenantAudit: async (tenantId) => {
    const res = await httpClient.get(API.console.auditLog, { params: { tenantId, limit: 20 } });
    return res.data.data as AuditEntry[];
  },
}));
