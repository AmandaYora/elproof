import { ROUTE_PATHS } from "@/app/routes/route-paths";
import type { PendingVerification } from "@/modules/finance/types";

/** The three books of a project's Keuangan tab: the client's payments, the vendors and venues the WO pays, and what the client paid them directly. */
export type ProjectFinanceTab = "klien" | "vendor-venue" | "dibayar-klien";

/** Where a row of the Keuangan page opens: the project's own Keuangan tab, where money is recorded. */
export function projectFinanceHref(projectId: string, tab: ProjectFinanceTab = "klien"): string {
  return ROUTE_PATHS.projectDetail(projectId, `keuangan/${tab}`);
}

/** "Citra & Bima" from the two names, whichever are filled in. */
export function coupleOf(row: { brideName: string; groomName: string }): string {
  return [row.brideName, row.groomName].filter(Boolean).join(" & ");
}

/** The key of a verification row: ids of a client payment and a supplier payment can be equal. */
export function verificationRowId(v: Pick<PendingVerification, "kind" | "id">): string {
  return `${v.kind}:${v.id}`;
}

/**
 * Where "Periksa bukti" of a verification row opens: a client payment in the client book; the couple's proof of paying
 * a vendor straight in its review on Keuangan › Dibayar klien langsung (`?periksa=<id>`).
 */
export function verificationHref(v: Pick<PendingVerification, "kind" | "id" | "projectId">): string {
  return v.kind === "supplierPayment"
    ? `${projectFinanceHref(v.projectId, "dibayar-klien")}?periksa=${encodeURIComponent(v.id)}`
    : projectFinanceHref(v.projectId, "klien");
}
