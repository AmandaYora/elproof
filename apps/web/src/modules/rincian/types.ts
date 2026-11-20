/**
 * The types the rincian building blocks work with. They describe data, not where it comes from: a project feeds them
 * today (modules/projects re-exports the event and supplier ones under its own names).
 */

export type EngagementStatus =
  | "Planned"
  | "Negotiation"
  | "Booked"
  | "DP Paid"
  | "In Progress"
  | "Fully Paid"
  | "Ready"
  | "Completed"
  | "Cancelled";

/** How guests stand at an event's venue; the venue's capacity for that layout is the limit per session. */
export type EventLayout = "Standing" | "Seated";

/** Who pays a vendor or venue: the WO (out of the contract) or the client directly. */
export type SupplierPayer = "WO" | "Client";

export type SupplierKind = "vendor" | "venue";

export interface EventSession {
  id: string;
  label: string;
  /** "HH:MM" or null. */
  startTime: string | null;
  endTime: string | null;
  guests: number;
}

/**
 * The venue an event takes place at. Staff get the venue row (supplierId) with the capacities copied when it was
 * chosen; the Client Portal gets only the name, address and city.
 */
export interface EventVenue {
  supplierId: string;
  venueId: string;
  name: string;
  capacityStanding: number | null;
  capacitySeated: number | null;
  address: string | null;
  city: string | null;
}

/** One acara (Akad, Resepsi, …) with its sessions. */
export interface RincianEvent {
  /** Version of this item as the API gave it; sent back as `If-Match` when saving (ADR-0036). */
  updatedAt: string;
  id: string;
  eventKindId: string;
  kindName: string;
  eventDate: string;
  startTime: string | null;
  endTime: string | null;
  venue: EventVenue | null;
  layout: EventLayout;
  /** The main event: its date is the Hari H. */
  isMain: boolean;
  order: number;
  sessions: EventSession[];
  totalGuests: number;
}

/** What an event write sends: the whole event and its sessions. */
export interface EventInput {
  eventKindId: string;
  eventDate: string;
  startTime: string | null;
  endTime: string | null;
  /** "" = no venue yet. */
  venueId: string;
  layout: EventLayout;
  sessions: Array<{ label: string; startTime: string | null; endTime: string | null; guests: number }>;
}

/* ------------------------------------------------------------------ rincian lines ------------------------------- */

/** What a line of the rincian is (PLAN §2.2). ExtraGuests is computed by the server, never typed. */
export type LineKind = "Package" | "Component" | "ExtraGuests" | "Item" | "Discount";

/** The catalog rule a package line was copied with (prices only for a caller who sees amounts). */
export interface PackageRule {
  packageId: string;
  name: string;
  price?: number;
  eventKindIds: string[];
  includedGuests: number;
  extraGuestStep: number;
  extraGuestStepPrice?: number;
}

/** How an "Tambahan tamu" line was worked out, so the screen can explain it ("150 tamu di atas 500 …"). */
export interface ExtraGuestsInfo {
  eventId: string;
  eventKindName: string;
  guests: number;
  includedGuests: number;
  over: number;
  step: number;
  stepPrice?: number;
  steps: number;
  /** The event has no guests yet: the line is 0 until it does. */
  noGuests: boolean;
}

/** One line of a rincian as the server gives it. Amounts are absent for a caller who may not see them. */
export interface RincianLine {
  /** "" for a line of a preview that is not saved yet. */
  id: string;
  /** Echoed from the draft that asked for it (preview only). */
  key?: string;
  kind: LineKind;
  /** The package a component belongs to ("" otherwise). */
  parentId: string;
  parentKey?: string;
  packageRule?: PackageRule;
  description: string;
  categoryId: string;
  eventId: string;
  quantity: number;
  unit: string;
  unitPrice?: number;
  amount?: number;
  isBonus: boolean;
  /** A vendor or venue the WO pays for this line ("" = none; staff only). */
  supplierRef?: string;
  sortOrder: number;
  extraGuests?: ExtraGuestsInfo;
  updatedAt?: string;
}

export interface RincianTotals {
  gross: number;
  discount: number;
  subtotal: number;
  tax: number;
  total: number;
}

export interface RincianMargin {
  subtotal: number;
  supplierCost: number;
  /** How many vendors the WO pays have no "Biaya rekanan" yet. */
  missingCost: number;
  incomplete: boolean;
  margin: number;
}

/** A whole rincian: GET …/lines, and what every write and preview gives back. */
export interface Rincian {
  projectId: string;
  lines: RincianLine[];
  /** Absent for a client without portal.finance.view. */
  totals?: RincianTotals;
  isPkp: boolean;
  /** Version of the whole rincian, sent as `If-Match` when saving (staff only). */
  linesVersion?: string;
  termsTotal?: number;
  hasTerms?: boolean;
  /** The payment schedule no longer adds up to the contract value. */
  termsMismatch?: boolean;
  /** Only with finance.margin.view. */
  margin?: RincianMargin;
}

/** One line as a save or a preview sends it; ids are strings here and numbers on the wire. */
export interface LineDraft {
  id?: string;
  key?: string;
  parentId?: string;
  parentKey?: string;
  kind: Exclude<LineKind, "ExtraGuests">;
  packageId?: string;
  description: string;
  categoryId?: string;
  eventId?: string;
  quantity?: number;
  unit?: string;
  unitPrice?: number;
  amount?: number;
  isBonus?: boolean;
  supplierRef?: string;
}

/** A package of the catalog as the picker lists it. */
export interface PackageChoice {
  id: string;
  name: string;
  price: number;
  includedGuests: number;
  componentCount: number;
  eventKindNames: string[];
}

/** A vendor or venue a line can point at (only those the WO pays). */
export interface SupplierChoice {
  id: string;
  name: string;
  kindLabel: string;
}

export interface NamedChoice {
  id: string;
  name: string;
}
