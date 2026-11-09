import { expect, visit, test } from "../lib/fixtures";
import { open } from "../lib/suite";

// PLAN Tahap 5: the project page loads from one overview call and the Vendor tab from two. The shell's own calls
// (permissions, branding, subscription, announcements) are not page data and are left out of the count.

const SHELL = [/\/access\/me$/, /\/tenants\/me\/branding$/, /\/tenants\/me\/logo$/, /\/subscription/, /\/announcements\/active$/, /\/auth\/me$/, /\/onboarding/, /\/tenants\/me\//];

function dataCalls(calls: string[]): string[] {
  return calls.filter((c) => !SHELL.some((re) => re.test(c.replace(/^GET /, ""))));
}

test.describe("project page", () => {
  test("Ringkasan loads from one overview call and makes no other data call", async ({ browser }, info) => {
    const { page, close } = await open(browser, info, "owner");
    const calls: string[] = [];
    page.on("request", (r) => {
      const u = new URL(r.url());
      if (u.pathname.startsWith("/api/v1/") && r.method() === "GET") calls.push(`GET ${u.pathname}`);
    });
    await visit(page, "/projects/1/ringkasan");
    const data = dataCalls(calls);
    expect(data.filter((c) => c.endsWith("/projects/1/overview")), data.join("\n")).toHaveLength(1);
    for (const old of ["/milestones", "/vendors", "/client-payments", "/payment-terms", "/issues", "/staff", "/evidence", "/activity"]) {
      expect(data.some((c) => c.includes(old)), `unexpected call ${old}\n${data.join("\n")}`).toBe(false);
    }
    expect(data.length, data.join("\n")).toBeLessThanOrEqual(3);
    await close();
  });

  test("opening the Vendor tab costs at most two data calls", async ({ browser }, info) => {
    const { page, close } = await open(browser, info, "owner");
    await visit(page, "/projects/1/ringkasan");
    const calls: string[] = [];
    page.on("request", (r) => {
      const u = new URL(r.url());
      if (u.pathname.startsWith("/api/v1/") && r.method() === "GET") calls.push(`GET ${u.pathname}`);
    });
    // The sidebar also has a "Vendor" link (the directory); the project's tab bar comes after it.
    await page.getByRole("link", { name: "Vendor", exact: true }).last().click();
    await expect(page).toHaveURL(/\/projects\/1\/vendor/);
    await expect(page.getByRole("heading", { name: /Vendor/ }).first()).toBeVisible();
    await page.waitForLoadState("networkidle");
    const data = dataCalls(calls);
    expect(data.length, data.join("\n")).toBeLessThanOrEqual(2);
    // The pickers (vendor list, categories, staff) belong to the forms and load only when a form opens.
    for (const lazy of ["/vendor-categories", "/staff/summary", "/vendors?"]) expect(data.some((c) => c.includes(lazy)), `${lazy} loaded too early\n${data.join("\n")}`).toBe(false);
    await close();
  });
});
