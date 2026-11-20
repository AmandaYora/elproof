import type { Locator } from "@playwright/test";
import { apiURL } from "./config";
import { call, must } from "./api";

/**
 * The `events` a spec sends with POST /projects: one Resepsi on `date`, no venue, one session.
 * Event kind ids differ per workspace, so the kind is looked up by name with the caller's token.
 */
export async function resepsiOn(token: string, date: string) {
  const kinds = must("event kinds", await call(apiURL(), token, "GET", "/api/v1/event-kinds")).json.data as { id: number; name: string }[];
  const kind = kinds.find((k) => k.name === "Resepsi") ?? kinds[0];
  return [
    {
      eventKindId: kind.id, eventDate: date, startTime: "11:00", endTime: "14:00", venueId: null, layout: "Standing",
      sessions: [{ label: "Sesi 1", startTime: "11:00", endTime: "14:00", guests: 100 }],
    },
  ];
}

/**
 * Gives a project its contract value: projects carry no package or contract value of their own any more, the
 * contract is the total of the project's Rincian. This saves a Rincian of one Item at `amount` (a new workspace is
 * not PKP, so the contract total is exactly `amount`), with the Rincian's current `linesVersion` as If-Match.
 */
export async function withContract(token: string, projectId: number, amount: number, description = "Paket Uji") {
  const current = must("lines", await call(apiURL(), token, "GET", `/api/v1/projects/${projectId}/lines`)).json.data as { linesVersion: string };
  must("rincian", await call(apiURL(), token, "PUT", `/api/v1/projects/${projectId}/lines`, {
    lines: [{ kind: "Item", description, quantity: 1, unit: "paket", unitPrice: amount }],
  }, { "If-Match": `"${current.linesVersion}"` }));
}

/**
 * Picks an option of the app's select (shared/components/ui/Combobox): a button that opens a listbox in a portal,
 * so Playwright's selectOption does not apply. `trigger` is the button (its aria-label starts with the field name).
 */
export async function choose(trigger: Locator, option: string | RegExp) {
  await trigger.click();
  const name = typeof option === "string" ? new RegExp(`^${option.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")}$`) : option;
  await trigger.page().getByRole("listbox").getByRole("option", { name }).click();
}
