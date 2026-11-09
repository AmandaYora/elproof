import { expect, settle, state, test, visit } from "../lib/fixtures";
import { open } from "../lib/suite";
import { apiURL } from "../lib/config";

// PLAN Tahap 3: the Client Portal loads with few calls, shows proof where the WO has it, never shows a stamp without
// proof, and no longer reads the tenant's vendor lists.

const API = (path: string) => `${apiURL()}/api/v1${path}`;

test.describe("portal Beranda", () => {
  test("loads from one data call and never touches the tenant's vendor lists", async ({ browser }, info) => {
    const { page, close } = await open(browser, info, "client");
    const calls: string[] = [];
    page.on("request", (r) => {
      const u = new URL(r.url());
      if (u.pathname.startsWith("/api/v1/")) calls.push(`${r.method()} ${u.pathname}`);
    });
    await visit(page, "/portal/beranda");
    expect(calls.filter((c) => c === "GET /api/v1/projects/me/home"), calls.join("\n")).toHaveLength(1);
    for (const old of ["/vendors/summary", "/vendor-categories", "/milestones", "/client-payments", "/payment-terms", "/issues", "/evidence", "/tenants/me/contact"]) {
      expect(calls.some((c) => c.includes(old)), `unexpected call ${old}\n${calls.join("\n")}`).toBe(false);
    }
    expect(calls.includes("GET /api/v1/projects/me"), "the old project call is gone").toBe(false);
    // Budget: the landing data, plus the shell's permissions and branding. (PLAN target: <= 2 for the data itself.)
    expect(calls.length, calls.join("\n")).toBeLessThanOrEqual(3);
    await close();
  });

  test("a verified payment shows a stamp that opens its proof, and focus comes back", async ({ browser }, info) => {
    test.skip(!state().storage, "needs object storage (MinIO) so the seed can attach proof");
    const { page, close } = await open(browser, info, "client");
    await visit(page, "/portal/beranda");
    const stamp = page.getByRole("button", { name: /Terverifikasi .* oleh/ });
    await expect(stamp).toHaveCount(1);
    await stamp.focus();
    await page.keyboard.press("Enter");
    const dialog = page.getByRole("dialog");
    await expect(dialog).toBeVisible();
    await expect(dialog.getByRole("heading").first()).toContainText(/DP|Bukti/i);
    await page.keyboard.press("Escape");
    await expect(dialog).toBeHidden();
    await expect(stamp).toBeFocused();
    await close();
  });
});

test.describe("portal Pembayaran", () => {
  test("only a verified payment that has a proof file gets a stamp", async ({ browser }, info) => {
    const { page, close } = await open(browser, info, "client");
    // A payment the server says is verified but that carries no file must not be stamped.
    await page.route(/\/api\/v1\/projects\/\d+\/client-payments$/, async (route) => {
      const res = await route.fetch();
      const body = await res.json();
      body.data = [
        ...body.data,
        { id: 9001, type: "Termin", amount: 1_000_000, paymentDate: "2026-11-01", method: "Transfer", referenceNumber: "", notes: "", evidenceComplete: false, verifiedAt: "2026-11-02T03:00:00Z", verifiedBy: { name: "Rina" }, evidence: [] },
      ];
      await route.fulfill({ response: res, json: body });
    });
    await visit(page, "/portal/pembayaran");
    await expect(page.getByText("Bukti transfer belum diunggah")).toBeVisible();
    // The seeded DP is the only stamped one (when storage let the seed attach its proof).
    await expect(page.getByRole("button", { name: /Terverifikasi/ })).toHaveCount(state().storage ? 1 : 0);
    await close();
  });
});

test.describe("client boundary", () => {
  test("a client cannot read the tenant's vendor names or categories", async ({ browser }, info) => {
    const { page, close } = await open(browser, info, "client");
    await visit(page, "/portal/beranda");
    const token = await page.evaluate(() => (JSON.parse(localStorage.getItem("elproof.session") ?? "{}") as { accessToken?: string }).accessToken ?? "");
    expect(token).not.toBe("");
    for (const path of ["/vendors/summary", "/vendor-categories?all=true", "/vendors"]) {
      const res = await page.request.get(API(path), { headers: { Authorization: `Bearer ${token}` } });
      expect(res.status(), path).toBe(403);
    }
    await settle(page);
    await close();
  });
});
