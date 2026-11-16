import { ROUTE_PATHS } from "@/app/routes/route-paths";

/** The three parts of a project's Keuangan tab. */
export type ProjectFinanceTab = "klien" | "vendor" | "venue";

/**
 * Where a row of the Keuangan page opens: the project's own Keuangan tab, where money is recorded. Written as a plain
 * path (the project detail owns its tab names; the old /pembayaran/client addresses redirect there).
 */
export function projectFinanceHref(projectId: string, tab: ProjectFinanceTab = "klien"): string {
  return `${ROUTE_PATHS.projects}/${projectId}/keuangan/${tab}`;
}

/** "Citra & Bima" from the two names, whichever are filled in. */
export function coupleOf(row: { brideName: string; groomName: string }): string {
  return [row.brideName, row.groomName].filter(Boolean).join(" & ");
}
