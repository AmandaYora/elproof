import { readFileSync } from "node:fs";
import { expect, state, test, visit } from "../lib/fixtures";
import { open } from "../lib/suite";
import { apiURL } from "../lib/config";
import { browserSession } from "../lib/session";
import { choose, resepsiOn, withContract } from "../lib/project";

// PLAN Tahap 4: a project is created in one all-or-nothing call, and a payment is recorded together with its proof in
// one call that can be retried safely. These tests WRITE data, so they run in the "mutating" project (see config).

const API = (path: string) => `${apiURL()}/api/v1${path}`;
const PDF = Buffer.from("%PDF-1.4\n1 0 obj<<>>endobj\ntrailer<<>>\n%%EOF");

test("project creation is one all-or-nothing call; a bad row creates nothing", async ({ browser }, info) => {
  const { page, close } = await open(browser, info, "paying");
  await visit(page, "/beranda");
  const session = await browserSession(page);
  const auth = { Authorization: `Bearer ${session.accessToken}` };
  const body = {
    name: "Uji Atomik", brideName: "Ani Lestari", groomName: "Budi Santoso", events: await resepsiOn(session.accessToken, "2027-05-01"), prepStartDate: "2026-11-09",
    status: "Draft", picStaffId: Number(session.principalId), picSalesStaffId: 0, description: "",
  };

  // The second event names a venue that does not exist and has no sessions: nothing is created, every problem comes back.
  const [resepsi] = body.events;
  const bad = await page.request.post(API("/projects"), {
    headers: { ...auth, "Idempotency-Key": `e2e-bad-${Date.now()}` },
    data: { ...body, name: "Uji Gagal", events: [resepsi, { ...resepsi, eventDate: "", venueId: 999_999, sessions: [] }] },
  });
  expect(bad.status()).toBe(422);
  const errors = (await bad.json()).errors as Record<string, string[]>;
  expect(Object.keys(errors).some((k) => k.startsWith("events.1."))).toBe(true);
  const list = await page.request.get(API("/projects?limit=100&search=Uji%20Gagal"), { headers: auth });
  expect(JSON.stringify(await list.json())).not.toContain("Uji Gagal");

  const key = `e2e-ok-${Date.now()}`;
  const good = body;
  const first = await page.request.post(API("/projects"), { headers: { ...auth, "Idempotency-Key": key }, data: good });
  expect(first.status()).toBe(201);
  const created = (await first.json()).data as { id: number; events: unknown[] };
  expect(created.events).toHaveLength(1);
  // The same key and body again: the same project comes back, nothing new is made.
  const again = await page.request.post(API("/projects"), { headers: { ...auth, "Idempotency-Key": key }, data: good });
  expect(again.headers()["idempotent-replay"]).toBe("true");
  expect(((await again.json()).data as { id: number }).id).toBe(created.id);
  await close();
});

test("a payment and its proof are one call: a failure keeps the drawer open with nothing saved, the retry saves once", async ({ browser }, info) => {
  test.skip(!state().storage, "needs object storage (MinIO)");
  const { page, close } = await open(browser, info, "paying");
  await visit(page, "/beranda");
  const session = await browserSession(page);
  const auth = { Authorization: `Bearer ${session.accessToken}` };
  const made = await page.request.post(API("/projects"), {
    headers: auth,
    data: {
      name: "Uji Pembayaran", brideName: "Sari Dewi", groomName: "Eko Putra", events: await resepsiOn(session.accessToken, "2027-06-01"), prepStartDate: "2026-11-09",
      status: "Draft", picStaffId: Number(session.principalId), picSalesStaffId: 0, description: "",
    },
  });
  expect(made.status()).toBe(201);
  const projectId = ((await made.json()).data as { id: number }).id;
  await withContract(session.accessToken, projectId, 10_000_000);

  const posts: Array<{ url: string; method: string; key: string | undefined; body: string | null }> = [];
  page.on("request", (r) => {
    if (r.method() === "POST" || r.method() === "PUT") posts.push({ url: r.url(), method: r.method(), key: r.headers()["idempotency-key"], body: r.postData() });
  });

  await visit(page, `/proyek/${projectId}/keuangan/klien`);
  await page.getByRole("button", { name: "Tambah Pembayaran" }).click();
  const drawer = page.getByRole("dialog");
  await expect(drawer).toBeVisible();
  await drawer.getByLabel(/Nominal/).fill("1000000");
  await drawer.locator('input[type="file"]').setInputFiles({ name: "bukti.pdf", mimeType: "application/pdf", buffer: PDF });

  // First try: the server fails. Nothing is saved, the drawer stays open, what was typed is still there.
  let failOnce = true;
  await page.route(/\/api\/v1\/projects\/\d+\/client-payments$/, async (route) => {
    if (route.request().method() === "POST" && failOnce) {
      failOnce = false;
      return route.fulfill({ status: 500, contentType: "application/json", body: JSON.stringify({ success: false, message: "Server sedang bermasalah", errors: null }) });
    }
    return route.fallback();
  });
  await drawer.getByRole("button", { name: "Simpan pembayaran" }).click();
  await expect(drawer.getByRole("alert")).toContainText(/bermasalah|belum tersimpan/i);
  await expect(drawer).toBeVisible();
  await expect(drawer.getByLabel(/Nominal/)).toHaveValue(/1[.,]?000[.,]?000/);

  // Retry: it saves, once, with the proof attached in the same request.
  await drawer.getByRole("button", { name: "Simpan pembayaran" }).click();
  await expect(drawer).toBeHidden();
  const rows = page.getByRole("row").filter({ hasText: "Rp1.000.000" });
  await expect(rows).toHaveCount(1);
  await expect(rows.first()).toContainText("Bukti ada");

  const payments = posts.filter((p) => /\/client-payments$/.test(p.url) && p.method === "POST");
  expect(payments.length, "one failed try and one retry").toBe(2);
  for (const p of payments) {
    expect(p.key, "every save carries an Idempotency-Key").toMatch(/^[A-Za-z0-9_-]{8,128}$/);
    expect(p.body ?? "").toContain("uploadToken");
  }
  // The file went to storage with a PUT (direct or through the API), never as base64 inside the payment.
  expect(posts.some((p) => p.method === "PUT")).toBe(true);
  expect(payments.every((p) => !(p.body ?? "").includes("base64"))).toBe(true);
  await close();
});

test("a JPG transfer proof is saved, and a file only named .jpg is refused under its own field (EP-01, EP-02)", async ({ browser }, info) => {
  test.skip(!state().storage, "needs object storage (MinIO)");
  const { page, close } = await open(browser, info, "paying");
  await visit(page, "/beranda");
  const session = await browserSession(page);
  const made = await page.request.post(API("/projects"), {
    headers: { Authorization: `Bearer ${session.accessToken}` },
    data: {
      name: "Uji Bukti JPG", brideName: "Wulan Sari", groomName: "Bayu Aji", events: await resepsiOn(session.accessToken, "2027-07-01"), prepStartDate: "2026-11-09",
      status: "Draft", picStaffId: Number(session.principalId), picSalesStaffId: 0, description: "",
    },
  });
  expect(made.status()).toBe(201);
  const projectId = ((await made.json()).data as { id: number }).id;
  await withContract(session.accessToken, projectId, 10_000_000);

  await visit(page, `/proyek/${projectId}/keuangan/klien`);
  await page.getByRole("button", { name: "Tambah Pembayaran" }).click();
  const drawer = page.getByRole("dialog");
  await drawer.getByLabel(/Nominal/).fill("1000000");

  // A binary renamed to .jpg: refused, with the reason under "Bukti transfer", the drawer stays open (was silent before).
  const file = drawer.locator('input[type="file"]');
  await file.setInputFiles({ name: "bukti.jpg", mimeType: "image/jpeg", buffer: Buffer.from("MZ\x90 not a photo at all, whatever the name says") });
  await drawer.getByRole("button", { name: "Simpan pembayaran" }).click();
  await expect(drawer.getByText(/bukan foto JPG yang bisa dibuka/)).toBeVisible();
  await expect(drawer).toBeVisible();

  // A real phone-style JPG: saved with its proof.
  await file.setInputFiles({ name: "IMG_2041.jpg", mimeType: "image/jpeg", buffer: readFileSync(new URL("../lib/files/bukti-transfer.jpg", import.meta.url)) });
  await drawer.getByRole("button", { name: "Simpan pembayaran" }).click();
  await expect(drawer).toBeHidden();
  const rows = page.getByRole("row").filter({ hasText: "Rp1.000.000" });
  await expect(rows).toHaveCount(1);
  await expect(rows.first()).toContainText("Bukti ada");

  // QA RT-01: right after saving, "Lihat bukti" shows that photo (it said "Belum ada bukti" while the page's document
  // list, loaded before the proof existed, was what it searched).
  await rows.first().getByRole("button", { name: /^Aksi lain untuk/ }).click();
  await page.getByRole("menuitem", { name: "Lihat bukti" }).click();
  const viewer = page.getByRole("dialog");
  await expect(viewer.locator("img")).toBeVisible();
  await expect(viewer.getByText("Belum ada bukti")).toHaveCount(0);
  await expect.poll(() => viewer.locator("img").evaluate((img: HTMLImageElement) => img.complete && img.naturalWidth > 0)).toBe(true);
  await page.keyboard.press("Escape");
  await expect(viewer).toBeHidden();
  await close();
});

test("moving Hari H offers to move the unpaid installments with it; paid ones stay (QA RT-04)", async ({ browser }, info) => {
  const { page, close } = await open(browser, info, "paying");
  await visit(page, "/beranda");
  const session = await browserSession(page);
  const auth = { Authorization: `Bearer ${session.accessToken}` };
  const made = await page.request.post(API("/projects"), {
    headers: auth,
    data: {
      name: "Uji Geser Termin", brideName: "Nadia Putri", groomName: "Rio Saputra", events: await resepsiOn(session.accessToken, "2027-06-01"), prepStartDate: "2026-11-09",
      status: "Draft", picStaffId: Number(session.principalId), picSalesStaffId: 0, description: "",
    },
  });
  expect(made.status()).toBe(201);
  const projectId = ((await made.json()).data as { id: number }).id;
  await withContract(session.accessToken, projectId, 10_000_000);
  const schedule = await page.request.put(API(`/projects/${projectId}/payment-terms`), {
    headers: auth,
    data: { terms: [{ label: "DP", amount: 3_000_000, dueDate: "2026-12-01" }, { label: "Pelunasan", amount: 7_000_000, dueDate: "2027-05-25" }] },
  });
  expect(schedule.status()).toBe(200);
  // The DP is paid in full; Pelunasan is not.
  const paid = await page.request.post(API(`/projects/${projectId}/client-payments`), {
    headers: auth,
    data: { type: "DP", amount: 3_000_000, paymentDate: "2026-11-09", method: "Transfer", referenceNumber: "", notes: "" },
  });
  expect(paid.status()).toBe(201);

  // Hari H is the main event's date: moving that event is what moves it.
  await visit(page, `/proyek/${projectId}/acara-rincian`);
  await page.getByRole("button", { name: "Ubah acara Resepsi" }).click();
  const drawer = page.getByRole("dialog", { name: /Ubah acara/ });
  await drawer.getByRole("textbox", { name: /^Tanggal \(wajib diisi\)/ }).fill("2027-06-29");
  await drawer.getByRole("button", { name: "Simpan acara" }).click();
  await expect(drawer).toBeHidden();

  const ask = page.getByRole("alertdialog", { name: "Geser jatuh tempo termin?" });
  await expect(ask).toBeVisible();
  await expect(ask).toContainText("Hari H mundur 28 hari");
  await expect(ask.getByRole("button", { name: "Biarkan" })).toBeVisible();
  const saved = page.waitForResponse((r) => r.request().method() === "PUT" && r.url().endsWith(`/projects/${projectId}/payment-terms`));
  await ask.getByRole("button", { name: "Geser jatuh tempo" }).click();
  expect((await saved).status()).toBe(200);
  await expect(ask).toBeHidden();
  await expect(page.getByText("Jatuh tempo termin digeser")).toBeVisible();

  const terms = (await (await page.request.get(API(`/projects/${projectId}/payment-terms`), { headers: auth })).json()).data as Array<{ label: string; dueDate: string; amount: number }>;
  expect(terms.map((t) => [t.label, t.dueDate, t.amount])).toEqual([
    ["DP", "2026-12-01", 3_000_000],
    ["Pelunasan", "2027-06-22", 7_000_000],
  ]);
  await close();
});

test("a project can be created with the keyboard alone, from the first field to the last button", async ({ browser }, info) => {
  const { page, close } = await open(browser, info, "paying");
  await visit(page, "/proyek/baru");
  const bride = page.getByLabel(/Nama mempelai wanita/);
  await expect(bride).toBeFocused();
  await page.keyboard.type("Rani Wulandari");
  await page.getByLabel(/Nama mempelai pria/).focus();
  await page.keyboard.type("Dimas Aditya");
  await page.keyboard.press("Enter"); // submits step 1 from a text field (no Paket or Nilai kontrak: the rincian holds them)
  await expect(page.getByText(/Langkah 2 dari 2/)).toBeVisible();
  const event = page.locator("ol > li").first();
  await event.getByLabel("Jenis acara").focus();
  await page.keyboard.press("ArrowDown"); // opens the list of kinds, its search field takes the focus
  await expect(page.getByPlaceholder("Cari...")).toBeFocused();
  await page.keyboard.press("Enter"); // the first kind
  await expect(event.getByRole("button", { name: /^Jenis acara: (?!Pilih)/ })).toBeFocused();
  await event.getByRole("textbox", { name: /^Tanggal \(wajib diisi\)/ }).focus();
  await page.keyboard.type("01052027"); // dd/mm/yyyy in the id-ID locale
  await event.getByLabel("Tamu sesi 1").focus();
  await page.keyboard.type("200");
  const create = page.getByRole("button", { name: "Buat proyek" });
  await create.focus();
  await page.keyboard.press("Enter");
  await expect(page).toHaveURL(/\/proyek\/\d+\/acara-rincian$/, { timeout: 20_000 });
  await close();
});

test("new project: no Paket or Nilai kontrak, an event is required, a past event date is confirmed", async ({ browser }, info) => {
  // EP-05 (QA report). Nothing is created: the test stops before the confirmation. The contract value is no longer
  // typed (it is the rincian's total, Tahap 3), so the old negative-value check has nothing left to test.
  const { page, close } = await open(browser, info, "paying");
  await visit(page, "/proyek/baru");
  await page.getByLabel(/Nama mempelai wanita/).fill("Tari Anggraini");
  await page.getByLabel(/Nama mempelai pria/).fill("Galih Pratama");
  await expect(page.getByLabel(/Nilai kontrak/)).toHaveCount(0);
  await expect(page.getByLabel(/^Paket/)).toHaveCount(0);

  await page.getByRole("button", { name: "Lanjut", exact: true }).click();
  await expect(page.getByText(/Langkah 2 dari 2/)).toBeVisible();
  // No payment-term step any more: the schedule is made once the rincian is saved.
  await expect(page.getByLabel(/Nominal/)).toHaveCount(0);

  // An event without its kind and date is not sent.
  const posts: string[] = [];
  page.on("request", (r) => {
    if (r.method() === "POST" && r.url().endsWith("/api/v1/projects")) posts.push(r.url());
  });
  await page.getByRole("button", { name: "Buat proyek" }).click();
  await expect(page.locator("ol > li").first().getByRole("alert").first()).toBeVisible();
  expect(posts).toHaveLength(0);

  const event = page.locator("ol > li").first();
  await choose(event.getByRole("button", { name: /^Jenis acara/ }), "Resepsi");
  await event.getByRole("textbox", { name: /^Tanggal \(wajib diisi\)/ }).fill("2026-01-10");
  await event.getByLabel("Tamu sesi 1").fill("100");
  await page.getByRole("button", { name: "Buat proyek" }).click();
  const dialog = page.getByRole("alertdialog");
  await expect(dialog.getByText("Tanggal acara sudah lewat")).toBeVisible();
  await dialog.getByRole("button", { name: "Batal" }).click();
  expect(posts).toHaveLength(0);
  await close();
});
