import { expect, visit, test } from "../lib/fixtures";
import { open } from "../lib/suite";

// PLAN Tahap 5: the project page loads from one overview call (plus the Riwayat panel's five entries) and the Vendor & venue tab from three. The shell's own calls
// (permissions, branding, subscription, announcements) are not page data and are left out of the count.

const SHELL = [/\/access\/me$/, /\/tenants\/me\/branding$/, /\/tenants\/me\/logo$/, /\/subscription/, /\/announcements\/active$/, /\/auth\/me$/, /\/onboarding/, /\/tenants\/me\//];

function dataCalls(calls: string[]): string[] {
  return calls.filter((c) => !SHELL.some((re) => re.test(c.replace(/^GET /, "").split("?")[0])));
}

test.describe("project page", () => {
  test("Ringkasan loads from one overview call plus the five latest history entries, and makes no other data call", async ({ browser }, info) => {
    const { page, close } = await open(browser, info, "owner");
    const calls: string[] = [];
    page.on("request", (r) => {
      const u = new URL(r.url());
      if (u.pathname.startsWith("/api/v1/") && r.method() === "GET") calls.push(`GET ${u.pathname}${u.search}`);
    });
    await visit(page, "/proyek/1/ringkasan");
    await expect(page.getByRole("heading", { name: "Riwayat" })).toBeVisible();
    await page.waitForLoadState("networkidle");
    const data = dataCalls(calls);
    expect(data.filter((c) => c.endsWith("/projects/1/overview")), data.join("\n")).toHaveLength(1);
    // The Riwayat panel (RISET WO Console §3 #4): one small page of the history, not the whole list.
    const history = data.filter((c) => c.includes("/projects/1/activity"));
    expect(history, data.join("\n")).toHaveLength(1);
    expect(history[0]).toMatch(/[?&]limit=5(&|$)/);
    for (const old of ["/milestones", "/vendors", "/client-payments", "/payment-terms", "/issues", "/staff", "/evidence", "/clients"]) {
      expect(data.some((c) => c.includes(old)), `unexpected call ${old}\n${data.join("\n")}`).toBe(false);
    }
    expect(data.length, data.join("\n")).toBeLessThanOrEqual(4);
    await close();
  });

  test("opening the Vendor & venue tab costs at most three data calls", async ({ browser }, info) => {
    const { page, close } = await open(browser, info, "owner");
    await visit(page, "/proyek/1/ringkasan");
    const calls: string[] = [];
    page.on("request", (r) => {
      const u = new URL(r.url());
      if (u.pathname.startsWith("/api/v1/") && r.method() === "GET") calls.push(`GET ${u.pathname}${u.search}`);
    });
    await page.getByRole("navigation", { name: "Bagian proyek" }).getByRole("link", { name: "Vendor & venue" }).click();
    await expect(page).toHaveURL(/\/proyek\/1\/vendor/);
    await expect(page.getByRole("heading", { name: /Vendor/ }).first()).toBeVisible();
    await page.waitForLoadState("networkidle");
    const data = dataCalls(calls);
    // The vendors and their steps (two), and the attached venue's own record (one).
    expect(data.length, data.join("\n")).toBeLessThanOrEqual(3);
    // The pickers (vendor list, categories, staff, venue directory) belong to the forms and load only when a form opens.
    for (const lazy of ["/vendor-categories", "/staff/summary", "/vendors?"]) expect(data.some((c) => c.includes(lazy)), `${lazy} loaded too early\n${data.join("\n")}`).toBe(false);
    expect(data.some((c) => /\/venues(\?|$)/.test(c.replace(/^GET /, ""))), `the venue directory loaded too early\n${data.join("\n")}`).toBe(false);
    await close();
  });

  test("old tab addresses land on the tab that holds them now", async ({ browser }, info) => {
    const { page, close } = await open(browser, info, "owner");
    for (const [from, to] of [
      ["milestone", /\/proyek\/1\/jadwal$/],
      ["venue", /\/proyek\/1\/vendor$/],
      ["client", /\/proyek\/1\/ringkasan$/],
      ["aktivitas", /\/proyek\/1\/ringkasan$/],
      ["pembayaran/client", /\/proyek\/1\/keuangan\/klien$/],
      ["pembayaran/vendor", /\/proyek\/1\/keuangan\/vendor$/],
      ["pembayaran/venue", /\/proyek\/1\/keuangan\/venue$/],
    ] as const) {
      await visit(page, `/proyek/1/${from}`);
      await expect(page, from).toHaveURL(to);
    }
    await close();
  });

  test("Jadwal berikutnya counts H-n to the event day, like the Client Portal, with how soon as a second part (QA RT-02)", async ({ browser }, info) => {
    const { page, close } = await open(browser, info, "owner");
    // A wedding on 12 Jul 2027 with its last two steps; the browser's today is fixed at 9 Nov 2026.
    await page.route(/\/api\/v1\/projects\/1\/overview$/, async (route) => {
      const res = await route.fetch();
      const body = await res.json();
      body.data.project.eventDate = "2027-07-12";
      body.data.nextMilestones = [
        { id: 9201, name: "Gladi resik", status: "Not Started", targetDate: "2027-07-11" },
        { id: 9202, name: "Hari-H pernikahan", status: "Not Started", targetDate: "2027-07-12" },
      ];
      await route.fulfill({ response: res, json: body });
    });
    await visit(page, "/proyek/1/ringkasan");
    const step = (name: string) => page.getByRole("listitem").filter({ hasText: name });
    await expect(step("Gladi resik")).toContainText("11 Jul 2027 · H-1");
    await expect(step("Gladi resik")).toContainText("dalam 244 hari");
    await expect(step("Hari-H pernikahan")).toContainText("12 Jul 2027 · Hari H");
    await expect(step("Hari-H pernikahan")).toContainText("dalam 245 hari");
    await expect(step("Gladi resik")).not.toContainText("H-244");
    // The header still counts down to the event itself, from today.
    await expect(page.getByRole("list", { name: "Data utama proyek" })).toContainText("H-245");
    await close();
  });
});
