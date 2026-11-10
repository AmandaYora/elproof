import { screenshotSuite, open, tabTo } from "../lib/suite";
import { WO_DIALOGS, WO_PAGES } from "../lib/pages";
import { expect, shot, test, visit } from "../lib/fixtures";

screenshotSuite("wo", WO_PAGES, WO_DIALOGS);

// Keyboard focus on kit controls (T-20261110-11): the ring a person sees after Tab, on a button and on a text field.
// "WO Berbayar" has no trial banner, so no volatile mask lands across the dialog.
test.describe("wo: focus", () => {
  test("tombol", async ({ browser }, info) => {
    const { page, close } = await open(browser, info, "paying");
    await visit(page, "/vendors");
    await tabTo(page, page.getByRole("button", { name: "Tambah Vendor" }).first());
    await shot(page, info, "wo", "fokus-tombol", { fullPage: false });
    await close();
  });

  test("isian", async ({ browser }, info) => {
    const { page, close } = await open(browser, info, "paying");
    await visit(page, "/vendors");
    await tabTo(page, page.getByRole("button", { name: "Tambah Vendor" }).first());
    await page.keyboard.press("Space"); // Space acts on keyup: no Enter keydown can reach the dialog's first field and submit it
    const dialog = page.getByRole("dialog");
    await expect(dialog).toBeVisible();
    await tabTo(page, dialog.locator('input[data-slot="input"]').first());
    await shot(page, info, "wo", "fokus-isian", { fullPage: false });
    await close();
  });
});
