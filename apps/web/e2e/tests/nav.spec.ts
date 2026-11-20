import { expect, visit, test } from "../lib/fixtures";
import { open } from "../lib/suite";

// PLAN Tahap 7: neutral menu with Direktori/Pengaturan groups, Cmd/Ctrl+B, a sheet on a phone, breadcrumbs and Cmd/Ctrl+K.

test.describe("menu", () => {
  test("Direktori holds Vendor and Venue and opens on the page it contains", async ({ browser }, info) => {
    const { page, close } = await open(browser, info, "owner");
    await visit(page, "/beranda");
    const menu = page.getByRole("navigation", { name: "Menu utama" });
    await expect(menu.getByRole("link", { name: "Proyek" })).toBeVisible();
    await expect(menu.getByRole("link", { name: "Klien" })).toBeVisible();
    const directory = menu.getByRole("button", { name: "Direktori" });
    await expect(directory).toHaveAttribute("aria-expanded", "false");
    await directory.click();
    await expect(menu.getByRole("link", { name: "Vendor" })).toBeVisible();
    await expect(menu.getByRole("link", { name: "Venue" })).toBeVisible();
    // Landing straight on a page of the group opens it.
    await visit(page, "/vendor");
    await expect(page.getByRole("navigation", { name: "Menu utama" }).getByRole("button", { name: "Direktori" })).toHaveAttribute("aria-expanded", "true");
    await expect(page.getByRole("navigation", { name: "Menu utama" }).getByRole("link", { name: "Vendor" })).toHaveAttribute("aria-current", "page");
    await close();
  });

  test("Ctrl+B narrows the menu to icons and the choice is remembered", async ({ browser }, info) => {
    const { page, close } = await open(browser, info, "owner");
    await visit(page, "/beranda");
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
    await visit(page, "/beranda");
    await page.getByRole("button", { name: "Buka menu" }).click();
    const sheet = page.getByRole("dialog", { name: "Menu" });
    await expect(sheet).toBeVisible();
    await sheet.getByRole("link", { name: "Proyek" }).click();
    await expect(page).toHaveURL(/\/proyek$/);
    await expect(sheet).toBeHidden();
    await close();
  });
});

test.describe("breadcrumbs", () => {
  test("a project page shows Proyek > its name, and the first is a link back", async ({ browser }, info) => {
    const { page, close } = await open(browser, info, "owner");
    await visit(page, "/proyek/1/ringkasan");
    const trail = page.getByRole("navigation", { name: "Jejak halaman" });
    await expect(trail.getByRole("link", { name: "Proyek" })).toBeVisible();
    await expect(trail.getByText("Citra & Bima")).toHaveAttribute("aria-current", "page");
    await trail.getByRole("link", { name: "Proyek" }).click();
    await expect(page).toHaveURL(/\/proyek$/);
    await close();
  });
});

test.describe("palette Ctrl+K", () => {
  test("finds a project from the keyboard and goes to it", async ({ browser }, info) => {
    const { page, close } = await open(browser, info, "owner");
    await visit(page, "/beranda");
    await page.keyboard.press("Control+k");
    const dialog = page.getByRole("dialog", { name: "Pencarian" });
    await expect(dialog).toBeVisible();
    await page.keyboard.type("Citra");
    // The record search goes to the API after a short debounce; under a full parallel run that can take a few seconds.
    await expect(dialog.getByRole("option", { name: /Citra & Bima/ })).toBeVisible({ timeout: 15_000 });
    await page.keyboard.press("Enter");
    await expect(page).toHaveURL(/\/proyek\/1/);
    await expect(dialog).toBeHidden();
    await close();
  });

  test("lists pages too, and Escape closes it with focus back where it was", async ({ browser }, info) => {
    const { page, close } = await open(browser, info, "owner");
    await visit(page, "/beranda");
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
    await visit(page, "/proyek/1/vendor");
    const table = page.getByRole("table", { name: "Vendor & venue" });
    const row = table.getByRole("row").filter({ hasText: "Katering Rasa Nusantara" });
    const opener = row.getByRole("link", { name: /Katering Rasa Nusantara/ });
    expect(await opener.locator("a, button, [role=button]").count()).toBe(0);
    const actions = row.getByRole("button", { name: "Aksi lain untuk Katering Rasa Nusantara" });
    await actions.focus();
    await page.keyboard.press("Enter");
    await expect(page.getByRole("menu")).toBeVisible();
    // Using the actions does not also open the vendor's drawer.
    await page.keyboard.press("Escape");
    await expect(page.getByRole("menu")).toHaveCount(0);
    await expect(page.getByRole("dialog")).toHaveCount(0);
    // The name opens it.
    await opener.click();
    await expect(page.getByRole("dialog", { name: /Katering Rasa Nusantara/ })).toBeVisible();
    await close();
  });
});

test.describe("harga and Daftar (EP-14)", () => {
  const BASIC = { id: 1, name: "Basic", durationMonths: 1, price: 149000, features: ["Semua fitur"] };
  const plans = (data: unknown[]) => ({ status: 200, contentType: "application/json", body: JSON.stringify({ success: true, message: "ok", data }) });
  const PLANS_API = /\/api\/v1\/public\/plans$/;

  test("the price heading follows the plans on sale and Pilih paket carries the plan to Daftar", async ({ browser }, info) => {
    const { page, close } = await open(browser, info, "anon");
    await page.route(PLANS_API, (route) => route.fulfill(plans([BASIC])));
    await visit(page, "/");
    const harga = page.locator("#harga");
    // One monthly plan: no "pilih lamanya".
    await expect(harga.getByRole("heading", { level: 2 })).toHaveText("Harga terbuka, satu paket");
    await expect(harga).toContainText("lanjutkan dengan paket Basic, ditagih per bulan");
    await harga.getByRole("link", { name: "Pilih paket Basic" }).click();
    await expect(page).toHaveURL(/\/daftar\?paket=1$/);
    const summary = page.getByRole("region", { name: "Paket pilihan Anda" });
    await expect(summary).toContainText("Basic");
    await expect(summary).toContainText("Rp149.000 per bulan");
    await expect(summary).toContainText("coba gratis 14 hari");
    await expect(summary.getByRole("link", { name: "Ganti paket" })).toHaveAttribute("href", "/#harga");
    await close();
  });

  test("several lengths keep 'pilih lamanya'; an unknown plan id leaves Daftar the plain trial signup", async ({ browser }, info) => {
    const { page, close } = await open(browser, info, "anon");
    const yearly = { id: 2, name: "Tahunan", durationMonths: 12, price: 1490000, features: ["Semua fitur"] };
    await page.route(PLANS_API, (route) => route.fulfill(plans([BASIC, yearly])));
    await visit(page, "/");
    await expect(page.locator("#harga").getByRole("heading", { level: 2 })).toHaveText("Harga terbuka, pilih lamanya");
    const answered = page.waitForResponse(PLANS_API);
    await visit(page, "/daftar?paket=99");
    await answered;
    await expect(page.getByRole("heading", { level: 1 })).toHaveText("Coba ElProof gratis 14 hari");
    await expect(page.getByRole("region", { name: "Paket pilihan Anda" })).toHaveCount(0);
    await close();
  });
});

