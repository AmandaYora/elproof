import { expect, visit, test } from "../lib/fixtures";
import { open } from "../lib/suite";
import { apiURL } from "../lib/config";
import { browserSession } from "../lib/session";
import { resepsiOn, withContract } from "../lib/project";

// PLAN Tahap 5: the Ringkasan progress and the ledger totals never go stale after a change, and a routine delete can be undone.
// These tests WRITE data, so they run in the "mutating" project (after every screenshot project).

const API = (path: string) => `${apiURL()}/api/v1${path}`;

async function newProject(page: import("@playwright/test").Page, name: string) {
  const session = await browserSession(page);
  const auth = { Authorization: `Bearer ${session.accessToken}` };
  const made = await page.request.post(API("/projects"), {
    headers: auth,
    data: {
      name, brideName: "Lina Kusuma", groomName: "Reza Mahendra", events: await resepsiOn(session.accessToken, "2027-07-01"), prepStartDate: "2026-11-09",
      status: "Draft", picStaffId: Number(session.principalId), picSalesStaffId: 0, description: "",
    },
  });
  expect(made.status()).toBe(201);
  const id = ((await made.json()).data as { id: number }).id;
  await withContract(session.accessToken, id, 10_000_000);
  return { id, auth };
}

test("finishing a step updates the Ringkasan progress without a reload", async ({ browser }, info) => {
  const { page, close } = await open(browser, info, "paying");
  await visit(page, "/beranda");
  const { id } = await newProject(page, "Uji Header");
  await visit(page, `/proyek/${id}/ringkasan`);
  const tabs = page.getByRole("navigation", { name: "Bagian proyek" });
  const percent = page.getByRole("region", { name: "Angka utama" }).getByText(/^\d+%$/).first();
  await expect(percent).toHaveText("0%");

  await tabs.getByRole("link", { name: "Jadwal" }).click();
  await expect(page).toHaveURL(new RegExp(`/proyek/${id}/jadwal$`));
  const firstRow = page.getByRole("row").filter({ hasText: /^1\./ }).first();
  const status = firstRow.getByRole("button", { name: /Belum mulai/ });
  // The list closes on any page scroll, so bring the row into view first and let the page settle.
  await status.scrollIntoViewIfNeeded();
  await page.waitForTimeout(200);
  await status.click();
  await page.getByRole("option", { name: "Selesai" }).click();
  await expect(firstRow.getByRole("button", { name: /Selesai/ })).toBeVisible();

  // Back on Ringkasan (same page load), the progress already counts the finished step.
  await tabs.getByRole("link", { name: "Ringkasan" }).click();
  await expect(percent).not.toHaveText("0%", { timeout: 10_000 });
  await close();
});

test("deleting a payment is instant, the totals follow, and Urungkan brings it back", async ({ browser }, info) => {
  const { page, close } = await open(browser, info, "paying");
  await visit(page, "/beranda");
  const { id, auth } = await newProject(page, "Uji Hapus");
  const paid = await page.request.post(API(`/projects/${id}/client-payments`), {
    headers: auth,
    data: { type: "DP", amount: 2_000_000, paymentDate: "2026-11-09", method: "Transfer Bank", referenceNumber: "UJI-1", notes: "" },
  });
  expect(paid.status()).toBe(201);

  await visit(page, `/proyek/${id}/keuangan/klien`);
  const remaining = page.getByRole("region", { name: "Uang masuk dari klien" }).locator("dl > div").filter({ hasText: "Sisa tagihan" });
  await expect(remaining).toContainText("Rp8.000.000");
  const rows = page.getByRole("row").filter({ hasText: "Rp2.000.000" });
  await expect(rows).toHaveCount(1);

  // Delete sits behind the row's "⋯" (RISET WO Console §3 #7).
  await rows.first().getByRole("button", { name: /^Aksi lain untuk DP/ }).click();
  await page.getByRole("menuitem", { name: "Hapus pembayaran" }).click();
  await expect(rows).toHaveCount(0); // gone at once, no "Yakin?" step
  const toast = page.getByText("Pembayaran dihapus");
  await expect(toast).toBeVisible();
  await expect(remaining).toContainText("Rp10.000.000", { timeout: 10_000 });

  await page.getByRole("button", { name: "Urungkan" }).click();
  await expect(rows).toHaveCount(1);
  await expect(remaining).toContainText("Rp8.000.000", { timeout: 10_000 });
  await close();
});
