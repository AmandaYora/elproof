import crypto from "node:crypto";
import fs from "node:fs";
import path from "node:path";
import { test as base, expect, type BrowserContext, type Page, type TestInfo, type Locator } from "@playwright/test";
import AxeBuilder from "@axe-core/playwright";
import { FIXED_NOW, STATE_FILE, TMP, type State } from "./config";

export function state(): State {
  return JSON.parse(fs.readFileSync(STATE_FILE, "utf8"));
}

const FONT_CACHE = path.join(TMP, "fonts");

/**
 * The app loads Inter/Fraunces from Google Fonts. Serve them from a local cache after the first fetch so
 * screenshots do not depend on the network (and do not change when Google updates a file). When offline
 * and not cached, the request is aborted and the page falls back to the system font.
 */
async function installFontCache(context: BrowserContext) {
  fs.mkdirSync(FONT_CACHE, { recursive: true });
  await context.route(/^https:\/\/fonts\.(googleapis|gstatic)\.com\//, async (route) => {
    const key = crypto.createHash("sha1").update(route.request().url()).digest("hex");
    const body = path.join(FONT_CACHE, key);
    const meta = `${body}.json`;
    if (fs.existsSync(body) && fs.existsSync(meta)) {
      const { contentType } = JSON.parse(fs.readFileSync(meta, "utf8"));
      return route.fulfill({ status: 200, body: fs.readFileSync(body), headers: { "content-type": contentType, "access-control-allow-origin": "*" } });
    }
    try {
      const res = await route.fetch({ timeout: 8000 });
      const buf = await res.body();
      const contentType = res.headers()["content-type"] ?? "application/octet-stream";
      fs.writeFileSync(body, buf);
      fs.writeFileSync(meta, JSON.stringify({ contentType }));
      return route.fulfill({ response: res, body: buf });
    } catch {
      return route.abort();
    }
  });
}

/** Font cache + a browser clock fixed at 2026-11-09 10:00 WIB (new Date() never moves; timers keep running). */
export async function prepareContext(page: Page, opts: { freezeClock?: boolean } = {}) {
  await installFontCache(page.context());
  // The fake clock also stubs performance.getEntries*, so the performance baseline measures with the real clock.
  if (opts.freezeClock !== false) await page.clock.setFixedTime(FIXED_NOW);
}

export const test = base;
export { expect };

/** Waits until the page has stopped loading: network idle, no "Memuat…" text, fonts ready. */
export async function settle(page: Page) {
  await page.waitForLoadState("networkidle");
  await page
    .waitForFunction(() => !/Memuat|memuat/.test(document.body.innerText) && !document.querySelector('[class*="animate-pulse"]'), undefined, { timeout: 15_000 })
    .catch(() => undefined);
  await page.evaluate(() => document.fonts.ready);
  await page.waitForTimeout(250);
}

export async function visit(page: Page, route: string) {
  await page.goto(route);
  await settle(page);
}

/** Text that follows the real server clock (or the wall clock) and so cannot be pinned: masked in every shot. */
export function volatile(page: Page): Locator[] {
  return [
    page.getByText(/Masa coba .*tersisa/),
    page.getByText(/tersisa \d+ hari/),
    page.getByText(/Terlambat \d+ hari|terlambat \d+ hari/),
  ];
}

export interface ShotOptions {
  mask?: Locator[];
  fullPage?: boolean;
}

export async function shot(page: Page, info: TestInfo, group: string, name: string, opts: ShotOptions = {}) {
  const file = [group, `${name}-${info.project.name}.png`];
  await expect(page).toHaveScreenshot(file, { fullPage: opts.fullPage ?? true, mask: [...volatile(page), ...(opts.mask ?? [])] });
}

const AXE_TAGS = ["wcag2a", "wcag2aa", "wcag21a", "wcag21aa", "wcag22aa"];

/** Runs axe on the current page and attaches the result for the summary reporter. Never fails by itself. */
export async function scanAxe(page: Page, info: TestInfo, label: string) {
  // @axe-core/playwright bundles its own playwright-core (1.65) next to @playwright/test 1.64: same API, different type identity.
  const results = await new AxeBuilder({ page: page as never }).withTags(AXE_TAGS).analyze();
  const violations = results.violations.map((v) => ({
    rule: v.id,
    impact: v.impact ?? "minor",
    help: v.help,
    nodes: v.nodes.length,
    sample: v.nodes[0]?.target?.join(" ") ?? "",
  }));
  await info.attach("axe", { body: JSON.stringify({ page: label, viewport: info.project.name, violations }), contentType: "application/json" });
  if (process.env.AXE_STRICT === "1") {
    const blocking = violations.filter((v) => v.impact === "serious" || v.impact === "critical");
    expect(blocking, `axe serious/critical on ${label} (${info.project.name})`).toEqual([]);
  }
}
