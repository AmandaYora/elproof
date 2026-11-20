/**
 * The rincian module's only public entry (PLAN §2.2). It holds the building blocks of "Acara & rincian" that do not
 * care where the data comes from: the event and session editor, the vendor/venue settings fields, and the rincian
 * editor with its "Ringkasan biaya". A screen feeds them through props and callbacks; the project wiring (store
 * calls, permissions, routes) stays in modules/projects, the portal's in modules/client-portal.
 */

// Types
export type {
  EngagementStatus,
  EventInput,
  EventLayout,
  EventSession,
  EventVenue,
  ExtraGuestsInfo,
  LineDraft,
  LineKind,
  NamedChoice,
  PackageChoice,
  PackageRule,
  Rincian,
  RincianEvent,
  RincianLine,
  RincianMargin,
  RincianTotals,
  SupplierChoice,
  SupplierKind,
  SupplierPayer,
} from "@/modules/rincian/types";

// Acara (events and sessions)
export { EVENT_LAYOUT_OPTIONS, eventSchema, eventSessionSchema, type EventFormValues, type EventSessionFormValues } from "@/modules/rincian/schemas/event.schema";
export { EventFields, type EventFieldsProps } from "@/modules/rincian/components/events/EventFields";
export { EventList, type EventRowAction } from "@/modules/rincian/components/events/EventList";
export { VenuePicker, capacityLine } from "@/modules/rincian/components/events/VenuePicker";
export { useEventPickers } from "@/modules/rincian/components/events/useEventPickers";
export { useServerRefusal } from "@/modules/rincian/components/events/useServerRefusal";
export {
  bracketKey,
  capacityFor,
  dayAndMonth,
  draftFromEvent,
  emptyEventDraft,
  emptySession,
  eventsLine,
  formatClock,
  formatCount,
  formatEventDay,
  formatTimeRange,
  locationLine,
  maxSessionGuests,
  otherLayout,
  refusedSessions,
  splitSession,
  toDraft,
  toEventInput,
  totalGuests,
  venueLookup,
  type EventDraftInput,
  type VenueCaps,
} from "@/modules/rincian/components/events/event-display";

// Vendor & venue settings
export {
  ENGAGEMENT_STATUS_OPTIONS,
  SUPPLIER_PAYER_OPTIONS,
  supplierSchema,
  type SupplierFormValues,
} from "@/modules/rincian/schemas/supplier.schema";
export { SupplierFields, SUPPLIER_FIELD_LABELS, type SupplierFieldValues } from "@/modules/rincian/components/suppliers/SupplierFields";
export { isMissingCost, supplierKindLabel } from "@/modules/rincian/lib/suppliers";

// Rincian (price lines)
export { toRincian, lineDraftBody, type RawRincian } from "@/modules/rincian/lib/wire";
export { extraGuestsTotal } from "@/modules/rincian/lib/editor";
export { extraGuestsFormula, extraGuestsPlain, itemFormula } from "@/modules/rincian/lib/describe";
export { RincianEditor, type RincianEditorProps } from "@/modules/rincian/components/lines/RincianEditor";
export { CostSummaryPanel, type CostSummaryPanelProps } from "@/modules/rincian/components/lines/CostSummaryPanel";
export { PortalRincian } from "@/modules/rincian/components/lines/PortalRincian";
