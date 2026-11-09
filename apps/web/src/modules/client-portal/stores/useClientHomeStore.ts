import { create } from "zustand";
import axios from "axios";
import { httpClient } from "@/shared/services/http-client";
import { API } from "@/shared/services/api-endpoints";
import { onSessionReset } from "@/shared/lib/session-reset";
import type { EvidenceType } from "@/modules/projects/types";
import type { PortalHome } from "@/modules/client-portal/types";

export type HomeLoad = "idle" | "loading" | "ready" | "no-project" | "error";

interface ClientHomeState {
  home: PortalHome | null;
  status: HomeLoad;
  /** One call for the whole landing page and the project id every other page needs. */
  fetch: () => Promise<void>;
  reset: () => void;
}

interface RawHome extends Omit<PortalHome, "project" | "recentUpdates" | "actions"> {
  project: Omit<PortalHome["project"], "id"> & { id: number };
  actions?: PortalHome["actions"] | null;
  recentUpdates?: Array<Omit<PortalHome["recentUpdates"][number], "proof"> & { proof: { evidenceId: number; type: string; fileName: string } | null }> | null;
}

function toHome(raw: RawHome): PortalHome {
  return {
    ...raw,
    project: { ...raw.project, id: String(raw.project.id) },
    actions: raw.actions ?? [],
    recentUpdates: (raw.recentUpdates ?? []).map((u) => ({
      ...u,
      proof: u.proof ? { evidenceId: String(u.proof.evidenceId), type: u.proof.type as EvidenceType, fileName: u.proof.fileName } : null,
    })),
  };
}

export const useClientHomeStore = create<ClientHomeState>((set) => ({
  home: null,
  status: "idle",
  fetch: async () => {
    // Never show a previous person's project while this one loads.
    set({ status: "loading", home: null });
    try {
      const res = await httpClient.get(API.projects.meHome);
      set({ home: toHome(res.data.data as RawHome), status: "ready" });
    } catch (err) {
      // Not signed in is handled by the HTTP client. No project yet is a message, anything else a retry.
      const code = axios.isAxiosError(err) ? err.response?.status : undefined;
      set({ status: code === 403 || code === 404 ? "no-project" : "error" });
    }
  },
  reset: () => set({ home: null, status: "idle" }),
}));

// Never leave one person's project in memory for the next person on a shared phone.
onSessionReset(() => useClientHomeStore.getState().reset());
