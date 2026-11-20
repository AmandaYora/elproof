import { isMissingCost } from "@/modules/rincian";
import type { ProjectSupplier, SupplierPayer, VendorMilestone } from "@/modules/projects/types";

/**
 * The filters of the Vendor & venue list ("Saring"), kept in the address as `?saring=` so a link can open the list
 * already filtered ("Margin belum lengkap (2 vendor)" on Ringkasan lands on `tanpa-biaya`).
 */
export type SupplierFilter = "semua" | "vendor" | "venue" | "dibayar-wo" | "dibayar-klien" | "tanpa-biaya";

/** The address parameters of the Vendor & venue tab: the filter and the supplier whose drawer is open. */
export const SUPPLIER_FILTER_PARAM = "saring";
export const SUPPLIER_OPEN_PARAM = "buka";

export const SUPPLIER_FILTER_LABELS: Record<SupplierFilter, string> = {
  semua: "Semua",
  vendor: "Vendor saja",
  venue: "Venue saja",
  "dibayar-wo": "Dibayar WO",
  "dibayar-klien": "Dibayar klien langsung",
  "tanpa-biaya": "Biaya rekanan belum diisi",
};

export function isSupplierFilter(value: string | null): value is SupplierFilter {
  return value !== null && value in SUPPLIER_FILTER_LABELS;
}

export function matchesFilter(s: ProjectSupplier, filter: SupplierFilter): boolean {
  switch (filter) {
    case "vendor":
    case "venue":
      return s.kind === filter;
    case "dibayar-wo":
      return s.paidBy === "WO";
    case "dibayar-klien":
      return s.paidBy === "Client";
    case "tanpa-biaya":
      return isMissingCost(s);
    default:
      return true;
  }
}

/** Done steps out of the steps that still count (a cancelled step is not work left). */
export function stepProgress(steps: VendorMilestone[]): { done: number; total: number } {
  const live = steps.filter((m) => m.status !== "Cancelled");
  return { done: live.filter((m) => m.status === "Completed").length, total: live.length };
}

/** Steps of one supplier in their order, cancelled ones last. */
export function stepsOf(all: VendorMilestone[], supplierId: string): VendorMilestone[] {
  return all
    .filter((m) => m.projectSupplierId === supplierId)
    .sort((a, b) => Number(a.status === "Cancelled") - Number(b.status === "Cancelled") || a.order - b.order);
}

/** Active ones first, cancelled last; venues before vendors, then by name. */
export function sortSuppliers(list: ProjectSupplier[]): ProjectSupplier[] {
  return [...list].sort(
    (a, b) =>
      Number(a.engagementStatus === "Cancelled") - Number(b.engagementStatus === "Cancelled") ||
      Number(a.kind === "vendor") - Number(b.kind === "vendor") ||
      a.name.localeCompare(b.name, "id"),
  );
}

/**
 * Whether the caller may record (and change) a payment to a supplier paid by `payer`: the WO's payments need the margin
 * permission, the client's need finance.record. Mirrors the API (canRecordSupplierPayment).
 */
export function canRecordFor(payer: SupplierPayer, perms: { margin: boolean; record: boolean }): boolean {
  return payer === "WO" ? perms.margin : perms.record;
}
