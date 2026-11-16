import { expect, visit, test } from "../lib/fixtures";
import { open } from "../lib/suite";
import { MARKETING_PAGES, PLATFORM_PAGES, PORTAL_PAGES, WO_PAGES, type GoldenPage } from "../lib/pages";

// PLAN Tahap 9, the part of the screen-reader test a machine can do: every page tells a person where they are. One h1, one main
// landmark, a tab title that names the page and differs from every other page of its console. (What is *read aloud*, and whether
// a label is right, still needs a person: UJI-MANUAL.md.)

const groups: Array<[string, GoldenPage[]]> = [
  ["wo", WO_PAGES],
  ["portal", PORTAL_PAGES],
  ["platform", PLATFORM_PAGES],
  ["marketing", MARKETING_PAGES],
];

for (const [group, pages] of groups) {
  test(`${group}: every page has one h1, one main and its own title`, async ({ browser }, info) => {
    test.setTimeout(180_000);
    const titles = new Map<string, string>();
    const problems: string[] = [];
    for (const p of pages) {
      const { page, close } = await open(browser, info, p.auth);
      await visit(page, p.route);
      const found = await page.evaluate(() => ({
        title: document.title.trim(),
        h1: Array.from(document.querySelectorAll("h1")).filter((h) => (h as HTMLElement).offsetParent !== null || getComputedStyle(h).position === "fixed").length,
        mains: document.querySelectorAll("main, [role=main]").length,
      }));
      if (found.h1 !== 1) problems.push(`${p.id}: ${found.h1} visible h1`);
      if (found.mains !== 1) problems.push(`${p.id}: ${found.mains} main landmarks`);
      if (!found.title || found.title === "ElProof") problems.push(`${p.id}: tab title is just "${found.title}"`);
      else if (titles.has(found.title)) problems.push(`${p.id}: same tab title as ${titles.get(found.title)} ("${found.title}")`);
      else titles.set(found.title, p.id);
      await close();
    }
    expect(problems, problems.join("\n")).toEqual([]);
  });
}

// QA RT-05: on a 375 px phone the Beranda was 382 px wide (the "Cara kerja" cards sized to their widest line and the preview
// tabs ran past the edge), so the page scrolled sideways. Every page a visitor can open before signing in must fit a narrow
// phone. The viewport is set without mobile emulation on purpose: with `isMobile` the browser widens the layout viewport to the
// content, and scrollWidth <= innerWidth would hold even for a page that overflows.
const PUBLIC_ROUTES = [
  ...MARKETING_PAGES.map((p) => p.route),
  "/kebijakan-privasi",
  "/perjanjian-pemrosesan-data",
  "/kebijakan-refund",
];

test("public pages: no sideways scroll at 360 and 375 px", async ({ browser }, info) => {
  test.setTimeout(180_000);
  const { page, close } = await open(browser, info, "anon");
  const problems: string[] = [];
  for (const width of [360, 375]) {
    await page.setViewportSize({ width, height: 800 });
    for (const route of PUBLIC_ROUTES) {
      await visit(page, route);
      const found = await page.evaluate(() => {
        const inner = window.innerWidth;
        // The outermost elements whose right edge passes the screen, to say what overflows (not only that something does).
        const culprits = Array.from(document.querySelectorAll<HTMLElement>("body *"))
          .filter((el) => {
            const box = el.getBoundingClientRect();
            const parent = el.parentElement?.getBoundingClientRect();
            return box.width > 0 && box.right > inner + 0.5 && !(parent && parent.right > inner + 0.5);
          })
          .slice(0, 3)
          .map((el) => `${el.tagName.toLowerCase()} "${(el.textContent ?? "").trim().slice(0, 30)}"`);
        return { scroll: document.documentElement.scrollWidth, inner, culprits };
      });
      if (found.scroll > found.inner) problems.push(`${route} at ${width} px: scrollWidth ${found.scroll} > ${found.inner} (${found.culprits.join(", ")})`);
    }
  }
  expect(problems, problems.join("\n")).toEqual([]);
  await close();
});

test("the menu and breadcrumbs are named landmarks on a project page", async ({ browser }, info) => {
  const { page, close } = await open(browser, info, "owner");
  await visit(page, "/proyek/1/ringkasan");
  await expect(page.getByRole("navigation", { name: "Menu utama" })).toHaveCount(1);
  await expect(page.getByRole("navigation", { name: "Jejak halaman" })).toHaveCount(1);
  await expect(page.getByRole("complementary", { name: "Menu" })).toHaveCount(1);
  await close();
});
