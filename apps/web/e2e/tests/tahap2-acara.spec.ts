import type { Locator, Page } from "@playwright/test";
import { expect, visit, test } from "../lib/fixtures";
import { open } from "../lib/suite";
import { apiURL } from "../lib/config";
import { call, must } from "../lib/api";
import { browserSession } from "../lib/session";
import { choose } from "../lib/project";

// PLAN paket WO Tahap 2: "buat proyek → acara → vendor & venue → bayar rekanan", by the Owner of WO Berbayar.
// This test WRITES data, so it runs in the "mutating" project (after every screenshot project).

const API = (path: string) => `${apiURL()}/api/v1${path}`;

/** The directory this flow needs, made through the API: three venues and one vendor. */
async function directory(token: string, tag: string) {
  const venue = async (name: string, standing: number | null, seated: number | null) =>
    must(`venue ${name}`, await call(apiURL(), token, "POST", "/api/v1/venues", {
      name, picName: "Pak Uji", phonePic: "081300000099", phoneVenue: "", email: "", address: `Jl. ${name} No. 1`, city: "Kota Bandung",
      capacityStanding: standing, capacitySeated: seated, charge: null, facilities: "", socialMedia: "", notes: "",
    }));
  await venue(`Masjid Al-Ikhlas Uji ${tag}`, null, 150);
  await venue(`Gedung Melati Uji ${tag}`, 600, null);
  await venue(`Gedung Kenanga Uji ${tag}`, 500, null);
  const cats = must("categories", await call(apiURL(), token, "GET", "/api/v1/vendor-categories?all=true")).json.data as { id: number }[];
  must("vendor", await call(apiURL(), token, "POST", "/api/v1/vendors", {
    name: `Foto Lensa Abadi Uji ${tag}`, categoryId: cats[0].id, picName: "Kak Uji", phone: "081211110099", email: "", socialMedia: "", city: "Kota Bandung", address: "", notes: "",
  }));
}

/** Fills one event of the create page or the event drawer. */
async function fillEvent(scope: Locator, page: Page, e: { kind: string; date: string; venue: string; layout: "Berdiri" | "Duduk"; guests: number[] }) {
  await choose(scope.getByRole("button", { name: /^Jenis acara/ }), e.kind);
  await scope.getByRole("textbox", { name: /^Tanggal \(wajib diisi\)/ }).fill(e.date);
  await scope.getByRole("button", { name: /^(Pilih|Ganti) venue$/ }).click();
  await scope.getByRole("button", { name: new RegExp(e.venue) }).click();
  await scope.getByRole("radio", { name: new RegExp(e.layout) }).check();
  for (const [i, guests] of e.guests.entries()) {
    if (i > 0 && (await scope.getByLabel(`Tamu sesi ${i + 1}`).count()) === 0) await scope.getByRole("button", { name: "Tambah sesi" }).click();
    await scope.getByLabel(`Tamu sesi ${i + 1}`).fill(String(guests));
  }
  void page;
}

test("buat proyek → acara → vendor & venue → bayar rekanan", async ({ browser }, info) => {
  test.setTimeout(120_000);
  const { page, close } = await open(browser, info, "paying");
  await visit(page, "/beranda");
  const session = await browserSession(page);
  // A tag per run: the stack can be kept up between runs (E2E_REUSE=1), and the directory would otherwise hold two of each.
  const tag = Date.now().toString(36).slice(-5);
  await directory(session.accessToken, tag);

  // 1. A new project without an offer: the couple, then its events (no payment-term step).
  await visit(page, "/proyek/baru");
  await page.getByLabel(/Nama mempelai wanita/).fill("Annisa Rahma");
  await page.getByLabel(/Nama mempelai pria/).fill("Fajar Hidayat");
  // No Paket or Nilai kontrak here any more: both come from the rincian (Tahap 3).
  await expect(page.getByLabel(/Nilai kontrak/)).toHaveCount(0);
  await page.getByRole("button", { name: "Lanjut", exact: true }).click();
  await expect(page.getByText(/Langkah 2 dari 2/)).toBeVisible();

  const events = page.locator("ol").first().locator(":scope > li"); // the events (each holds its own list of sessions)
  await fillEvent(events.nth(0), page, { kind: "Akad", date: "2027-04-10", venue: `Masjid Al-Ikhlas Uji ${tag}`, layout: "Duduk", guests: [120] });
  await page.getByRole("button", { name: "Tambah acara" }).click();
  await fillEvent(events.nth(1), page, { kind: "Resepsi", date: "2027-04-11", venue: `Gedung Melati Uji ${tag}`, layout: "Berdiri", guests: [550, 550] });
  await page.getByRole("button", { name: "Buat proyek" }).click();
  await expect(page).toHaveURL(/\/proyek\/\d+\/acara-rincian$/, { timeout: 20_000 });
  const projectId = Number(/\/proyek\/(\d+)\//.exec(page.url())![1]);

  // 2. Acara: both events are saved; 650 in a 600 hall is refused with the numbers, and so is a 500 hall.
  await expect(page.getByRole("heading", { name: /^Akad/ })).toBeVisible();
  await expect(page.getByRole("heading", { name: /^Resepsi/ })).toBeVisible();
  await page.getByRole("button", { name: "Ubah acara Resepsi" }).click();
  const drawer = page.getByRole("dialog", { name: /Ubah acara: Resepsi/ });
  await drawer.getByLabel("Tamu sesi 2").fill("650");
  await expect(drawer.getByText(/50 di atas kapasitas/)).toBeVisible();
  await drawer.getByRole("button", { name: "Simpan acara" }).click();
  await expect(drawer.getByRole("alert").filter({ hasText: /600/ }).first()).toContainText(/650/);
  await expect(drawer.getByRole("button", { name: "Bagi jadi 2 sesi" })).toBeVisible();

  await drawer.getByLabel("Tamu sesi 2").fill("550");
  await drawer.getByRole("button", { name: "Ganti venue" }).first().click();
  await drawer.getByRole("button", { name: new RegExp(`Gedung Kenanga Uji ${tag}`) }).click();
  await drawer.getByRole("button", { name: "Simpan acara" }).click();
  await expect(drawer.getByRole("alert").filter({ hasText: new RegExp(`Gedung Kenanga Uji ${tag}`) }).first()).toContainText(/500/);
  await drawer.getByRole("button", { name: "Batal" }).click();
  await expect(drawer).toBeHidden();

  // 3. Vendor & venue: the venues chosen for the events are there already; a WO-paid vendor is added.
  await page.getByRole("navigation", { name: "Bagian proyek" }).getByRole("link", { name: "Vendor & venue" }).click();
  await expect(page.getByRole("cell", { name: new RegExp(`Masjid Al-Ikhlas Uji ${tag}`) }).first()).toBeVisible();
  await expect(page.getByRole("cell", { name: new RegExp(`Gedung Melati Uji ${tag}`) }).first()).toBeVisible();
  await page.getByRole("button", { name: "Tambah vendor" }).first().click();
  const add = page.getByRole("dialog", { name: "Tambah vendor" });
  await choose(add.getByRole("button", { name: /^Vendor:/ }), `Foto Lensa Abadi Uji ${tag}`);
  await add.getByRole("radio", { name: "WO" }).check();
  await add.getByLabel(/Biaya rekanan/).fill("10000000");
  await add.getByRole("button", { name: "Tambah vendor" }).click();
  await expect(add).toBeHidden();
  await expect(page.getByRole("cell", { name: new RegExp(`Foto Lensa Abadi Uji ${tag}`) }).first()).toBeVisible();

  // 4. Keuangan › Vendor & venue dibayar WO: a payment to the photographer.
  await visit(page, `/proyek/${projectId}/keuangan/vendor-venue`);
  await page.getByRole("button", { name: "Catat pembayaran" }).click();
  const pay = page.getByRole("dialog", { name: "Catat pembayaran ke rekanan" });
  await choose(pay.getByRole("button", { name: /^Vendor atau venue:/ }), new RegExp(`Foto Lensa Abadi Uji ${tag}`));
  await pay.getByLabel(/Nominal/).fill("3000000");
  await pay.getByRole("button", { name: "Simpan pembayaran" }).click();
  await expect(pay).toBeHidden();
  await expect(page.getByText("Rp3.000.000").first()).toBeVisible();

  // 5. Once paid, who pays it is locked (the server refuses, and the drawer says so).
  const auth = { Authorization: `Bearer ${session.accessToken}` };
  const suppliers = (await (await page.request.get(API(`/projects/${projectId}/suppliers`), { headers: auth })).json()).data as Array<{ id: number; name: string; paidByLocked: string; updatedAt: string }>;
  const photo = suppliers.find((s) => s.name === `Foto Lensa Abadi Uji ${tag}`)!;
  expect(photo.paidByLocked).not.toBe("");
  const locked = await page.request.patch(API(`/projects/${projectId}/suppliers/${photo.id}`), {
    headers: { ...auth, "If-Match": `"${photo.updatedAt}"` },
    data: { paidBy: "Client" },
  });
  expect(locked.status()).toBe(422);
  await close();
});
