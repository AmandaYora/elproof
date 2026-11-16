import { expect, visit, test } from "../lib/fixtures";
import { open } from "../lib/suite";

// Keuangan (RISET WO Console §3 #1, §7.6): money across projects. The seed gives "Citra & Bima" a paid DP and an
// overdue "Termin 2" (Rp40.000.000); the page opens on the overdue installments.

test.describe("Keuangan", () => {
  test("opens on the overdue installments, filters on the server, and a row opens the project's Keuangan tab", async ({ browser }, info) => {
    const { page, close } = await open(browser, info, "owner");
    const firstList = page.waitForResponse((r) => r.request().method() === "GET" && r.url().includes("/api/v1/finance/receivables"));
    await visit(page, "/keuangan");
    await expect(page.getByRole("heading", { level: 1, name: "Keuangan" })).toBeVisible();

    // The default view asks the server for the overdue installments only.
    expect((await firstList).url()).toMatch(/status=overdue/);
    const kpis = page.getByRole("region", { name: "Ringkasan keuangan" });
    await expect(kpis.getByRole("link", { name: /Tagihan terlambat/ })).toContainText("Rp40.000.000");

    const table = page.getByRole("table", { name: "Daftar tagihan klien" });
    const overdueRow = table.getByRole("row").filter({ hasText: "Termin 2" });
    await expect(overdueRow).toContainText("Citra & Bima");
    await expect(overdueRow).toContainText("Terlambat");
    await expect(overdueRow).toContainText(/terlambat \d+ hari/);
    await expect(overdueRow).toContainText("Rp40.000.000");
    await expect(table.getByRole("row").filter({ hasText: "DP 30%" })).toHaveCount(0);

    // "Lunas": a server-side view, kept in the address.
    const paidList = page.waitForResponse((r) => r.url().includes("/api/v1/finance/receivables") && /status=paid/.test(r.url()));
    await page.getByRole("group", { name: "Saring menurut status termin" }).getByRole("button", { name: /Lunas/ }).click();
    await paidList;
    await expect(page).toHaveURL(/status=paid/);
    await expect(table.getByRole("row").filter({ hasText: "DP 30%" })).toContainText("Lunas");
    await expect(table.getByRole("row").filter({ hasText: "Termin 2" })).toHaveCount(0);

    // A search with no match says so; Reset filter keeps the chosen status view.
    await page.getByPlaceholder("Cari proyek, pasangan, atau termin...").fill("zzzz-tidak-ada");
    await expect(page.getByText("Tidak ada hasil")).toBeVisible();
    await page.getByRole("button", { name: "Reset filter" }).click();
    await expect(page).toHaveURL(/status=paid/);
    await expect(page).not.toHaveURL(/q=/);
    await expect(table.getByRole("row").filter({ hasText: "DP 30%" })).toBeVisible();

    // The row's link leads to the project's Keuangan tab (client payments), where money is recorded.
    await table.getByRole("row").filter({ hasText: "DP 30%" }).getByRole("link", { name: /Citra & Bima/ }).click();
    await expect(page).toHaveURL(/\/proyek\/\d+\/keuangan\/klien$/);
    await close();
  });

  test("a couple's unverified submission is shown apart, not counted as received (QA RT-03)", async ({ browser }, info) => {
    const { page, close } = await open(browser, info, "owner");
    const figure = (label: string) => page.locator("dt", { hasText: label }).locator("xpath=following-sibling::dd[1]");
    await visit(page, "/proyek/1/keuangan/klien");
    const received = (await figure("Total diterima").innerText()).trim();
    const remaining = (await figure("Sisa tagihan").innerText()).trim();
    await expect(figure("Total diterima")).not.toContainText("menunggu verifikasi");

    // The same ledger plus a transfer the couple sent from the portal that the WO has not verified yet (ADR-0047):
    // the server does not count it as paid, so neither may the page.
    await page.route(/\/api\/v1\/projects\/\d+\/client-payments$/, async (route) => {
      if (route.request().method() !== "GET") return route.fallback();
      const res = await route.fetch();
      const body = await res.json();
      body.data = [
        ...body.data,
        { id: 9101, type: "Termin", amount: 2_000_000, paymentDate: "2026-11-08", method: "Transfer", referenceNumber: "", notes: "", evidenceComplete: true, recordedBy: null, verifiedAt: null, verifiedBy: null, evidence: [], source: "client", updatedAt: "2026-11-08T03:00:00.000000Z" },
      ];
      await route.fulfill({ response: res, json: body });
    });
    await visit(page, "/proyek/1/keuangan/klien");
    await expect(page.getByRole("row").filter({ hasText: "Dikirim pasangan" })).toHaveCount(1);
    await expect(figure("Total diterima")).toContainText("Rp2.000.000 menunggu verifikasi");
    expect((await figure("Total diterima").innerText()).trim().split("\n")[0]).toBe(received.split("\n")[0]);
    await expect(figure("Sisa tagihan")).toHaveText(remaining);
    await close();
  });
});
