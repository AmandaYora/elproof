import { expect, visit, test } from "../lib/fixtures";
import { open } from "../lib/suite";

// PLAN Tahap 7: neutral menu with Direktori/Pengaturan groups, Cmd/Ctrl+B, a sheet on a phone, breadcrumbs and Cmd/Ctrl+K.

test.describe("menu", () => {
  test("Direktori holds Vendor and Venue and opens on the page it contains", async ({ browser }, info) => {
    const { page, close } = await open(browser, info, "owner");
    await visit(page, "/dashboard");
    const menu = page.getByRole("navigation", { name: "Menu utama" });
    await expect(menu.getByRole("link", { name: "Proyek" })).toBeVisible();
    await expect(menu.getByRole("link", { name: "Klien" })).toBeVisible();
    const directory = menu.getByRole("button", { name: "Direktori" });
    await expect(directory).toHaveAttribute("aria-expanded", "false");
    await directory.click();
    await expect(menu.getByRole("link", { name: "Vendor" })).toBeVisible();
    await expect(menu.getByRole("link", { name: "Venue" })).toBeVisible();
    // Landing straight on a page of the group opens it.
    await visit(page, "/vendors");
    await expect(page.getByRole("navigation", { name: "Menu utama" }).getByRole("button", { name: "Direktori" })).toHaveAttribute("aria-expanded", "true");
    await expect(page.getByRole("navigation", { name: "Menu utama" }).getByRole("link", { name: "Vendor" })).toHaveAttribute("aria-current", "page");
    await close();
  });

  test("Ctrl+B narrows the menu to icons and the choice is remembered", async ({ browser }, info) => {
    const { page, close } = await open(browser, info, "owner");
    await visit(page, "/dashboard");
    const aside = page.getByRole("complementary", { name: "Menu" });
    const wide = (await aside.boundingBox())!.width;
    await page.keyboard.press("Control+b");
    await expect.poll(async () => (await aside.boundingBox())!.width).toBeLessThan(wide / 2);
    // Every item still has a name for assistive technology and a tooltip target.
    await expect(aside.getByRole("link", { name: "Proyek" })).toBeVisible();
    await page.reload();
    await expect.poll(async () => (await page.getByRole("complementary", { name: "Menu" }).boundingBox())!.width).toBeLessThan(wide / 2);
    await page.keyboard.press("Control+b");
    await expect.poll(async () => (await page.getByRole("complementary", { name: "Menu" }).boundingBox())!.width).toBeGreaterThan(wide - 4);
    await close();
  });

  test("on a phone the menu is a sheet that closes after choosing a page", async ({ browser }, info) => {
    const { page, close } = await open(browser, info, "owner");
    await page.setViewportSize({ width: 375, height: 812 });
    await visit(page, "/dashboard");
    await page.getByRole("button", { name: "Buka menu" }).click();
    const sheet = page.getByRole("dialog", { name: "Menu" });
    await expect(sheet).toBeVisible();
    await sheet.getByRole("link", { name: "Proyek" }).click();
    await expect(page).toHaveURL(/\/projects$/);
    await expect(sheet).toBeHidden();
    await close();
  });
});

test.describe("breadcrumbs", () => {
  test("a project page shows Proyek > its name, and the first is a link back", async ({ browser }, info) => {
    const { page, close } = await open(browser, info, "owner");
    await visit(page, "/projects/1/ringkasan");
    const trail = page.getByRole("navigation", { name: "Jejak halaman" });
    await expect(trail.getByRole("link", { name: "Proyek" })).toBeVisible();
    await expect(trail.getByText("Citra & Bima")).toHaveAttribute("aria-current", "page");
    await trail.getByRole("link", { name: "Proyek" }).click();
    await expect(page).toHaveURL(/\/projects$/);
    await close();
  });
});

test.describe("palette Ctrl+K", () => {
  test("finds a project from the keyboard and goes to it", async ({ browser }, info) => {
    const { page, close } = await open(browser, info, "owner");
    await visit(page, "/dashboard");
    await page.keyboard.press("Control+k");
    const dialog = page.getByRole("dialog", { name: "Pencarian" });
    await expect(dialog).toBeVisible();
    await page.keyboard.type("Citra");
    await expect(dialog.getByRole("option", { name: /Citra & Bima/ })).toBeVisible();
    await page.keyboard.press("Enter");
    await expect(page).toHaveURL(/\/projects\/1/);
    await expect(dialog).toBeHidden();
    await close();
  });

  test("lists pages too, and Escape closes it with focus back where it was", async ({ browser }, info) => {
    const { page, close } = await open(browser, info, "owner");
    await visit(page, "/dashboard");
    const trigger = page.getByRole("button", { name: /Cari proyek, vendor, atau klien/ });
    await trigger.focus();
    await page.keyboard.press("Enter");
    const dialog = page.getByRole("dialog", { name: "Pencarian" });
    await expect(dialog.getByRole("option", { name: "Proyek" })).toBeVisible();
    await page.keyboard.press("Escape");
    await expect(dialog).toBeHidden();
    await expect(trigger).toBeFocused();
    await close();
  });
});

test.describe("vendor rows", () => {
  // Axe flagged the row actions as controls nested inside the toggle: they were unnamed and out of keyboard reach.
  test("a row's actions are named buttons beside the toggle, not inside it", async ({ browser }, info) => {
    const { page, close } = await open(browser, info, "owner");
    await visit(page, "/projects/1/vendor");
    const toggle = page.getByRole("button", { name: /Katering Rasa Nusantara/ }).first();
    await expect(toggle).toHaveAttribute("aria-expanded", "false");
    expect(await toggle.locator("button, [role=button]").count()).toBe(0);
    const edit = page.getByRole("button", { name: "Ubah Katering Rasa Nusantara" });
    await edit.focus();
    await page.keyboard.press("Enter");
    await expect(page.getByRole("dialog", { name: "Ubah vendor di proyek" })).toBeVisible();
    // Using an action does not also open or close the row.
    await page.keyboard.press("Escape");
    await expect(toggle).toHaveAttribute("aria-expanded", "false");
    await close();
  });
});

