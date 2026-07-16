import { create } from "zustand";
import { httpClient } from "@/shared/services/http-client";
import { API } from "@/shared/services/api-endpoints";
import type { Client, ClientRole } from "@/modules/clients/types";
import type { ClientContactFormValues, ClientCreateFormValues, RepresentativeFormValues } from "@/modules/clients/schemas/client.schema";

interface RawClient {
  id: number;
  projectId: number;
  role: ClientRole;
  relationNote: string;
  name: string;
  phone: string;
  email: string;
  isActive: boolean;
  lastCredentialResetAt: string | null;
}

function toClient(raw: RawClient): Client {
  return {
    id: String(raw.id),
    projectId: String(raw.projectId),
    role: raw.role,
    relationNote: raw.relationNote,
    name: raw.name,
    phone: raw.phone,
    email: raw.email,
    isActive: raw.isActive,
    lastCredentialResetAt: raw.lastCredentialResetAt,
  };
}

interface ClientState {
  clientsByProject: Record<string, Client[]>;
  allClients: Client[];

  fetchClients: (projectId: string) => Promise<void>;
  fetchAllClients: () => Promise<void>;
  createClient: (projectId: string, role: ClientRole, values: ClientCreateFormValues) => Promise<void>;
  updateContact: (projectId: string, clientId: string, values: ClientContactFormValues) => Promise<void>;
  toggleActive: (projectId: string, clientId: string) => Promise<void>;
  resetCredential: (projectId: string, clientId: string, password: string) => Promise<void>;
  replaceRepresentative: (projectId: string, clientId: string, values: RepresentativeFormValues) => Promise<void>;
}

// Backed by the real `clients` module (Fase 4) — clients only exist scoped to
// a project (`GET /clients?projectId=`), so the store keys its cache by
// project id rather than holding one flat list.
export const useClientStore = create<ClientState>((set, get) => ({
  clientsByProject: {},
  allClients: [],

  fetchClients: async (projectId) => {
    const res = await httpClient.get(API.clients.byProject(projectId));
    const list = (res.data.data as RawClient[]).map(toClient);
    set((state) => ({ clientsByProject: { ...state.clientsByProject, [projectId]: list } }));
  },

  // Tenant-wide, unscoped from any single project — powers WO Console's
  // global search, which needs to match client names across all projects.
  fetchAllClients: async () => {
    const res = await httpClient.get(API.clients.base, { params: { all: true } });
    set({ allClients: (res.data.data as RawClient[]).map(toClient) });
  },

  createClient: async (projectId, role, values) => {
    await httpClient.post(API.clients.base, {
      projectId: Number(projectId),
      role,
      relationNote: values.relationNote,
      name: values.name,
      phone: values.phone,
      email: values.email,
      password: values.password,
    });
    await get().fetchClients(projectId);
  },

  updateContact: async (projectId, clientId, values) => {
    await httpClient.patch(API.clients.item(clientId), values);
    await get().fetchClients(projectId);
  },

  toggleActive: async (projectId, clientId) => {
    await httpClient.post(API.clients.toggleActive(clientId));
    await get().fetchClients(projectId);
  },

  resetCredential: async (projectId, clientId, password) => {
    await httpClient.post(API.clients.resetCredential(clientId), { password });
    await get().fetchClients(projectId);
  },

  replaceRepresentative: async (projectId, clientId, values) => {
    await httpClient.post(API.clients.replaceRepresentative(clientId), values);
    await get().fetchClients(projectId);
  },
}));
