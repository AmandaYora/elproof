import { PAYABLE_KIND_LABELS } from "@/shared/lib/labels";
import type { EngagementStatus, SupplierKind, SupplierPayer } from "@/modules/rincian/types";

/**
 * A vendor or venue the WO pays that still has no "Biaya rekanan": the margin is incomplete while one exists. Same rule
 * as the server's missingCost (a cancelled one is left out). Only meaningful to a caller who sees the cost.
 */
export function isMissingCost(s: { paidBy: SupplierPayer; agreedCost: number | null; engagementStatus: EngagementStatus }): boolean {
  return s.paidBy === "WO" && s.agreedCost === null && s.engagementStatus !== "Cancelled";
}

/** "Venue" or "Katering" (a vendor is named by its category; one without a category is just "Vendor"). */
export function supplierKindLabel(s: { kind: SupplierKind; categoryName: string }): string {
  if (s.kind === "venue") return PAYABLE_KIND_LABELS.venue;
  return s.categoryName || PAYABLE_KIND_LABELS.vendor;
}
