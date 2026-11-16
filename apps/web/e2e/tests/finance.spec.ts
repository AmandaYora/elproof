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
});
