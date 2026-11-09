import { expect, visit, test } from "../lib/fixtures";
import { open } from "../lib/suite";
import { apiURL } from "../lib/config";

// PLAN Tahap 5: the header and side panel never go stale after a change, and a routine delete can be undone.
// These tests WRITE data, so they run in the "mutating" project (after every screenshot project).

const API = (path: string) => `${apiURL()}/api/v1${path}`;

async function newProject(page: import("@playwright/test").Page, name: string) {
  const session = await page.evaluate(() => JSON.parse(localStorage.getItem("elproof.session") ?? "{}") as { accessToken: string; principalId: string });
  const auth = { Authorization: `Bearer ${session.accessToken}` };
  const made = await page.request.post(API("/projects"), {
    headers: auth,
    data: {
      name, brideName: "Lina Kusuma", groomName: "Reza Mahendra", eventDate: "2027-07-01", venue: "Gedung Uji", prepStartDate: "2026-11-09",
      packageName: "Paket Uji", contractValue: 10_000_000, status: "Draft", picStaffId: Number(session.principalId), picSalesStaffId: 0, description: "",
    },
  });
  expect(made.status()).toBe(201);
  return { id: ((await made.json()).data as { id: number }).id, auth };
}

test("finishing a step updates the header percentage without a reload", async ({ browser }, info) => {
  const { page, close } = await open(browser, info, "paying");
  await visit(page, "/dashboard");
  const { id } = await newProject(page, "Uji Header");
  await visit(page, `/projects/${id}/milestone`);
  const percent = page.locator("h1").locator("xpath=ancestor::*[contains(@class,'flex-col')][1]").getByText(/^\d+%$/).first();
  await expect(percent).toHaveText("0%");
  const firstRow = page.getByRole("row").filter({ hasText: /^1\./ }).first();
  const status = firstRow.getByRole("button", { name: /Belum mulai/ });
  // The list closes on any page scroll, so bring the row into view first and let the page settle.
  await status.scrollIntoViewIfNeeded();
  await page.waitForTimeout(200);
  await status.click();
  await page.getByRole("option", { name: "Selesai" }).click();
  await expect(percent).not.toHaveText("0%", { timeout: 10_000 });
  await close();
});

test("deleting a payment is instant, the side panel follows, and Urungkan brings it back", async ({ browser }, info) => {
  const { page, close } = await open(browser, info, "paying");
  await visit(page, "/dashboard");
  const { id, auth } = await newProject(page, "Uji Hapus");
  const paid = await page.request.post(API(`/projects/${id}/client-payments`), {
    headers: auth,
    data: { type: "DP", amount: 2_000_000, paymentDate: "2026-11-09", method: "Transfer Bank", referenceNumber: "UJI-1", notes: "" },
  });
  expect(paid.status()).toBe(201);

  await visit(page, `/projects/${id}/pembayaran/client`);
  const panel = page.getByRole("complementary", { name: "Ringkasan proyek" });
  await expect(panel).toContainText("Rp8.000.000");
  const rows = page.getByRole("row").filter({ hasText: "Rp2.000.000" });
  await expect(rows).toHaveCount(1);

  await page.getByRole("button", { name: "Hapus Pembayaran" }).first().click();
  await expect(rows).toHaveCount(0); // gone at once, no "Yakin?" step
  const toast = page.getByText("Pembayaran dihapus");
  await expect(toast).toBeVisible();
  await expect(panel).toContainText("Rp10.000.000", { timeout: 10_000 }); // the panel follows the server's numbers

  await page.getByRole("button", { name: "Urungkan" }).click();
  await expect(rows).toHaveCount(1);
  await expect(panel).toContainText("Rp8.000.000", { timeout: 10_000 });
  await close();
});
