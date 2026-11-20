import { create } from "zustand";
import { httpClient } from "@/shared/services/http-client";
import { API } from "@/shared/services/api-endpoints";
import { ifMatch } from "@/shared/lib/concurrency";
import type { EventKind } from "@/modules/event-kinds/types";

interface RawEventKind extends Omit<EventKind, "id" | "quotationCount"> {
  id: number;
  quotationCount?: number;
}

function toEventKind(raw: RawEventKind): EventKind {
  return { ...raw, id: String(raw.id), quotationCount: raw.quotationCount ?? 0 };
}

interface EventKindState {
  /** Every kind of the workspace in its own order (a short list: no paging). */
  kinds: EventKind[];
  fetchKinds: () => Promise<EventKind[]>;
  createKind: (name: string) => Promise<void>;
  /** Saves a new name for the kind as it was read (If-Match): a 409 means someone changed it meanwhile. */
  renameKind: (kind: EventKind, name: string) => Promise<void>;
  /** Sets the state explicitly (ADR-0036): a double click or a retry cannot flip it back. */
  setKindActive: (id: string, isActive: boolean) => Promise<void>;
  /** Refused with 409 while a package covers the kind. */
  deleteKind: (id: string) => Promise<void>;
}

export const useEventKindStore = create<EventKindState>((set, get) => ({
  kinds: [],

  fetchKinds: async () => {
    const res = await httpClient.get(API.offers.eventKinds);
    const kinds = (res.data.data as RawEventKind[]).map(toEventKind);
    set({ kinds });
    return kinds;
  },

  createKind: async (name) => {
    await httpClient.post(API.offers.eventKinds, { name });
    await get().fetchKinds();
  },

  renameKind: async (kind, name) => {
    await httpClient.patch(API.offers.eventKind(kind.id), { name }, { headers: ifMatch(kind.updatedAt) });
    await get().fetchKinds();
  },

  setKindActive: async (id, isActive) => {
    await httpClient.patch(API.offers.eventKind(id), { isActive });
    await get().fetchKinds();
  },

  deleteKind: async (id) => {
    await httpClient.delete(API.offers.eventKind(id));
    await get().fetchKinds();
  },
}));
