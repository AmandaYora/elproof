import { create } from "zustand";
import axios from "axios";
import { httpClient } from "@/shared/services/http-client";
import { API } from "@/shared/services/api-endpoints";
import { onSessionReset } from "@/shared/lib/session-reset";
import { todayISO } from "@/shared/lib/formatters";
import type { EvidenceType } from "@/modules/projects/types";
import type { ClientTask, PortalHome } from "@/modules/client-portal/types";

export type HomeLoad = "idle" | "loading" | "ready" | "no-project" | "error";

interface ClientHomeState {
  home: PortalHome | null;
  status: HomeLoad;
  /** One call for the whole landing page and the project id every other page needs. */
  fetch: () => Promise<void>;
  /**
   * The couple marks one of their tasks done. Shown done at once; put back (and the error rethrown for a toast) when
   * the server refuses.
   */
  completeTask: (taskId: string) => Promise<void>;
  reset: () => void;
}

interface RawHome extends Omit<PortalHome, "project" | "recentUpdates" | "actions" | "clientTasks"> {
  project: Omit<PortalHome["project"], "id"> & { id: number };
  actions?: PortalHome["actions"] | null;
  recentUpdates?: Array<Omit<PortalHome["recentUpdates"][number], "proof"> & { proof: { evidenceId: number; type: string; fileName: string } | null }> | null;
  clientTasks?: Array<Omit<ClientTask, "id"> & { id: number | string }> | null;
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
    // Absent on an older API: stays undefined so the section is hidden rather than shown empty.
    clientTasks: raw.clientTasks ? raw.clientTasks.map((t) => ({ ...t, id: String(t.id), completedDate: t.completedDate ?? null })) : undefined,
  };
}

function withTask(home: PortalHome, taskId: string, patch: Partial<ClientTask>): PortalHome {
  return { ...home, clientTasks: home.clientTasks?.map((t) => (t.id === taskId ? { ...t, ...patch } : t)) };
}

export const useClientHomeStore = create<ClientHomeState>((set, get) => ({
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
  completeTask: async (taskId) => {
    const home = get().home;
    const before = home?.clientTasks?.find((t) => t.id === taskId);
    if (!home || !before) return;
    set({ home: withTask(home, taskId, { status: "Completed", completedDate: todayISO() }) });
    try {
      const res = await httpClient.post(API.projects.milestoneComplete(home.project.id, taskId));
      const saved = res.data?.data as { completedDate?: string | null } | undefined;
      const now = get().home;
      if (now && saved?.completedDate) set({ home: withTask(now, taskId, { completedDate: saved.completedDate }) });
    } catch (err) {
      const now = get().home;
      if (now) set({ home: withTask(now, taskId, { status: before.status, completedDate: before.completedDate }) });
      throw err;
    }
  },
  reset: () => set({ home: null, status: "idle" }),
}));

// Never leave one person's project in memory for the next person on a shared phone.
onSessionReset(() => useClientHomeStore.getState().reset());
