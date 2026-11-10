import { expect, settle, test, visit } from "../lib/fixtures";
import { open } from "../lib/suite";

// Behavior of the overlay layer (PLAN Tahap 2): focus goes in and comes back, Escape closes only the top layer,
// nothing closes while a save runs, and a failure is shown inside the dialog.

test.describe("modal", () => {
  test("focus goes to the first field, Escape closes, focus returns to the opener", async ({ browser }, info) => {
    const { page, close } = await open(browser, info, "owner");
    await visit(page, "/vendors");
    const opener = page.getByRole("button", { name: "Tambah Vendor" });
    await opener.focus();
    await page.keyboard.press("Enter");
    const dialog = page.getByRole("dialog");
    await expect(dialog).toBeVisible();
    await expect(dialog.getByLabel(/Nama Vendor/)).toBeFocused();
    await page.keyboard.press("Escape");
    await expect(dialog).toBeHidden();
    await expect(opener).toBeFocused();
    await close();
  });

  test("Tab stays inside the dialog", async ({ browser }, info) => {
    const { page, close } = await open(browser, info, "owner");
    await visit(page, "/vendors");
    await page.getByRole("button", { name: "Tambah Vendor" }).click();
    const dialog = page.getByRole("dialog");
    await expect(dialog).toBeVisible();
    for (let i = 0; i < 25; i++) {
      await page.keyboard.press("Tab");
      // Focus guards at the edges hand focus back a moment after Tab, so poll instead of reading once.
      await expect
        .poll(() => page.evaluate(() => !!document.activeElement?.closest('[role="dialog"]')), { message: `Tab #${i + 1} left the dialog`, timeout: 2000 })
        .toBe(true);
    }
    await close();
  });
});

test.describe("action menu", () => {
  test("keyboard: Enter opens, Escape closes only the menu; choosing an item opens a dialog that keeps focus", async ({ browser }, info) => {
    const { page, close } = await open(browser, info, "owner");
    await visit(page, "/projects/1/ringkasan");
    const trigger = page.getByRole("button", { name: "Aksi lainnya" });
    await trigger.focus();
    await page.keyboard.press("Enter");
    const menu = page.getByRole("menu");
    await expect(menu).toBeVisible();
    await expect(page.getByRole("menuitem").first()).toBeFocused();
    await page.keyboard.press("ArrowDown");
    await page.keyboard.press("Escape");
    await expect(menu).toBeHidden();
    await expect(trigger).toBeFocused();

    await page.keyboard.press("Enter");
    await page.getByRole("menuitem", { name: "Duplikat proyek" }).click();
    const dialog = page.getByRole("dialog");
    await expect(dialog).toBeVisible();
    // The menu returning focus to its button must not steal it from the dialog.
    await settle(page);
    expect(await page.evaluate(() => !!document.activeElement?.closest('[role="dialog"]'))).toBe(true);
    await close();
  });
});

test.describe("confirm dialog", () => {
  test("a failed action shows its reason inside the dialog; Escape closes it and focus returns to the opener", async ({ browser }, info) => {
    const { page, close } = await open(browser, info, "admin");
    await page.route(/\/api\/v1\/console\/announcements\/\d+$/, (route) =>
      route.request().method() !== "DELETE"
        ? route.fallback()
        : route.fulfill({ status: 500, contentType: "application/json", body: JSON.stringify({ success: false, message: "Server sedang bermasalah", errors: null }) }),
    );
    await visit(page, "/platform/pengumuman");
    const opener = page.getByRole("button", { name: /^Hapus/ }).first();
    await opener.click();
    const dialog = page.getByRole("alertdialog");
    await expect(dialog).toBeVisible();
    // Safe default: focus starts on "Batal", never on the destructive action.
    await expect(dialog.getByRole("button", { name: "Batal" })).toBeFocused();
    await dialog.getByRole("button", { name: "Hapus pengumuman" }).click();
    await expect(dialog.getByRole("alert")).toContainText(/belum bisa|bermasalah|gagal/i);
    await expect(dialog).toBeVisible();
    await page.keyboard.press("Escape");
    await expect(dialog).toBeHidden();
    await expect(opener).toBeFocused();
    await close();
  });

  test("an outside click does not close it", async ({ browser }, info) => {
    const { page, close } = await open(browser, info, "admin");
    await visit(page, "/platform/pengumuman");
    await page.getByRole("button", { name: /^Hapus/ }).first().click();
    const dialog = page.getByRole("alertdialog");
    await expect(dialog).toBeVisible();
    await page.mouse.click(5, 5);
    await expect(dialog).toBeVisible();
    await close();
  });
});

test.describe("drawer", () => {
  test("nothing closes it while a save runs; a failure shows inside; then Escape closes it", async ({ browser }, info) => {
    const { page, close } = await open(browser, info, "owner");
    await page.route(/\/api\/v1\/tenants\/me\/business-details$/, async (route) => {
      if (route.request().method() !== "PUT") return route.fallback();
      await new Promise((r) => setTimeout(r, 900));
      return route.fulfill({ status: 500, contentType: "application/json", body: JSON.stringify({ success: false, message: "Server sedang bermasalah", errors: null }) });
    });
    await visit(page, "/profil-usaha");
    const opener = page.getByRole("button", { name: "Ubah" }).last();
    await opener.click();
    const drawer = page.getByRole("dialog");
    await expect(drawer).toBeVisible();
    await drawer.getByRole("button", { name: "Simpan" }).click();
    await expect(drawer.getByRole("button", { name: "Menyimpan..." })).toBeVisible();
    // In flight: Escape, a click on the dim area and the close button all do nothing.
    await page.keyboard.press("Escape");
    await page.mouse.click(5, 5);
    await expect(drawer.getByRole("button", { name: "Tutup" })).toBeDisabled();
    await expect(drawer).toBeVisible();
    // The save fails: the reason is inside the drawer, which stays open with the user's input.
    await expect(drawer.getByRole("alert")).toContainText(/belum tersimpan|bermasalah/i);
    await expect(drawer).toBeVisible();
    await page.keyboard.press("Escape");
    await expect(drawer).toBeHidden();
    await close();
  });
});
