import { create } from "zustand";
import { httpClient } from "@/shared/services/http-client";
import { API } from "@/shared/services/api-endpoints";
import type { CalendarItem, CalendarKind } from "@/modules/calendar/types";

export interface CalendarQuery {
  from: string;
  to: string;
  picStaffId?: string;
  /** Empty = every kind. */
  kinds: CalendarKind[];
}

/** One string per query, so the page can tell whether what it holds answers what it shows. */
export function calendarQueryKey(q: CalendarQuery): string {
  return [q.from, q.to, q.picStaffId ?? "", [...q.kinds].sort().join(",")].join("|");
}

interface CalendarState {
  items: CalendarItem[];
  /** The query `items` answers (null before the first answer). */
  loadedKey: string | null;
  status: "idle" | "loading" | "ready" | "error";
  fetchCalendar: (q: CalendarQuery) => Promise<void>;
}

// Only the latest request may write: switching months quickly must not let a slow answer for the previous month
// replace the current one.
let generation = 0;

export const useCalendarStore = create<CalendarState>((set) => ({
  items: [],
  loadedKey: null,
  status: "idle",
  fetchCalendar: async (q) => {
    const mine = ++generation;
    set({ status: "loading" });
    try {
      const params: Record<string, string> = { from: q.from, to: q.to };
      if (q.picStaffId) params.picStaffId = q.picStaffId;
      if (q.kinds.length > 0 && q.kinds.length < 3) params.kinds = q.kinds.join(",");
      const res = await httpClient.get(API.calendar, { params });
      if (mine !== generation) return;
      set({ items: (res.data.data.items ?? []) as CalendarItem[], loadedKey: calendarQueryKey(q), status: "ready" });
    } catch (err) {
      if (mine !== generation) return;
      set({ status: "error" });
      throw err;
    }
  },
}));
