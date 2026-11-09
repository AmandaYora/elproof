import { expect, settle, test, visit } from "../lib/fixtures";
import { open } from "../lib/suite";

// PLAN Tahap 4: hybrid validation, error summary from 3 errors up, server errors on their fields,
// a submit button that is never disabled for invalidity, and a full keyboard path.

test.describe("Tambah vendor", () => {
  test("Enter on an empty form shows every problem at once and focuses the summary", async ({ browser }, info) => {
    const { page, close } = await open(browser, info, "owner");
    await visit(page, "/vendors");
    await page.getByRole("button", { name: "Tambah Vendor" }).click();
    const dialog = page.getByRole("dialog");
    await expect(dialog).toBeVisible();
    const save = dialog.getByRole("button", { name: "Simpan vendor" });
    await expect(save).toBeEnabled(); // never disabled because the form is empty
    await dialog.getByLabel(/Nama Vendor/).press("Enter");
    const summary = dialog.locator("#form-error-summary");
    await expect(summary).toBeVisible();
    await expect(summary).toBeFocused();
    await expect(summary).toContainText(/Ada \d+ isian yang perlu diperbaiki/);
    await expect(dialog.getByLabel(/Nama Vendor/)).toHaveAttribute("aria-invalid", "true");
    // Jumping from the summary lands on the field.
    await summary.getByRole("button").first().click();
    await expect(dialog.getByLabel(/Nama Vendor/)).toBeFocused();
    // Typed input is never cleared, and the message disappears as the field is fixed.
    await dialog.getByLabel(/Nama Vendor/).fill("Katering Baru");
    await expect(dialog.getByLabel(/Nama Vendor/)).not.toHaveAttribute("aria-invalid", "true");
    await expect(dialog.getByLabel(/Nama Vendor/)).toHaveValue("Katering Baru");
    await close();
  });

  test("a 422 from the server lands on its own field, and the modal stays open", async ({ browser }, info) => {
    const { page, close } = await open(browser, info, "owner");
    await page.route(/\/api\/v1\/vendors$/, async (route) => {
      if (route.request().method() !== "POST") return route.fallback();
      return route.fulfill({
        status: 422,
        contentType: "application/json",
        body: JSON.stringify({ success: false, message: "Data vendor belum lengkap", errors: { name: ["Nama vendor sudah dipakai. Gunakan nama lain."] } }),
      });
    });
    await visit(page, "/vendors");
    await page.getByRole("button", { name: "Tambah Vendor" }).click();
    const dialog = page.getByRole("dialog");
    await dialog.getByLabel(/Nama Vendor/).fill("Katering Rasa Nusantara");
    await dialog.getByLabel(/Nama PIC/).fill("Bu Wati");
    await dialog.getByLabel(/No Tlp Vendor/).fill("081211110000");
    await dialog.getByLabel(/Kota/).click();
    await page.getByRole("option", { name: "Kota Bandung" }).first().click();
    const money = dialog.getByLabel(/Harga Akad \(Rp\)/);
    await money.fill("5000000");
    await dialog.getByLabel(/Harga Akad\+Resepsi/).fill("12000000");
    await dialog.getByRole("button", { name: "Simpan vendor" }).click();
    await expect(dialog.getByText("Nama vendor sudah dipakai. Gunakan nama lain.")).toBeVisible();
    await expect(dialog).toBeVisible();
    await settle(page);
    await close();
  });
});

test.describe("Masuk", () => {
  test("Enter with nothing typed says what to fill in and focuses the first field", async ({ browser }, info) => {
    const { page, close } = await open(browser, info, "anon");
    await visit(page, "/login");
    const username = page.getByLabel(/Nama pengguna|Email/i).first();
    await username.press("Enter");
    await expect(page.getByText(/Isi nama pengguna atau email/)).toBeVisible();
    await expect(username).toBeFocused();
    await close();
  });
});

test.describe("Peran dan akses", () => {
  test("Enter in the name field with nothing typed focuses the name and says what to fill in", async ({ browser }, info) => {
    const { page, close } = await open(browser, info, "owner");
    await visit(page, "/peran-akses");
    await page.getByRole("button", { name: "Buat peran" }).first().click();
    const drawer = page.getByRole("dialog");
    await expect(drawer).toBeVisible();
    const name = drawer.getByLabel(/Nama peran/);
    await expect(name).toBeFocused();
    await name.press("Enter");
    await expect(name).toHaveAttribute("aria-invalid", "true");
    await expect(drawer.getByRole("button", { name: "Buat peran" })).toBeEnabled();
    await close();
  });
});

test.describe("Tab order", () => {
  test("Tab never traps the keyboard on the new-project form", async ({ browser }, info) => {
    const { page, close } = await open(browser, info, "owner");
    await visit(page, "/projects/baru");
    await expect(page.getByLabel(/Nama mempelai wanita/)).toBeFocused(); // starts on the first field
    const seen = new Set<string>();
    for (let i = 0; i < 40; i++) {
      await page.keyboard.press("Tab");
      seen.add(await page.evaluate(() => document.activeElement?.getAttribute("aria-label") ?? document.activeElement?.id ?? document.activeElement?.tagName ?? ""));
    }
    expect(seen.size, "focus moves through many different controls").toBeGreaterThan(8);
    await close();
  });
});
