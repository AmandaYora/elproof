import { create } from "zustand";
import { httpClient } from "@/shared/services/http-client";
import { API } from "@/shared/services/api-endpoints";
import type { Client, ClientRole } from "@/modules/clients/types";
import type { ClientContactFormValues, ClientCreateFormValues, RepresentativeFormValues } from "@/modules/clients/schemas/client.schema";
import type { AccessLink } from "@/shared/lib/access-link";
import { toPaginationMeta, EMPTY_PAGINATION_META, type PaginationMeta, type RawPaginationMeta } from "@/shared/types/pagination";

interface RawClient {
  id: number;
  projectId: number;
  role: ClientRole;
  username: string;
  relationNote: string;
  name: string;
  phone: string;
  email: string;
  isActive: boolean;
  lastCredentialResetAt: string | null;
  loginStatus?: Client["loginStatus"];
  accessRoleId: number;
  accessRoleName?: string;
  projectName?: string;
  projectEventDate?: string;
}

function toClient(raw: RawClient): Client {
  return {
    id: String(raw.id),
    projectId: String(raw.projectId),
    role: raw.role,
    username: raw.username,
    relationNote: raw.relationNote,
    name: raw.name,
    phone: raw.phone,
    email: raw.email,
    isActive: raw.isActive,
    lastCredentialResetAt: raw.lastCredentialResetAt,
    loginStatus: raw.loginStatus,
    accessRoleId: raw.accessRoleId,
    accessRoleName: raw.accessRoleName,
    projectName: raw.projectName,
    projectEventDate: raw.projectEventDate,
  };
}

interface ClientWithLink {
  client: RawClient;
  accessLink: AccessLink;
}

interface ClientState {
  clientsByProject: Record<string, Client[]>;
  clientPage: Client[];
  clientPageMeta: PaginationMeta;

  fetchClients: (projectId: string) => Promise<void>;
  /** One page of the tenant-wide list. `params` is the list's own query: page, limit, sort, q, role, loginStatus, projectId. */
  fetchClientPage: (params: Record<string, string | number>) => Promise<void>;
  /** Creates the client and returns their invitation link (they set their own password). */
  createClient: (projectId: string, role: ClientRole, values: ClientCreateFormValues) => Promise<{ client: Client; link: AccessLink }>;
  updateContact: (projectId: string, clientId: string, values: ClientContactFormValues) => Promise<void>;
  /** Sets the state explicitly (ADR-0036): a double click or a retry cannot flip it back. */
  setActive: (projectId: string, clientId: string, isActive: boolean) => Promise<void>;
  /** Returns the function that undoes the delete (the server keeps the client 24 h). */
  deleteClient: (projectId: string, clientId: string) => Promise<() => Promise<void>>;
  /** New invitation (no password yet) or 24-hour password-reset link. */
  issueAccessLink: (projectId: string, clientId: string) => Promise<AccessLink>;
  /** Replaces the person; the old one loses access, the new one gets an invitation. */
  replaceRepresentative: (projectId: string, clientId: string, values: RepresentativeFormValues) => Promise<{ client: Client; link: AccessLink }>;
}

// Backed by the real `clients` module (Fase 4) — clients only exist scoped to
// a project (`GET /clients?projectId=`), so the store keys its cache by
// project id rather than holding one flat list.
export const useClientStore = create<ClientState>((set, get) => ({
  clientsByProject: {},
  clientPage: [],
  clientPageMeta: EMPTY_PAGINATION_META,

  fetchClients: async (projectId) => {
    const res = await httpClient.get(API.clients.byProject(projectId));
    const list = (res.data.data as RawClient[]).map(toClient);
    set((state) => ({ clientsByProject: { ...state.clientsByProject, [projectId]: list } }));
  },

  // Tenant-wide flat list for ClientListPage: the server pages, sorts and filters it, and every row says which
  // project it belongs to. Always sent with page/limit, which selects the paginated form of the endpoint.
  fetchClientPage: async (params) => {
    const res = await httpClient.get(API.clients.base, { params });
    set({
      clientPage: (res.data.data as RawClient[]).map(toClient),
      clientPageMeta: toPaginationMeta(res.data.meta as RawPaginationMeta),
    });
  },

  createClient: async (projectId, role, values) => {
    const res = await httpClient.post(API.clients.base, {
      projectId: Number(projectId),
      role,
      relationNote: values.relationNote,
      name: values.name,
      phone: values.phone,
      email: values.email,
      username: values.username,
      accessRoleId: values.accessRoleId ?? 0,
    });
    const data = res.data.data as ClientWithLink;
    // The one-time link must reach the screen even if the list refresh fails.
    await get().fetchClients(projectId).catch(() => undefined);
    return { client: toClient(data.client), link: data.accessLink };
  },

  updateContact: async (projectId, clientId, values) => {
    await httpClient.patch(API.clients.item(clientId), values);
    await get().fetchClients(projectId);
  },

  setActive: async (projectId, clientId, isActive) => {
    await httpClient.patch(API.clients.item(clientId), { isActive });
    await get().fetchClients(projectId);
  },

  // Hard delete — for clearing a client stuck with no working login
  // credential (e.g. left behind before Create's compensating rollback
  // existed). Frees up its role slot on the project so a replacement can be
  // added; every other action in this store is a soft toggle instead.
  deleteClient: async (projectId, clientId) => {
    await httpClient.delete(API.clients.item(clientId));
    await get().fetchClients(projectId);
    // The server keeps the client (login switched off) for 24 hours, so this really restores it.
    return async () => {
      await httpClient.post(API.clients.restore(clientId));
      await get().fetchClients(projectId);
    };
  },

  issueAccessLink: async (projectId, clientId) => {
    const res = await httpClient.post(API.clients.accessLink(clientId));
    await get().fetchClients(projectId).catch(() => undefined);
    return res.data.data as AccessLink;
  },

  replaceRepresentative: async (projectId, clientId, values) => {
    const res = await httpClient.post(API.clients.replaceRepresentative(clientId), values);
    const data = res.data.data as ClientWithLink;
    // The one-time link must reach the screen even if the list refresh fails.
    await get().fetchClients(projectId).catch(() => undefined);
    return { client: toClient(data.client), link: data.accessLink };
  },
}));
