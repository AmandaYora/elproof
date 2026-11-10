import { screenshotSuite, open, tabTo } from "../lib/suite";
import { WO_DIALOGS, WO_PAGES } from "../lib/pages";
import { expect, shot, test, visit } from "../lib/fixtures";

screenshotSuite("wo", WO_PAGES, WO_DIALOGS);

// Keyboard focus on kit controls (T-20261110-11): the ring a person sees after Tab, on a button and on a text field.
test.describe("wo: focus", () => {
  test("tombol", async ({ browser }, info) => {
    const { page, close } = await open(browser, info, "owner");
    await visit(page, "/vendors");
    await tabTo(page, page.getByRole("button", { name: "Tambah Vendor" }));
    await shot(page, info, "wo", "fokus-tombol", { fullPage: false });
    await close();
  });

  test("isian", async ({ browser }, info) => {
    const { page, close } = await open(browser, info, "owner");
    await visit(page, "/vendors");
    await tabTo(page, page.getByRole("button", { name: "Tambah Vendor" }));
    await page.keyboard.press("Enter");
    const dialog = page.getByRole("dialog");
    await expect(dialog).toBeVisible();
    await tabTo(page, dialog.locator('input[data-slot="input"]').first());
    await shot(page, info, "wo", "fokus-isian", { fullPage: false });
    await close();
  });
});
