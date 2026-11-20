import { expect, visit, test } from "../lib/fixtures";
import { open } from "../lib/suite";

// PLAN Tahap 6: list state lives in the URL, headers sort (aria-sort, keyboard), money is right aligned, an empty
// result offers "Reset filter", and no list loads every row (all=true) just to filter or sort in the browser.

// MySQL's collation ignores case and punctuation; compare the same way.
const compare = (x: string, y: string) => x.localeCompare(y, "id", { sensitivity: "base", ignorePunctuation: true });

test.describe("Vendor list", () => {
  test("sorting is a header button, shows in aria-sort and the URL, and survives a reload", async ({ browser }, info) => {
    const { page, close } = await open(browser, info, "owner");
    await visit(page, "/vendor");
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
    await visit(page, "/vendor");
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
    await visit(page, "/vendor");
    await page.getByRole("searchbox").or(page.getByPlaceholder(/Cari nama vendor/)).first().fill("zzzz-tidak-ada");
    await expect(page.getByText("Tidak ada hasil")).toBeVisible();
    await expect(page).toHaveURL(/q=zzzz/);
    await page.getByRole("button", { name: "Reset filter" }).click();
    await expect(page.getByText("Tidak ada hasil")).toBeHidden();
    await expect(page).not.toHaveURL(/q=/);
    await expect(page.getByRole("row").nth(1)).toBeVisible();
    await close();
  });

  test("the export carries the filters the list is showing", async ({ browser }, info) => {
    const { page, close } = await open(browser, info, "owner");
    await visit(page, "/vendor?isActive=false&q=Uji");
    const exported = page.waitForRequest((r) => r.url().includes("/vendors/export"));
    await page.getByRole("button", { name: "Ekspor ke Excel" }).first().click();
    const url = new URL((await exported).url());
    expect(url.searchParams.get("isActive")).toBe("false");
    expect(url.searchParams.get("q")).toBe("Uji");
    await close();
  });

  test("money is right aligned in tabular figures", async ({ browser }, info) => {
    const { page, close } = await open(browser, info, "owner");
    await visit(page, "/vendor");
    const cell = page.getByRole("row").nth(1).getByRole("cell").filter({ hasText: /^Rp/ }).first();
    await expect(cell).toBeVisible();
    const style = await cell.evaluate((el) => ({ align: getComputedStyle(el).textAlign, nums: getComputedStyle(el).fontVariantNumeric }));
    expect(style.align).toBe("right");
    expect(style.nums).toContain("tabular-nums");
    await close();
  });

  test("a vendor a project uses is not offered for deletion: the dialog says where and offers to deactivate (QA FQ-02)", async ({ browser }, info) => {
    const { page, close } = await open(browser, info, "owner");
    await visit(page, "/vendor");
    await page.getByRole("button", { name: "Aksi lain untuk Katering Rasa Nusantara" }).click();
    await page.getByRole("menuitem", { name: "Hapus permanen" }).click();
    const dialog = page.getByRole("dialog", { name: /Vendor masih dipakai|Hapus vendor/ });
    await expect(dialog).toContainText("Vendor ini masih dipakai di 1 proyek: Citra & Bima");
    await expect(dialog.getByRole("button", { name: "Nonaktifkan vendor" })).toBeVisible();
    await expect(dialog.getByRole("button", { name: "Hapus vendor permanen" })).toHaveCount(0);
    await dialog.getByRole("button", { name: "Batal" }).click();
    await expect(dialog).toBeHidden();

    // Used by a project after the check: the server's 409 reason shows in the dialog, which turns into the same offer.
    await page.route(/\/api\/v1\/vendors\/\d+$/, (route) =>
      route.request().method() === "DELETE"
        ? route.fulfill({
            status: 409,
            contentType: "application/json",
            body: JSON.stringify({
              success: false,
              message: "Vendor ini masih dipakai di 1 proyek: Sari & Doni. Nonaktifkan saja bila tidak dipakai lagi.",
              errors: { code: "vendor_in_use" },
              data: { affectedProjects: [{ id: "3", name: "Sari & Doni" }], hiddenProjectCount: 0 },
            }),
          })
        : route.fallback(),
    );
    await page.getByRole("button", { name: "Aksi lain untuk Foto Cahaya Studio" }).click();
    await page.getByRole("menuitem", { name: "Hapus permanen" }).click();
    await dialog.getByRole("button", { name: "Hapus vendor permanen" }).click();
    await expect(dialog.getByText("Vendor ini masih dipakai di 1 proyek: Sari & Doni. Nonaktifkan saja bila tidak dipakai lagi.")).toBeVisible();
    await expect(dialog.getByRole("button", { name: "Nonaktifkan vendor" })).toBeVisible();
    await expect(dialog.getByRole("button", { name: "Hapus vendor permanen" })).toHaveCount(0);
    await close();
  });
});

test.describe("Kartu: Proyek and Vendor can be read as a grid of cards", () => {
  test("Proyek: the switch shows cards, the sort menu sorts on the server, and the choice is remembered", async ({ browser }, info) => {
    const { page, close } = await open(browser, info, "owner");
    await visit(page, "/proyek");
    const group = page.getByRole("group", { name: "Bentuk daftar" });
    await expect(group.getByRole("button", { name: "Tabel" })).toHaveAttribute("aria-pressed", "true");
    await group.getByRole("button", { name: "Kartu" }).click();
    await expect(group.getByRole("button", { name: "Kartu" })).toHaveAttribute("aria-pressed", "true");

    const cards = page.getByRole("list", { name: "Daftar proyek" }).getByRole("listitem");
    await expect(cards.first()).toBeVisible();
    await expect(page.getByRole("table")).toHaveCount(0);
    // A card leads with the project, linked to it, and says what is still owed.
    await expect(cards.first().getByRole("link").first()).toHaveAttribute("href", /\/proyek\//);
    await expect(cards.first().getByText("Sisa tagihan")).toBeVisible();

    // Cards have no headers: the sort is a menu, and it is the same server-side sort the table uses.
    const answer = page.waitForResponse((r) => r.request().method() === "GET" && /\/projects\?/.test(r.url()) && /sort=name(:|%3A)desc/.test(r.url()));
    await page.getByRole("button", { name: /^Urutkan:/ }).click();
    await page.getByRole("option", { name: "Proyek (turun)" }).click();
    const names = ((await (await answer).json()).data as Array<{ name: string }>).map((p) => p.name.toLowerCase());
    expect(names).toEqual([...names].sort((x, y) => compare(y, x)));
    await expect(page).toHaveURL(/sort=name%3Adesc/);

    // Remembered in this browser for this list only.
    await page.reload();
    await expect(page.getByRole("list", { name: "Daftar proyek" })).toBeVisible();
    await visit(page, "/klien");
    await expect(page.getByRole("table")).toBeVisible();
    await expect(page.getByRole("group", { name: "Bentuk daftar" })).toHaveCount(0);
    await close();
  });

  test("Vendor: a card shows all three price packages and can be selected for a bulk action", async ({ browser }, info) => {
    const { page, close } = await open(browser, info, "owner");
    await visit(page, "/vendor");
    await page.getByRole("group", { name: "Bentuk daftar" }).getByRole("button", { name: "Kartu" }).click();
    const card = page.getByRole("list", { name: "Daftar vendor" }).getByRole("listitem").first();
    for (const label of ["Harga Akad", "Harga Akad + Resepsi", "Harga Resepsi saja"]) await expect(card.getByText(label, { exact: true })).toBeVisible();
    await card.getByRole("checkbox").check();
    await expect(page.getByRole("region", { name: "Aksi untuk baris terpilih" })).toContainText("1 dipilih");
    await page.getByRole("button", { name: "Batal pilih" }).click();
    // Back to the table.
    await page.getByRole("group", { name: "Bentuk daftar" }).getByRole("button", { name: "Tabel" }).click();
    await expect(page.getByRole("table", { name: "Daftar vendor" })).toBeVisible();
    await close();
  });
});

test.describe("no list loads every row to filter in the browser", () => {
  for (const [name, route, endpoint] of [
    ["Proyek", "/proyek", /\/api\/v1\/projects(\?|$)/],
    ["Klien", "/klien", /\/api\/v1\/clients(\?|$)/],
    ["Vendor", "/vendor", /\/api\/v1\/vendors(\?|$)/],
    ["Venue", "/venue", /\/api\/v1\/venues(\?|$)/],
    ["Tim", "/tim", /\/api\/v1\/staff(\?|$)/],
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
