import { expect, visit, test } from "../lib/fixtures";
import { open } from "../lib/suite";
import { apiURL } from "../lib/config";

// PLAN Tahap 6: bulk activate/deactivate. These tests WRITE data, so they run in the "mutating" project.

const API = (path: string) => `${apiURL()}/api/v1${path}`;

test("selecting vendors and pressing Nonaktifkan changes them all in one call", async ({ browser }, info) => {
  const { page, close } = await open(browser, info, "paying");
  await visit(page, "/dashboard");
  const session = await page.evaluate(() => JSON.parse(localStorage.getItem("elproof.session") ?? "{}") as { accessToken: string });
  const auth = { Authorization: `Bearer ${session.accessToken}` };
  const cats = (await (await page.request.get(API("/vendor-categories?all=true"), { headers: auth })).json()).data as Array<{ id: number }>;
  for (const name of ["Uji Massal A", "Uji Massal B"]) {
    const made = await page.request.post(API("/vendors"), {
      headers: auth,
      data: {
        name, categoryId: cats[0].id, picName: "Pak Uji", phone: "081200001111", email: "", socialMedia: "", city: "Kota Bandung", address: "Jl. Uji",
        priceAkad: 1_000_000, priceAkadResepsi: 2_000_000, priceResepsi: 0, notes: "",
      },
    });
    expect(made.status()).toBe(201);
  }

  await visit(page, "/vendors?q=Uji%20Massal");
  await expect(page.getByRole("row").filter({ hasText: "Uji Massal" })).toHaveCount(2);
  await page.getByRole("checkbox", { name: "Pilih semua di halaman ini" }).check();
  await expect(page.getByText("2 dipilih")).toBeVisible();

  const calls: string[] = [];
  page.on("request", (r) => r.method() === "POST" && calls.push(new URL(r.url()).pathname));
  await page.getByRole("button", { name: "Nonaktifkan", exact: true }).first().click();
  await expect(page.getByText("2 vendor dinonaktifkan")).toBeVisible();
  expect(calls.filter((c) => c.endsWith("/vendors/bulk")), "one bulk call").toHaveLength(1);
  await expect(page.getByRole("row").filter({ hasText: "Uji Massal" }).filter({ hasText: "Nonaktif" })).toHaveCount(2);
  await expect(page.getByText("2 dipilih")).toBeHidden(); // the selection is cleared
  await close();
});
