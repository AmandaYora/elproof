import type { Browser, TestInfo } from "@playwright/test";
import { expect, prepareContext, test, visit } from "../lib/fixtures";
import { open } from "../lib/suite";
import { apiURL, webURL, type Session } from "../lib/config";
import { call, must } from "../lib/api";
import { browserSession, sessionOf, storageStateFor } from "../lib/session";
import { resepsiOn } from "../lib/project";

// PLAN paket WO Tahap 3 "Rincian & nilai kontrak", by the Owner of WO Berbayar: a package from the catalog becomes the
// contract value, more guests at the Resepsi add a "Tambahan tamu" line, the margin says when a vendor's cost is
// missing, a schedule that no longer adds up is fitted only on request, and the couple's portal shows the rincian
// (numbers only for a role with portal.finance.view, never a price on an isi paket).
// This test WRITES data, so it runs in the "mutating" project (after every screenshot project).

const API = (path: string) => `${apiURL()}/api/v1${path}`;
const COMPONENTS = ["Katering buffet 300 porsi", "Dekorasi pelaminan", "Dokumentasi foto dan video", "MC dan hiburan"];

interface RincianReply {
  totals?: { subtotal: number; tax: number; total: number };
  termsMismatch?: boolean;
  margin?: { missingCost: number; incomplete: boolean };
  lines: Array<{ kind: string; amount?: number; unitPrice?: number; description: string }>;
}

/** A client of the project with the workspace's portal role `roleName`, signed in (password set through the access link). */
async function clientSession(token: string, projectId: number, roleName: "Pengantin" | "Keluarga", tag: string): Promise<Session> {
  const roles = must("client roles", await call(apiURL(), token, "GET", "/api/v1/access/roles?kind=client")).json.data as { id: number; name: string }[];
  const role = roles.find((r) => r.name === roleName)!;
  const bride = roleName === "Pengantin";
  const client = must(`client ${roleName}`, await call(apiURL(), token, "POST", "/api/v1/clients", {
    projectId, role: bride ? "Bride" : "Family Representative", relationNote: bride ? "" : "Ibu mempelai wanita",
    name: bride ? "Nadia Putri" : "Ibu Ratna", phone: bride ? "081298763001" : "081298763002",
    username: `${bride ? "nadia" : "ratna"}.${tag}`, email: "", accessRoleId: role.id,
  }));
  const link = client.json.data.accessLink.token as string;
  return sessionOf(must(`${roleName} password`, await call(apiURL(), null, "POST", `/api/v1/auth/links/${link}/password`, { password: "Kata-sandi-klien-123" })));
}

/** A browser context signed in as `session` (a client of the project). */
async function openAs(browser: Browser, info: TestInfo, session: Session) {
  const context = await browser.newContext({ ...info.project.use, storageState: storageStateFor(webURL(), session) });
  const page = await context.newPage();
  await prepareContext(page);
  return { page, close: () => context.close() };
}

test("rincian: paket dari katalog, tambahan tamu, margin, hitung ulang termin, dan Layanan di portal", async ({ browser }, info) => {
  test.setTimeout(150_000);
  const { page, close } = await open(browser, info, "paying");
  await visit(page, "/beranda");
  const token = (await browserSession(page)).accessToken;
  const auth = { Authorization: `Bearer ${token}` };
  const tag = Date.now().toString(36).slice(-5);

  // The catalog package (Resepsi, 300 guests included, Rp5.000.000 per 50 more) and a project with 300 guests at its
  // Resepsi, without a rincian yet.
  const kinds = must("event kinds", await call(apiURL(), token, "GET", "/api/v1/event-kinds")).json.data as { id: number; name: string }[];
  const packageName = `Paket Rincian Uji ${tag}`;
  must("package", await call(apiURL(), token, "POST", "/api/v1/packages", {
    name: packageName, price: 100_000_000, eventKindIds: [kinds.find((k) => k.name === "Resepsi")!.id],
    includedGuests: 300, extraGuestStep: 50, extraGuestStepPrice: 5_000_000, defaultVenueId: null,
    components: COMPONENTS.map((description) => ({ description, categoryId: null, vendorId: null, estimatedCost: 10_000_000 })),
  }));
  const events = await resepsiOn(token, "2027-08-14");
  events[0].sessions[0].guests = 300;
  const made = must("project", await call(apiURL(), token, "POST", "/api/v1/projects", {
    name: `Nadia & Raka ${tag}`, brideName: "Nadia Putri", groomName: "Raka Saputra", events, prepStartDate: "2026-11-09",
    status: "Preparation", picStaffId: Number((await browserSession(page)).principalId), picSalesStaffId: 0, description: "",
  }));
  const projectId = made.json.data.id as number;
  const rincianNow = async () => (await (await page.request.get(API(`/projects/${projectId}/lines`), { headers: auth })).json()).data as RincianReply;

  // 1. Acara & rincian: an empty rincian invites a package; the package comes from the catalog with its isi.
  await visit(page, `/proyek/${projectId}/acara-rincian`);
  const rincian = page.getByRole("region", { name: "Rincian", exact: true });
  const summary = page.getByRole("region", { name: "Ringkasan biaya" });
  await expect(rincian.getByText("Belum ada rincian")).toBeVisible();
  await rincian.getByRole("button", { name: "Tambah paket" }).click();
  const picker = page.getByRole("dialog", { name: "Tambah paket" });
  await picker.getByPlaceholder("Cari nama paket").fill(tag);
  await picker.getByRole("option", { name: new RegExp(packageName) }).getByRole("button").click();
  await picker.getByRole("button", { name: "Tambah paket" }).click();
  await expect(picker).toBeHidden();
  await expect(rincian.getByText("Baru dari katalog")).toBeVisible();
  // The running total is the server's preview, never a sum made in the browser.
  await expect(page.getByTestId("nilai-kontrak")).toHaveText("Rp100.000.000", { timeout: 10_000 });
  const saved = page.waitForResponse((r) => r.url().endsWith(`/projects/${projectId}/lines`) && r.request().method() === "PUT");
  await page.getByRole("button", { name: "Simpan rincian" }).click();
  const put = await saved;
  expect(put.status()).toBe(200);
  expect(put.request().headers()["if-match"], "the save sends the rincian's version").toBeTruthy();
  await expect(page.getByText("Rincian disimpan").first()).toBeVisible();
  await expect(rincian.getByText("Baru dari katalog")).toHaveCount(0);
  await rincian.getByRole("button", { name: new RegExp(`^Tampilkan 4 isi ${packageName}`) }).click();
  const isiInputs = rincian.getByRole("textbox", { name: "Uraian isi paket" });
  await expect(isiInputs).toHaveCount(COMPONENTS.length);
  expect(await isiInputs.evaluateAll((els) => els.map((e) => (e as HTMLInputElement).value)), "the isi are copied from the catalog").toEqual(COMPONENTS);
  await expect(rincian.getByText("Termasuk paket").first()).toBeVisible();
  let api = await rincianNow();
  expect(api.totals?.total).toBe(100_000_000);
  expect(api.lines.filter((l) => l.kind === "Component")).toHaveLength(COMPONENTS.length);

  // 2. Margin (finance.margin.view): complete at first; a vendor the WO pays without a cost makes it incomplete.
  await expect(summary.getByText("Biaya rekanan")).toBeVisible();
  await expect(summary.getByText("Margin", { exact: true })).toBeVisible();
  await expect(summary.getByText(/Margin belum lengkap/)).toHaveCount(0);
  const cats = must("categories", await call(apiURL(), token, "GET", "/api/v1/vendor-categories?all=true")).json.data as { id: number }[];
  const vendor = must("vendor", await call(apiURL(), token, "POST", "/api/v1/vendors", {
    name: `Katering Rincian Uji ${tag}`, categoryId: cats[0].id, picName: "Bu Uji", phone: "081211110199", email: "", socialMedia: "", city: "Kota Bandung", address: "", notes: "",
  })).json.data.id as number;
  must("supplier without cost", await call(apiURL(), token, "POST", `/api/v1/projects/${projectId}/suppliers`, {
    vendorId: vendor, paidBy: "WO", showName: true, agreedCost: null, engagementStatus: "Negotiation",
    bookingDate: "", dueDate: "2027-06-01", picStaffId: Number((await browserSession(page)).principalId), scope: "Katering", notes: "",
  }));
  // A schedule that adds up to today's contract value.
  must("payment terms", await call(apiURL(), token, "PUT", `/api/v1/projects/${projectId}/payment-terms`, {
    terms: [
      { label: "DP", amount: 30_000_000, dueDate: "2026-12-15" },
      { label: "Pelunasan", amount: 70_000_000, dueDate: "2027-07-14" },
    ],
  }));
  await page.reload();
  const missing = summary.getByRole("link", { name: "Margin belum lengkap (1 vendor)" });
  await expect(missing).toBeVisible();
  await expect(missing).toHaveAttribute("href", /\/vendor\?saring=tanpa-biaya$/);
  api = await rincianNow();
  expect(api.margin).toMatchObject({ missingCost: 1, incomplete: true });
  expect(api.termsMismatch).toBe(false);
  await expect(summary.getByRole("button", { name: "Hitung ulang termin" })).toHaveCount(0);

  // 3. 400 guests at the Resepsi: 100 over the 300 included = 2 steps of 50 at Rp5.000.000, a line nobody types.
  await page.getByRole("button", { name: "Ubah acara Resepsi" }).click();
  const drawer = page.getByRole("dialog", { name: /Ubah acara: Resepsi/ });
  await drawer.getByLabel("Tamu sesi 1").fill("400");
  await drawer.getByRole("button", { name: "Simpan acara" }).click();
  await expect(drawer).toBeHidden();
  const extra = rincian.getByRole("row").filter({ hasText: "Tambahan tamu" });
  await expect(extra).toContainText("Rp10.000.000");
  await expect(extra).toContainText("400 tamu − 300 termasuk = 100 → 2 × 50 tamu × Rp5.000.000");
  await expect(extra.getByRole("textbox")).toHaveCount(0); // computed, never typed
  await expect(page.getByTestId("nilai-kontrak")).toHaveText("Rp110.000.000");
  api = await rincianNow();
  expect(api.totals?.total).toBe(110_000_000);
  expect(api.termsMismatch).toBe(true);

  // 4. The schedule (Rp100.000.000) no longer matches: said in Ringkasan biaya, fitted only when asked.
  await expect(summary.getByText("Jadwal termin (Rp100.000.000) tidak sama dengan nilai kontrak (Rp110.000.000).")).toBeVisible();
  await summary.getByRole("button", { name: "Hitung ulang termin" }).click();
  const confirm = page.getByRole("alertdialog", { name: "Hitung ulang jadwal termin?" });
  const recalculated = page.waitForResponse((r) => r.url().endsWith(`/projects/${projectId}/payment-terms/recalculate`));
  await confirm.getByRole("button", { name: "Hitung ulang termin" }).click();
  expect((await recalculated).status()).toBe(200);
  await expect(page.getByText("Jadwal termin dihitung ulang").first()).toBeVisible();
  await expect(summary.getByText(/tidak sama dengan nilai kontrak/)).toHaveCount(0);
  const terms = (await (await page.request.get(API(`/projects/${projectId}/payment-terms`), { headers: auth })).json()).data as Array<{ amount: number }>;
  expect(terms.reduce((sum, t) => sum + t.amount, 0)).toBe(110_000_000);
  expect((await rincianNow()).termsMismatch).toBe(false);

  // 5. Portal › Layanan, Keluarga (no portal.finance.view): the packages and their isi, not one number.
  const famSession = await clientSession(token, projectId, "Keluarga", tag);
  const famLines = must("lines as Keluarga", await call(apiURL(), famSession.accessToken, "GET", `/api/v1/projects/${projectId}/lines`)).json.data as RincianReply;
  expect(famLines.totals, "no totals without portal.finance.view").toBeUndefined();
  expect(famLines.lines.every((l) => l.amount === undefined && l.unitPrice === undefined)).toBe(true);
  const keluarga = await openAs(browser, info, famSession);
  await visit(keluarga.page, "/portal/layanan");
  const famSection = keluarga.page.getByRole("region", { name: "Paket & rincian" });
  await expect(famSection.getByText(packageName)).toBeVisible();
  await expect(famSection.getByText(COMPONENTS[0])).toBeVisible();
  await expect(famSection).not.toContainText("Rp");
  await expect(famSection).not.toContainText("Nilai kontrak");
  // "Paket & rincian" comes before the vendor progress.
  const famHeadings = await keluarga.page.getByRole("heading", { level: 2 }).allTextContents();
  expect(famHeadings.indexOf("Paket & rincian")).toBeLessThan(famHeadings.indexOf("Progres vendor"));
  await keluarga.close();

  // 6. Pengantin (portal.finance.view): the contract value and line amounts, but an isi paket never has a price.
  const pengantin = await openAs(browser, info, await clientSession(token, projectId, "Pengantin", tag));
  await visit(pengantin.page, "/portal/layanan");
  const section = pengantin.page.getByRole("region", { name: "Paket & rincian" });
  await expect(section.getByText("Rp110.000.000").first()).toBeVisible();
  const isi = section.locator("ul ul > li");
  await expect(isi).toHaveCount(3); // the first three, then "Lihat 1 isi lainnya"
  await section.getByRole("button", { name: "Lihat 1 isi lainnya" }).click();
  await expect(isi).toHaveCount(COMPONENTS.length);
  for (const [i, c] of COMPONENTS.entries()) {
    await expect(isi.nth(i)).toHaveText(c);
    await expect(isi.nth(i)).not.toContainText("Rp");
  }
  await expect(section.getByText("100 tamu di atas 300 yang termasuk paket")).toBeVisible();
  await pengantin.close();

  await close();
});

