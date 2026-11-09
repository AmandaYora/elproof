import { scanAxe, settle, test, visit } from "../lib/fixtures";
import { open } from "../lib/suite";
import { MARKETING_PAGES, PLATFORM_DIALOGS, PLATFORM_PAGES, PORTAL_PAGES, WO_DIALOGS, WO_PAGES } from "../lib/pages";

// Records WCAG violations of every golden page and every dialog. By default it never fails (the old UI has known
// failures; the baseline is the point). AXE_STRICT=1 fails on serious/critical so the gate can be switched on later.
const groups = [
  ["wo", WO_PAGES, WO_DIALOGS],
  ["portal", PORTAL_PAGES, []],
  ["platform", PLATFORM_PAGES, PLATFORM_DIALOGS],
  ["marketing", MARKETING_PAGES, []],
] as const;

for (const [group, pages, dialogs] of groups) {
  test.describe(`axe ${group}`, () => {
    for (const p of pages) {
      test(p.id, async ({ browser }, info) => {
        const { page, close } = await open(browser, info, p.auth);
        await visit(page, p.route);
        await scanAxe(page, info, `${group}/${p.id}`);
        await close();
      });
    }
    for (const d of dialogs) {
      test(`${d.id} (dialog)`, async ({ browser }, info) => {
        const { page, close } = await open(browser, info, d.auth);
        await visit(page, d.route);
        await d.open(page);
        await page.getByRole("dialog").first().waitFor();
        await settle(page);
        await scanAxe(page, info, `${group}/${d.id} (dialog)`);
        await close();
      });
    }
  });
}
