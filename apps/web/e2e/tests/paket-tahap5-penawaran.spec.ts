import type { Page } from "@playwright/test";
import { expect, test, visit } from "../lib/fixtures";
import { open } from "../lib/suite";
import { apiURL, config } from "../lib/config";
import { call, must } from "../lib/api";
import { browserSession } from "../lib/session";
import { resepsiOn } from "../lib/project";
import { sql } from "../lib/sql";

// PLAN paket WO Tahap 5 "Penawaran", by the Owner of WO Berbayar: a quotation is made from the top bar's list page,
// filled with an event, a vendor, a rincian and a payment schedule, previewed as a DRAF PDF (no revision), shared as
// Revisi 1, changed and shared again as Revisi 2, marked accepted and turned into a project ("Jadikan proyek") that has
// the same events, Vendor & venue, rincian and termin, plus the link to its Kesepakatan awal. A second test: a quotation
// whose validity has passed is refused "diterima" until it is extended.
// This test WRITES data, so it runs in the "mutating" project (after every screenshot project).

const NUMBER = /^[A-Z0-9]+-PNW-\d{6}-\d{3}$/;
const EVENT_DATE = "2027-10-16";
const PRICE_1 = 60_000_000;
const PRICE_2 = 65_000_000;

interface Detail {
  quotation: { id: number; number: string; status: string; displayStatus: string; linesUpdatedAt: string; termsUpdatedAt: string; acceptedRevisionN: number | null; convertedProjectId: number | null };
  totals: { total: number } | null;
}

/** The Owner's token and staff id, read from the signed-in page. */
async function owner(page: Page) {
  const session = await browserSession(page);
  return { token: session.accessToken, staffId: Number(session.principalId) };
}

const detailOf = async (token: string, id: number) => must("quotation", await call(apiURL(), token, "GET", `/api/v1/quotations/${id}`)).json.data as Detail;
const revisionsOf = async (token: string, id: number) =>
  must("revisions", await call(apiURL(), token, "GET", `/api/v1/quotations/${id}/revisions`)).json.data as Array<{ n: number; contractTotal: number }>;

/** The rincian of the working copy: one item at `price` (a new workspace is not PKP, so the contract is exactly that). */
async function saveLines(token: string, id: number, price: number) {
  const d = await detailOf(token, id);
  must("lines", await call(apiURL(), token, "PUT", `/api/v1/quotations/${id}/lines`, {
    lines: [{ kind: "Item", description: "Paket Pernikahan Uji", quantity: 1, unit: "paket", unitPrice: price }],
  }, { "If-Match": `"${d.quotation.linesUpdatedAt}"` }));
}

/** A schedule of 30% when accepted and 70% fourteen days before the main event. */
async function saveTerms(token: string, id: number) {
  const d = await detailOf(token, id);
  must("terms", await call(apiURL(), token, "PUT", `/api/v1/quotations/${id}/terms`, {
    terms: [
      { label: "DP", percent: 30, amount: null, dueBasis: "Accepted", dueOffsetDays: 0, dueEventId: null },
      { label: "Pelunasan", percent: 70, amount: null, dueBasis: "Event", dueOffsetDays: -14, dueEventId: null },
    ],
  }, { "If-Match": `"${d.quotation.termsUpdatedAt}"` }));
}

test("penawaran: revisi 1 dan 2, diterima, lalu jadikan proyek dengan isi yang sama dan kesepakatan awal", async ({ browser }, info) => {
  test.setTimeout(180_000);
  const { page, close } = await open(browser, info, "paying");
  await visit(page, "/beranda");
  const { token } = await owner(page);
  const tag = Date.now().toString(36).slice(-5);
  const couple = `Laras & Dimas ${tag}`;

  // 1. "Penawaran baru" (the list opens it with ?baru=1, as the top bar's "Baru" menu does).
  await visit(page, "/penawaran?baru=1");
  const create = page.getByRole("dialog", { name: "Penawaran baru" });
  await expect(create).toBeVisible();
  await create.getByLabel("Nama pasangan").fill(couple);
  await create.getByRole("button", { name: "Buat penawaran" }).click();
  await expect(page).toHaveURL(/\/penawaran\/\d+$/);
  const quotationId = Number(/\/penawaran\/(\d+)$/.exec(page.url())![1]);
  const number = (await detailOf(token, quotationId)).quotation.number;
  expect(number, "numbered KODE-PNW-YYMMDD-NNN").toMatch(NUMBER);
  await expect(page.getByRole("heading", { level: 1, name: couple })).toBeVisible();
  await expect(page.getByText(number, { exact: true }).first()).toBeVisible();

  // 2. Its content (through the API, as the editors on this page send it): a Resepsi, a vendor, a rincian, a schedule.
  const event = (await resepsiOn(token, EVENT_DATE))[0];
  must("event", await call(apiURL(), token, "POST", `/api/v1/quotations/${quotationId}/events`, event));
  const cats = must("categories", await call(apiURL(), token, "GET", "/api/v1/vendor-categories?all=true")).json.data as { id: number; name: string }[];
  const vendorName = `Dekorasi Melati ${tag}`;
  const vendorId = must("vendor", await call(apiURL(), token, "POST", "/api/v1/vendors", {
    name: vendorName, categoryId: (cats.find((c) => /dekor/i.test(c.name)) ?? cats[0]).id, picName: "Bu Sari", phone: "081211110377",
    email: "", socialMedia: "", city: "Kota Bandung", address: "", notes: "",
  })).json.data.id as number;
  must("supplier", await call(apiURL(), token, "POST", `/api/v1/quotations/${quotationId}/suppliers`, {
    vendorId, paidBy: "WO", showName: true, agreedCost: 12_000_000, notes: "",
  }));
  await saveLines(token, quotationId, PRICE_1);
  await saveTerms(token, quotationId);
  await page.reload();
  await expect(page.getByRole("heading", { level: 1, name: couple })).toBeVisible();

  // 3. "Pratinjau PDF" shows the DRAF PDF and makes no revision.
  const previewed = page.waitForResponse((r) => r.url().includes(`/quotations/${quotationId}/preview.pdf`) && r.request().method() === "GET");
  await page.getByRole("button", { name: "Pratinjau PDF" }).click();
  expect((await previewed).status()).toBe(200);
  await expect(page.getByRole("dialog")).toBeVisible();
  await page.keyboard.press("Escape");
  await expect(page.getByRole("dialog")).toHaveCount(0);
  expect(await revisionsOf(token, quotationId), "a preview makes no revision").toEqual([]);

  // 4. "Buat revisi untuk klien": Revisi 1.
  const first = page.waitForResponse((r) => r.url().endsWith(`/quotations/${quotationId}/revisions`) && r.request().method() === "POST");
  await page.getByRole("button", { name: "Buat revisi untuk klien" }).click();
  expect((await first).status()).toBe(201);
  const riwayat = page.locator("#riwayat");
  await expect(riwayat.getByText("Revisi 1", { exact: true })).toBeVisible();
  expect((await revisionsOf(token, quotationId)).map((r) => r.n)).toEqual([1]);

  // 5. A change after Revisi 1 is flagged, and the next revision is Revisi 2.
  await saveLines(token, quotationId, PRICE_2);
  await page.reload();
  await expect(page.getByText(/Ada perubahan yang belum dikirim ke klien \(sejak Revisi 1\)/)).toBeVisible();
  const second = page.waitForResponse((r) => r.url().endsWith(`/quotations/${quotationId}/revisions`) && r.request().method() === "POST");
  await page.getByRole("button", { name: "Buat revisi untuk klien" }).click();
  expect((await second).status()).toBe(201);
  await expect(riwayat.getByText("Revisi 2", { exact: true })).toBeVisible();
  const revisions = await revisionsOf(token, quotationId);
  expect(revisions.map((r) => r.n).sort()).toEqual([1, 2]);
  expect(revisions.find((r) => r.n === 2)!.contractTotal).toBe(PRICE_2);

  // 6. "Tandai diterima": Revisi 2 (the one the working copy matches) is chosen.
  await page.getByRole("button", { name: "Tandai diterima" }).first().click();
  const accept = page.getByRole("dialog", { name: "Tandai penawaran diterima" });
  await expect(accept).toBeVisible();
  await expect(accept).toContainText("Revisi 2");
  const accepted = page.waitForResponse((r) => r.url().endsWith(`/quotations/${quotationId}/status`) && r.request().method() === "POST");
  await accept.getByRole("button", { name: "Tandai diterima" }).click();
  expect((await accepted).status()).toBe(200);
  await expect(accept).toBeHidden();
  const afterAccept = await detailOf(token, quotationId);
  expect(afterAccept.quotation.status).toBe("Accepted");
  expect(afterAccept.quotation.acceptedRevisionN).toBe(2);

  // 7. "Jadikan proyek": the dates come from the revision, the check shows the contract, "Buat proyek" makes it.
  await page.getByRole("button", { name: "Jadikan proyek" }).click();
  await expect(page).toHaveURL(new RegExp(`/penawaran/${quotationId}/jadikan-proyek$`));
  await expect(page.getByRole("heading", { level: 1, name: "Jadikan proyek" })).toBeVisible();
  await expect(page.getByRole("textbox", { name: /^Resepsi/ })).toHaveValue(EVENT_DATE);
  await expect(page.getByText("Rp65.000.000").first()).toBeVisible();
  const converted = page.waitForResponse((r) => r.url().endsWith(`/projects/from-quotation/${quotationId}`) && r.request().method() === "POST");
  await page.getByRole("button", { name: "Buat proyek" }).click();
  const reply = await converted;
  expect(reply.status()).toBe(201);
  expect(reply.request().headers()["idempotency-key"], "the conversion carries an Idempotency-Key").toBeTruthy();
  await expect(page).toHaveURL(/\/proyek\/\d+\//);
  const projectId = Number(/\/proyek\/(\d+)\//.exec(page.url())![1]);

  // 8. The project holds what the client accepted.
  const project = must("project", await call(apiURL(), token, "GET", `/api/v1/projects/${projectId}`)).json.data;
  expect(project.quotation).toEqual({ id: quotationId, number, revisionN: 2 });
  const events = must("events", await call(apiURL(), token, "GET", `/api/v1/projects/${projectId}/events`)).json.data as Array<{ eventKindId: number; eventDate: string }>;
  expect(events.map((e) => [e.eventKindId, e.eventDate.slice(0, 10)])).toEqual([[event.eventKindId, EVENT_DATE]]);
  const suppliers = must("suppliers", await call(apiURL(), token, "GET", `/api/v1/projects/${projectId}/suppliers`)).json.data as Array<{ name: string; paidBy: string }>;
  expect(suppliers.map((s) => [s.name, s.paidBy])).toEqual([[vendorName, "WO"]]);
  const lines = must("lines", await call(apiURL(), token, "GET", `/api/v1/projects/${projectId}/lines`)).json.data as { lines: Array<{ description: string }>; totals: { total: number } };
  expect(lines.lines.map((l) => l.description)).toEqual(["Paket Pernikahan Uji"]);
  expect(lines.totals.total).toBe(PRICE_2);
  const terms = must("terms", await call(apiURL(), token, "GET", `/api/v1/projects/${projectId}/payment-terms`)).json.data as Array<{ label: string; amount: number }>;
  expect(terms.map((t) => [t.label, t.amount])).toEqual([["DP", 19_500_000], ["Pelunasan", 45_500_000]]);

  // A second "Buat proyek" (another key, e.g. another tab) lands on the same project.
  const again = await call(apiURL(), token, "POST", `/api/v1/projects/from-quotation/${quotationId}`, {
    name: couple, brideName: "", groomName: "", events: [{ eventDate: EVENT_DATE }], prepStartDate: "2026-11-09",
    picStaffId: (await owner(page)).staffId, picSalesStaffId: null, description: "",
  }, { "Idempotency-Key": `e2e-again-${tag}` });
  expect(again.status).toBe(200);
  expect(again.json.data.created).toBe(false);
  expect(again.json.data.id).toBe(projectId);

  // 9. The project header names its quotation; "Lihat kesepakatan awal" opens the read-only Kesepakatan awal.
  await expect(page.getByText(`${number} Revisi 2`).first()).toBeVisible();
  await page.getByRole("link", { name: "Lihat kesepakatan awal" }).click();
  await expect(page).toHaveURL(new RegExp(`/proyek/${projectId}/acara-rincian\\?kesepakatan=1$`));
  const agreement = page.getByRole("dialog", { name: "Kesepakatan awal" });
  await expect(agreement).toBeVisible();
  await expect(agreement).toContainText(`${number} Revisi 2`);
  await expect(agreement).toContainText("Paket Pernikahan Uji");
  await expect(agreement).toContainText(vendorName);
  await expect(agreement).toContainText("Rp65.000.000");
  await agreement.getByRole("button", { name: "Bandingkan dengan rincian sekarang" }).click();
  await expect(agreement.getByText("Rincian sama dengan kesepakatan awal.")).toBeVisible();
  await expect(agreement.getByText("Acara sama dengan kesepakatan awal.")).toBeVisible();
  await agreement.getByRole("button", { name: "Tutup" }).last().click();
  await expect(agreement).toBeHidden();

  // 10. The quotation is now read-only and points at its project.
  await visit(page, `/penawaran/${quotationId}`);
  await expect(page.getByRole("button", { name: "Buka proyek" }).first()).toBeVisible();
  await expect(page.getByRole("button", { name: "Buat revisi untuk klien" })).toHaveCount(0);
  await close();
});

test("penawaran kedaluwarsa: tidak bisa ditandai diterima sebelum diperpanjang", async ({ browser }, info) => {
  test.setTimeout(120_000);
  const { page, close } = await open(browser, info, "paying");
  await visit(page, "/beranda");
  const { token } = await owner(page);
  const tag = Date.now().toString(36).slice(-5);
  const couple = `Wulan & Arga ${tag}`;

  // A quotation with Revisi 1, whose validity then passes (written straight into the table: the API refuses a past date).
  const quotationId = must("quotation", await call(apiURL(), token, "POST", "/api/v1/quotations", {
    coupleNames: couple, contactWhatsapp: "", picStaffId: 0, validUntil: "2026-11-30", notes: "",
  })).json.data.quotation.id as number;
  must("event", await call(apiURL(), token, "POST", `/api/v1/quotations/${quotationId}/events`, (await resepsiOn(token, EVENT_DATE))[0]));
  await saveLines(token, quotationId, PRICE_1);
  await saveTerms(token, quotationId);
  must("revision", await call(apiURL(), token, "POST", `/api/v1/quotations/${quotationId}/revisions`));
  sql(`UPDATE quotations SET valid_until = '2026-11-02' WHERE id = ${quotationId}`, config.dbName);
  expect((await detailOf(token, quotationId)).quotation.displayStatus).toBe("Expired");

  // Accepting it as it is is refused.
  const refused = await call(apiURL(), token, "POST", `/api/v1/quotations/${quotationId}/status`, {
    status: "Accepted", revisionN: 1, acceptedOn: "2026-11-09", channel: "WhatsApp",
  });
  expect(refused.status).toBe(422);
  expect(refused.json.errors.code).toBe("quotation_expired_needs_extension");

  // The page says so, and its first action is "Perpanjang masa berlaku".
  await visit(page, `/penawaran/${quotationId}`);
  await expect(page.getByText("Kedaluwarsa").first()).toBeVisible();
  await expect(page.getByText(/Masa berlaku lewat sejak/)).toBeVisible();
  await page.getByRole("button", { name: "Perpanjang masa berlaku" }).first().click();
  const extend = page.getByRole("dialog", { name: "Perpanjang masa berlaku" });
  await extend.getByLabel("Berlaku sampai").fill("2026-12-31");
  const extended = page.waitForResponse((r) => r.url().endsWith(`/quotations/${quotationId}/extend`) && r.request().method() === "POST");
  await extend.getByRole("button", { name: "Perpanjang" }).click();
  expect((await extended).status()).toBe(200);
  await expect(extend).toBeHidden();
  await expect(page.getByText(/Masa berlaku lewat sejak/)).toHaveCount(0);
  expect((await detailOf(token, quotationId)).quotation.displayStatus).toBe("Draft");

  // Now it can be marked accepted.
  must("accept", await call(apiURL(), token, "POST", `/api/v1/quotations/${quotationId}/status`, {
    status: "Accepted", revisionN: 1, acceptedOn: "2026-11-09", channel: "WhatsApp",
  }));
  await page.reload();
  await expect(page.getByRole("button", { name: "Jadikan proyek" }).first()).toBeVisible();
  await close();
});
