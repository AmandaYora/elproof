import { expect, visit, test } from "../lib/fixtures";
import { open } from "../lib/suite";

// Paket WO, Tahap 1 (docs/plan/paket-wo/PLAN.md §8, DESAIN.md §6.6–6.8): the catalog a WO prepares before it sends quotations.
// These tests WRITE data, so they run in the "mutating" project, on WO Berbayar (WO Contoh and WO Kosong feed the screenshots).

test.describe("catalog", () => {
  test("a package filled from the example is priced by the server, saved, and counted on its event kinds", async ({ browser }, info) => {
    const { page, close } = await open(browser, info, "paying");
    await visit(page, "/paket");
    await page.getByRole("button", { name: "Isi dengan contoh" }).click();
    await expect(page).toHaveURL(/\/paket\/baru\?contoh=1$/);
    // Wait for the form itself: right after the click the list (and its "Cari nama paket..." box) can still be on screen.
    await expect(page.getByText("Terisi dengan contoh.")).toBeVisible();

    // The example covers Akad + Resepsi, 500 guests included, +Rp4 jt per 50: the preview for 620 guests comes from the server.
    await page.getByRole("textbox", { name: /^Nama paket/ }).fill("Paket Resepsi Uji");
    await expect(page.getByText("Rp132.000.000")).toBeVisible();
    await page.getByRole("textbox", { name: "Untuk tamu" }).fill("450");
    await expect(page.getByText("Rp132.000.000")).toHaveCount(0); // at or below the included guests the price stays fixed

    await page.getByRole("button", { name: "Simpan paket" }).first().click();
    await expect(page.getByText("Paket disimpan")).toBeVisible();
    await expect(page).toHaveURL(/\/paket\/\d+$/);
    await expect(page.getByRole("heading", { level: 1, name: "Paket Resepsi Uji" })).toBeVisible();

    await visit(page, "/paket");
    const row = page.getByRole("row", { name: /Paket Resepsi Uji/ });
    await expect(row).toContainText("Akad, Resepsi");
    await expect(row).toContainText("+Rp4 jt per 50 tamu");

    await visit(page, "/jenis-acara");
    await expect(page.getByRole("row", { name: /Akad/ }).first()).toContainText("1 paket");
    await expect(page.getByRole("row", { name: /Lamaran/ })).toContainText("Belum dipakai");
    await close();
  });

  test("an empty package form names every missing field, and nothing is saved", async ({ browser }, info) => {
    const { page, close } = await open(browser, info, "paying");
    await visit(page, "/paket/baru");
    await page.getByRole("button", { name: "Simpan paket" }).first().click();
    await expect(page.getByText("Isi nama paket").first()).toBeVisible();
    await expect(page.getByText("Pilih minimal satu acara yang dicakup").first()).toBeVisible();
    await expect(page).toHaveURL(/\/paket\/baru$/);
    await close();
  });

  test("the business profile shows the quotation settings a new WO starts with", async ({ browser }, info) => {
    const { page, close } = await open(browser, info, "paying");
    await visit(page, "/profil-usaha");
    const card = page.locator("#penawaran");
    await expect(card).toContainText("Bukan PKP");
    await expect(card).toContainText("14 hari");
    await expect(card).toContainText(/Contoh nomor: [A-Z0-9]{2,5}-PNW-\d{6}-001/);
    await close();
  });
});
