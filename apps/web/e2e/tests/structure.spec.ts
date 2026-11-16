import { expect, prepareContext, visit, test } from "../lib/fixtures";
import { AUTH } from "../lib/config";
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

// QA FQ-05: on a 375 px phone no WO page scrolls sideways; a row that does not fit (tabs, chips) scrolls inside itself.
test("no WO page is wider than a 375 px phone", async ({ browser }, info) => {
  test.setTimeout(180_000);
  const context = await browser.newContext({
    ...info.project.use,
    viewport: { width: 375, height: 812 },
    isMobile: true,
    hasTouch: true,
    storageState: AUTH.owner,
  });
  const page = await context.newPage();
  await prepareContext(page);
  // Keuangan with proofs waiting: the count badge on its last tab (with a screen-reader word) once widened the page.
  await page.route(/\/api\/v1\/finance\/summary$/, async (route) => {
    const res = await route.fetch();
    const body = await res.json();
    body.data.pendingVerification = { ...body.data.pendingVerification, count: 128, amount: 1_234_567_890 };
    await route.fulfill({ response: res, json: body });
  });
  const routes = [
    "/beranda", "/proyek", "/proyek/1/ringkasan", "/proyek/1/keuangan/klien", "/keuangan", "/keuangan?tab=keluar", "/keuangan?tab=arus-kas",
    "/keuangan?tab=verifikasi", "/kalender", "/klien", "/vendor", "/venue", "/kategori-vendor", "/jadwal-bawaan", "/tim", "/peran-akses",
    "/profil-usaha", "/langganan",
  ];
  const wide: string[] = [];
  for (const route of routes) {
    await visit(page, route);
    const { scroll, view } = await page.evaluate(() => ({ scroll: document.documentElement.scrollWidth, view: document.documentElement.clientWidth }));
    if (scroll > view) wide.push(`${route}: ${scroll}px wide in a ${view}px window`);
  }
  expect(wide, wide.join("\n")).toEqual([]);
  await context.close();
});

// QA RT-06: an unknown address names itself in the tab and leads back to where the person can go.
test("the not-found page has its own title and a way back that fits who is looking", async ({ browser }, info) => {
  for (const [auth, label, href] of [
    ["anon", "Ke halaman utama", "/"],
    ["owner", "Kembali ke Beranda", "/beranda"],
    ["client", "Kembali ke Portal Klien", "/portal/beranda"],
  ] as const) {
    const { page, close } = await open(browser, info, auth);
    await visit(page, "/alamat-yang-tidak-ada");
    await expect(page.getByRole("heading", { level: 1, name: "Halaman tidak ditemukan" })).toBeVisible();
    await expect(page).toHaveTitle("Halaman tidak ditemukan — ElProof");
    await expect(page.getByRole("link", { name: label }), auth).toHaveAttribute("href", href);
    await expect(page.getByRole("link", { name: "Masuk" })).toHaveCount(auth === "anon" ? 1 : 0);
    await expect(page.getByText("Dashboard")).toHaveCount(0);
    await close();
  }
});

test("the menu and breadcrumbs are named landmarks on a project page", async ({ browser }, info) => {
  const { page, close } = await open(browser, info, "owner");
  await visit(page, "/proyek/1/ringkasan");
  await expect(page.getByRole("navigation", { name: "Menu utama" })).toHaveCount(1);
  await expect(page.getByRole("navigation", { name: "Jejak halaman" })).toHaveCount(1);
  await expect(page.getByRole("complementary", { name: "Menu" })).toHaveCount(1);
  await close();
});
