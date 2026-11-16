import { AUTH } from "../lib/config";
import { expect, prepareContext, settle, test, visit } from "../lib/fixtures";
import { open } from "../lib/suite";

// Kalender (RISET WO Console §3 #2, §7.7). Seed: "Citra & Bima" has its wedding on 14 Mar 2027 and an installment
// "Termin 2" due on 25 Oct 2026 (day -15 from the seed's base day), still unpaid, so overdue. The browser clock is
// fixed at 9 Nov 2026, so "this month" is November 2026.

test.describe("Kalender", () => {
  test("owner sees the wedding day and a term due date in the right month, and the grid moves by keyboard", async ({ browser }, info) => {
    const { page, close } = await open(browser, info, "owner");
    await visit(page, "/kalender");
    await expect(page.getByRole("heading", { level: 1, name: "Kalender" })).toBeVisible();
    // Desktop opens the month grid, labelled by the shown month.
    const grid = page.getByRole("grid", { name: "November 2026" });
    await expect(grid).toBeVisible();
    await expect(grid.locator('[aria-current="date"]')).toHaveAttribute("data-day", "2026-11-09");

    // The wedding day sits on 14 March 2027 and opens the project.
    await visit(page, "/kalender?bulan=2027-03");
    const wedding = page.locator('[data-day="2027-03-14"] a[data-kind="event"]');
    await expect(wedding).toHaveAccessibleName(/^Citra & Bima ?, hari acara, Persiapan$/);
    await expect(wedding).toHaveAttribute("href", /\/proyek\/\d+\/ringkasan$/);
    await expect(page.locator('[data-day="2027-03-13"] a[data-kind="event"]')).toHaveCount(0);

    // Termin 2 is due on 25 October 2026 and is overdue (status in words, not colour alone).
    await visit(page, "/kalender?bulan=2026-10");
    const term = page.locator('[data-day="2026-10-25"] a[data-kind="payment"]');
    // Its name starts with what is on screen (WCAG 2.5.3) and adds the installment and amount.
    await expect(term).toHaveAccessibleName(/^Terlambat · Citra & Bima ?, termin Termin 2, Rp40\.000\.000, Terlambat$/);

    // Keyboard: one Tab stop on the active day, arrows move it, Enter opens the day panel with the entries as links.
    const active = page.locator('td[role="gridcell"][tabindex="0"]');
    await expect(active).toHaveCount(1);
    await active.focus();
    await expect(active).toHaveAttribute("data-day", "2026-10-01");
    for (let i = 0; i < 24; i++) await page.keyboard.press("ArrowRight");
    await expect(page.locator(':focus')).toHaveAttribute("data-day", "2026-10-25");
    await page.keyboard.press("Enter");
    const panel = page.getByRole("dialog", { name: "Minggu, 25 Oktober 2026" });
    await expect(panel).toBeVisible();
    await expect(panel.getByRole("link", { name: /Termin 2/ })).toBeVisible();
    await page.keyboard.press("Escape");
    await expect(panel).toBeHidden();
    // Focus returns to the day; PageDown moves to the same day of the next month.
    await expect(page.locator(":focus")).toHaveAttribute("data-day", "2026-10-25");
    await page.keyboard.press("PageDown");
    await expect(page.getByRole("grid", { name: "November 2026" })).toBeVisible();
    await expect(page.locator(":focus")).toHaveAttribute("data-day", "2026-11-25");

    // The kind filter is applied by the API: only payments asked for, the wedding gone.
    await visit(page, "/kalender?bulan=2027-03&jenis=payment");
    await expect(page.locator('a[data-kind="event"]')).toHaveCount(0);
    await expect(page.getByText("Tidak ada agenda di Maret 2027")).toBeVisible();
    await close();
  });

  test("a phone opens the agenda list", async ({ browser }, info) => {
    const context = await browser.newContext({
      ...info.project.use,
      storageState: AUTH.owner,
      viewport: { width: 375, height: 812 },
      isMobile: true,
      hasTouch: true,
    });
    const page = await context.newPage();
    await prepareContext(page);
    await page.goto("/kalender?bulan=2027-03");
    await settle(page);
    await expect(page.getByRole("grid")).toHaveCount(0);
    await expect(page.getByRole("button", { name: "Agenda", pressed: true })).toBeVisible();
    const day = page.getByRole("listitem").filter({ has: page.getByRole("heading", { name: /Minggu, 14 Maret/ }) });
    const wedding = day.locator('a[data-kind="event"]');
    await expect(wedding).toContainText("Citra & Bima");
    await expect(wedding).toContainText("Hari acara");
    await context.close();
  });
});
