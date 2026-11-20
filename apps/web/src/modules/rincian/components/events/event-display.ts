import type { z } from "zod";
import { formatDate } from "@/shared/lib/formatters";
import type { eventSchema, EventFormValues, EventSessionFormValues } from "@/modules/rincian/schemas/event.schema";
import type { EventInput, EventLayout, RincianEvent } from "@/modules/rincian/types";

/** What a form holds for one event before the schema has filled its defaults. */
export type EventDraftInput = z.input<typeof eventSchema>;

/** A venue as the event screens need it: who it is and how many people it takes per layout. */
export interface VenueCaps {
  venueId: string;
  name: string;
  city: string | null;
  capacityStanding: number | null;
  capacitySeated: number | null;
}

const WEEKDAYS = ["Minggu", "Senin", "Selasa", "Rabu", "Kamis", "Jumat", "Sabtu"];

/** "Jumat, 13 Mar 2027" (DESAIN §5.1). */
export function formatEventDay(iso: string): string {
  if (!iso) return "Tanggal belum ditentukan";
  const [y, m, d] = iso.split("T")[0].split("-").map(Number);
  const weekday = WEEKDAYS[new Date(Date.UTC(y, m - 1, d)).getUTCDay()];
  return `${weekday}, ${formatDate(iso)}`;
}

/** "13" and "Mar" for the date tile. */
export function dayAndMonth(iso: string): { day: string; month: string } {
  const [, , d] = iso.split("T")[0].split("-").map(Number);
  return { day: String(d), month: formatDate(iso).split(" ")[1] ?? "" };
}

/** "08.00": Indonesian clock notation. */
export function formatClock(t: string | null | undefined): string {
  return t ? t.slice(0, 5).replace(":", ".") : "";
}

/** "08.00–10.00", "mulai 08.00", "sampai 10.00", or "" when neither is known. */
export function formatTimeRange(start: string | null | undefined, end: string | null | undefined): string {
  const s = formatClock(start);
  const e = formatClock(end);
  if (s && e) return `${s}–${e}`;
  if (s) return `mulai ${s}`;
  if (e) return `sampai ${e}`;
  return "";
}

export function formatCount(n: number): string {
  return n.toLocaleString("id-ID");
}

export function capacityFor(venue: VenueCaps | null | undefined, layout: EventLayout): number | null {
  if (!venue) return null;
  return layout === "Standing" ? venue.capacityStanding : venue.capacitySeated;
}

export function otherLayout(layout: EventLayout): EventLayout {
  return layout === "Standing" ? "Seated" : "Standing";
}

/** The largest number of guests in one session: what has to fit the venue. */
export function maxSessionGuests(sessions: Array<{ guests?: unknown }>): number {
  return sessions.reduce<number>((max, s) => Math.max(max, Number(s.guests) || 0), 0);
}

export function totalGuests(sessions: Array<{ guests?: unknown }>): number {
  return sessions.reduce<number>((sum, s) => sum + (Number(s.guests) || 0), 0);
}

export function emptySession(): EventSessionFormValues {
  return { label: "", startTime: "", endTime: "", guests: 0 };
}

export function emptyEventDraft(): EventFormValues {
  return { eventKindId: "", eventDate: "", startTime: "", endTime: "", venueId: "", layout: "Standing", sessions: [emptySession()] };
}

export function draftFromEvent(e: RincianEvent): EventFormValues {
  return {
    eventKindId: e.eventKindId,
    eventDate: e.eventDate,
    startTime: e.startTime ?? "",
    endTime: e.endTime ?? "",
    venueId: e.venue?.venueId ?? "",
    layout: e.layout,
    sessions:
      e.sessions.length > 0
        ? e.sessions.map((s) => ({ label: s.label, startTime: s.startTime ?? "", endTime: s.endTime ?? "", guests: s.guests }))
        : [emptySession()],
  };
}

/** Fills what the form may still hold as undefined (a value the schema defaults). */
export function toDraft(v: EventDraftInput): EventFormValues {
  return {
    eventKindId: v.eventKindId,
    eventDate: v.eventDate,
    startTime: v.startTime ?? "",
    endTime: v.endTime ?? "",
    venueId: v.venueId ?? "",
    layout: v.layout,
    sessions: v.sessions.map((s) => ({ label: s.label ?? "", startTime: s.startTime ?? "", endTime: s.endTime ?? "", guests: Number(s.guests) || 0 })),
  };
}

export function toEventInput(d: EventFormValues): EventInput {
  return {
    eventKindId: d.eventKindId,
    eventDate: d.eventDate,
    startTime: d.startTime || null,
    endTime: d.endTime || null,
    venueId: d.venueId,
    layout: d.layout,
    sessions: d.sessions.map((s) => ({ label: s.label.trim(), startTime: s.startTime || null, endTime: s.endTime || null, guests: s.guests })),
  };
}

/**
 * The API names a row "sessions.1.guests" (or "events.0.sessions.1.guests"); the forms name it "sessions[1].guests",
 * the key a Zod issue gets. One spelling, so a server message lands on the same input as a client one.
 */
export function bracketKey(key: string): string {
  return key.replace(/\.(\d+)(?=\.|$)/g, "[$1]");
}

/** Splits session `index` into two with the guests shared evenly; the new one comes right after it, its times empty. */
export function splitSession(draft: EventFormValues, index: number): EventFormValues {
  const s = draft.sessions[index];
  if (!s) return draft;
  const first = Math.ceil(s.guests / 2);
  const sessions = [...draft.sessions];
  sessions.splice(index, 1, { ...s, guests: first }, { label: "", startTime: "", endTime: "", guests: s.guests - first });
  return { ...draft, sessions };
}

/** Session indexes the server refused for capacity ("sessions[2].guests"). */
export function refusedSessions(keys: string[], prefix = ""): number[] {
  const re = new RegExp(`^${prefix.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")}sessions\\[(\\d+)\\]\\.guests$`);
  return keys.flatMap((k) => {
    const m = re.exec(k);
    return m ? [Number(m[1])] : [];
  });
}

/**
 * Where a venue's numbers come from: a venue the project already uses keeps the numbers copied when it was chosen
 * (the server checks against those), any other comes from the directory.
 */
export function venueLookup(projectEvents: RincianEvent[], directory: VenueCaps[]): (venueId: string) => VenueCaps | null {
  const fromProject = new Map<string, VenueCaps>();
  for (const e of projectEvents) {
    if (e.venue?.venueId && !fromProject.has(e.venue.venueId)) {
      fromProject.set(e.venue.venueId, {
        venueId: e.venue.venueId,
        name: e.venue.name,
        city: e.venue.city,
        capacityStanding: e.venue.capacityStanding,
        capacitySeated: e.venue.capacitySeated,
      });
    }
  }
  const fromDirectory = new Map(directory.map((v) => [v.venueId, v]));
  return (venueId) => (venueId ? (fromProject.get(venueId) ?? fromDirectory.get(venueId) ?? null) : null);
}

/**
 * Where the project's events happen, as the API's `location` says it: "Akad: Masjid Al-Ikhlas · Resepsi: Gedung
 * Melati", or just the name when every event with a venue uses the same one; "" when none has a venue. Lets the page
 * header follow an event write without asking the server again.
 */
export function locationLine(events: RincianEvent[]): string {
  const withVenue = events.filter((e) => e.venue?.name);
  const distinct = new Set(withVenue.map((e) => e.venue!.name));
  if (distinct.size === 1) return [...distinct][0];
  return withVenue.map((e) => `${e.kindName}: ${e.venue!.name}`).join(" · ");
}

/** The API's `eventsLine`: "2 acara: Akad 13 Mar · Resepsi 14 Mar"; "" for a single event. */
export function eventsLine(events: RincianEvent[]): string {
  if (events.length < 2) return "";
  const parts = events.map((e) => {
    const { day, month } = dayAndMonth(e.eventDate);
    return `${e.kindName} ${day} ${month}`;
  });
  return `${events.length} acara: ${parts.join(" · ")}`;
}
