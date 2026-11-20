import type { RincianEvent } from "@/modules/rincian";
import type { QuotationSupplier } from "@/modules/offers/types";

/** What the quotation page has open over it (each loaded on the click that opens it). */
export type QuotationDialog =
  | { kind: "header" }
  | { kind: "event"; event: RincianEvent | null }
  | { kind: "deleteEvent"; event: RincianEvent }
  | { kind: "supplier"; supplier: QuotationSupplier | null }
  | { kind: "deleteSupplier"; supplier: QuotationSupplier }
  | { kind: "terms" }
  | { kind: "termsText" }
  | { kind: "accept" }
  | { kind: "reject" }
  | { kind: "extend" }
  | { kind: "unaccept" }
  | { kind: "delete" }
  | { kind: "pdf"; title: string; description?: string; url: string; revisionN?: number }
  | { kind: "compare"; from: number; to: number };
