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

test("the menu and breadcrumbs are named landmarks on a project page", async ({ browser }, info) => {
  const { page, close } = await open(browser, info, "owner");
  await visit(page, "/projects/1/ringkasan");
  await expect(page.getByRole("navigation", { name: "Menu utama" })).toHaveCount(1);
  await expect(page.getByRole("navigation", { name: "Jejak halaman" })).toHaveCount(1);
  await expect(page.getByRole("complementary", { name: "Menu" })).toHaveCount(1);
  await close();
});
