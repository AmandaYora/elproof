import { expect, visit, test } from "../lib/fixtures";
import { open } from "../lib/suite";

// PLAN Tahap 6: list state lives in the URL, headers sort (aria-sort, keyboard), money is right aligned, an empty
// result offers "Reset filter", and no list loads every row (all=true) just to filter or sort in the browser.

// MySQL's collation ignores case and punctuation; compare the same way.
const compare = (x: string, y: string) => x.localeCompare(y, "id", { sensitivity: "base", ignorePunctuation: true });

test.describe("Vendor list", () => {
  test("sorting is a header button, shows in aria-sort and the URL, and survives a reload", async ({ browser }, info) => {
    const { page, close } = await open(browser, info, "owner");
    await visit(page, "/vendors");
    const header = page.getByRole("columnheader", { name: /Nama vendor/ });
    await expect(header).toHaveAttribute("aria-sort", "none");

    // The order is judged on what the server sent, not on the first row on screen: other tests add vendors to this workspace
    // at the same moment, so a row can appear between two looks at the table.
    const sentNames = async (direction: "asc" | "desc") => {
      const answer = page.waitForResponse((r) => r.request().method() === "GET" && r.url().includes("/vendors") && new RegExp(`sort=name(:|%3A)${direction}`).test(r.url()));
      await header.getByRole("button").click();
      const names = ((await (await answer).json()).data as Array<{ name: string }>).map((v) => v.name.toLowerCase());
      await expect(header).toHaveAttribute("aria-sort", direction === "asc" ? "ascending" : "descending");
      return names;
    };
    const asc = await sentNames("asc");
    await expect(page).toHaveURL(/sort=name%3Aasc/);
    expect(asc.length).toBeGreaterThan(1);
    expect(asc).toEqual([...asc].sort(compare));

    const desc = await sentNames("desc");
    expect(desc).toEqual([...desc].sort((x, y) => compare(y, x)));

    // Reload keeps the sort: it is in the address, not in memory.
    await page.reload();
    await expect(page.getByRole("columnheader", { name: /Nama vendor/ })).toHaveAttribute("aria-sort", "descending");

    // Third press returns to the list's own order and clears the address.
    await page.getByRole("columnheader", { name: /Nama vendor/ }).getByRole("button").click();
    await expect(page).not.toHaveURL(/sort=/);
    await close();
  });

  test("the sort button works from the keyboard", async ({ browser }, info) => {
    const { page, close } = await open(browser, info, "owner");
    await visit(page, "/vendors");
    const button = page.getByRole("columnheader", { name: /Kota/ }).getByRole("button");
    await button.focus();
    await page.keyboard.press("Enter");
    await expect(page.getByRole("columnheader", { name: /Kota/ })).toHaveAttribute("aria-sort", "ascending");
    await page.keyboard.press("Space");
    await expect(page.getByRole("columnheader", { name: /Kota/ })).toHaveAttribute("aria-sort", "descending");
    await close();
  });

  test("a search with no match says so and Reset filter brings the list back", async ({ browser }, info) => {
    const { page, close } = await open(browser, info, "owner");
    await visit(page, "/vendors");
    await page.getByRole("searchbox").or(page.getByPlaceholder(/Cari nama vendor/)).first().fill("zzzz-tidak-ada");
    await expect(page.getByText("Tidak ada hasil")).toBeVisible();
    await expect(page).toHaveURL(/q=zzzz/);
    await page.getByRole("button", { name: "Reset filter" }).click();
    await expect(page.getByText("Tidak ada hasil")).toBeHidden();
    await expect(page).not.toHaveURL(/q=/);
    await expect(page.getByRole("row").nth(1)).toBeVisible();
    await close();
  });

  test("money is right aligned in tabular figures", async ({ browser }, info) => {
    const { page, close } = await open(browser, info, "owner");
    await visit(page, "/vendors");
    const cell = page.getByRole("row").nth(1).getByRole("cell").filter({ hasText: /^Rp/ }).first();
    await expect(cell).toBeVisible();
    const style = await cell.evaluate((el) => ({ align: getComputedStyle(el).textAlign, nums: getComputedStyle(el).fontVariantNumeric }));
    expect(style.align).toBe("right");
    expect(style.nums).toContain("tabular-nums");
    await close();
  });
});

test.describe("no list loads every row to filter in the browser", () => {
  for (const [name, route, endpoint] of [
    ["Proyek", "/projects", /\/api\/v1\/projects(\?|$)/],
    ["Klien", "/clients", /\/api\/v1\/clients(\?|$)/],
    ["Vendor", "/vendors", /\/api\/v1\/vendors(\?|$)/],
    ["Venue", "/venues", /\/api\/v1\/venues(\?|$)/],
    ["Tim", "/pengguna", /\/api\/v1\/staff(\?|$)/],
  ] as const) {
    test(name, async ({ browser }, info) => {
      const { page, close } = await open(browser, info, "owner");
      const urls: string[] = [];
      page.on("request", (r) => {
        if (r.method() === "GET" && endpoint.test(new URL(r.url()).pathname + new URL(r.url()).search)) urls.push(r.url());
      });
      await visit(page, route);
      expect(urls.length, `${name} made no request to its list endpoint`).toBeGreaterThan(0);
      expect(urls.filter((u) => /[?&]all=true/.test(u)), urls.join("\n")).toEqual([]);
      await close();
    });
  }
});
