import type { Browser, Page, TestInfo } from "@playwright/test";
import { AUTH } from "./config";
import { expect, prepareContext, settle, shot, test, visit } from "./fixtures";
import type { AuthKey, GoldenDialog, GoldenPage } from "./pages";

/** A fresh browser context (own storage) for one auth role, with the clock and fonts prepared. */
export async function open(browser: Browser, info: TestInfo, auth: AuthKey): Promise<{ page: Page; close: () => Promise<void> }> {
  const context = await browser.newContext({ ...info.project.use, storageState: AUTH[auth] });
  const page = await context.newPage();
  await prepareContext(page);
  return { page, close: () => context.close() };
}

/** Registers the screenshot tests of one console: normal, empty, error and dialog states. */
export function screenshotSuite(group: string, pages: GoldenPage[], dialogs: GoldenDialog[] = []) {
  test.describe(`${group}: normal`, () => {
    for (const p of pages) {
      test(p.id, async ({ browser }, info) => {
        const { page, close } = await open(browser, info, p.auth);
        await visit(page, p.route);
        await shot(page, info, group, `${p.id}-normal`);
        await close();
      });
    }
  });

  test.describe(`${group}: empty workspace`, () => {
    for (const p of pages.filter((x) => x.empty)) {
      test(p.id, async ({ browser }, info) => {
        const { page, close } = await open(browser, info, "empty");
        await visit(page, p.route);
        await shot(page, info, group, `${p.id}-empty`);
        await close();
      });
    }
  });

  test.describe(`${group}: error`, () => {
    for (const p of pages.filter((x) => x.fail?.length)) {
      test(p.id, async ({ browser }, info) => {
        const { page, close } = await open(browser, info, p.auth);
        await page.route(
          (url) => p.fail!.some((prefix) => url.pathname.startsWith(prefix)),
          (route) => route.abort("failed")
        );
        await visit(page, p.route);
        await shot(page, info, group, `${p.id}-error`);
        await close();
      });
    }
  });

  test.describe(`${group}: dialog`, () => {
    for (const d of dialogs) {
      test(d.id, async ({ browser }, info) => {
        const { page, close } = await open(browser, info, d.auth);
        await visit(page, d.route);
        await d.open(page);
        await expect(page.getByRole("dialog").first()).toBeVisible();
        await settle(page);
        await shot(page, info, group, `${d.id}-dialog`, { fullPage: false });
        await close();
      });
    }
  });
}

/** Presses Tab until `target` has focus (keyboard modality, so :focus-visible applies). Fails after `max` presses. */
export async function tabTo(page: Page, target: import("@playwright/test").Locator, max = 80) {
  for (let i = 0; i < max; i++) {
    if (await target.evaluate((el) => el === document.activeElement).catch(() => false)) return;
    await page.keyboard.press("Tab");
  }
  await expect(target).toBeFocused();
}

/** The focus ring a kit control draws: style, width and colour of its outline, and the --ring token for comparison. */
export async function focusRing(target: import("@playwright/test").Locator) {
  return target.evaluate((el) => {
    const cs = getComputedStyle(el);
    const probe = document.createElement("span");
    probe.style.color = "var(--ring)";
    document.body.appendChild(probe);
    const ring = getComputedStyle(probe).color;
    probe.remove();
    return { style: cs.outlineStyle, width: parseFloat(cs.outlineWidth), color: cs.outlineColor, ring };
  });
}
