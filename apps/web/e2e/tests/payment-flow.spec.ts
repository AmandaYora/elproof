import { expect, state, test, visit } from "../lib/fixtures";
import { open } from "../lib/suite";
import { apiURL } from "../lib/config";

// PLAN Tahap 4: a project is created in one all-or-nothing call, and a payment is recorded together with its proof in
// one call that can be retried safely. These tests WRITE data, so they run in the "mutating" project (see config).

const API = (path: string) => `${apiURL()}/api/v1${path}`;
const PDF = Buffer.from("%PDF-1.4\n1 0 obj<<>>endobj\ntrailer<<>>\n%%EOF");

test("project creation is one all-or-nothing call; a bad row creates nothing", async ({ browser }, info) => {
  const { page, close } = await open(browser, info, "paying");
  await visit(page, "/dashboard");
  const session = await page.evaluate(() => JSON.parse(localStorage.getItem("elproof.session") ?? "{}") as { accessToken: string; principalId: string });
  const auth = { Authorization: `Bearer ${session.accessToken}` };
  const body = {
    name: "Uji Atomik", brideName: "Ani Lestari", groomName: "Budi Santoso", eventDate: "2027-05-01", venue: "Gedung Uji", prepStartDate: "2026-11-09",
    packageName: "Paket Uji", contractValue: 10_000_000, status: "Draft", picStaffId: Number(session.principalId), picSalesStaffId: 0, description: "",
  };

  const bad = await page.request.post(API("/projects"), {
    headers: { ...auth, "Idempotency-Key": `e2e-bad-${Date.now()}` },
    data: { ...body, name: "Uji Gagal", terms: [{ label: "DP", amount: 0, dueDate: "2026-12-01" }, { label: "", amount: 5_000_000, dueDate: "" }] },
  });
  expect(bad.status()).toBe(422);
  const errors = (await bad.json()).errors as Record<string, string[]>;
  // Every row problem is reported at once.
  expect(Object.keys(errors)).toEqual(expect.arrayContaining(["terms[0].amount", "terms[1].label", "terms[1].dueDate"]));
  const list = await page.request.get(API("/projects?limit=100&search=Uji%20Gagal"), { headers: auth });
  expect(JSON.stringify(await list.json())).not.toContain("Uji Gagal");

  const key = `e2e-ok-${Date.now()}`;
  const good = { ...body, terms: [{ label: "DP", amount: 3_000_000, dueDate: "2026-12-01" }, { label: "Pelunasan", amount: 7_000_000, dueDate: "2027-04-01" }] };
  const first = await page.request.post(API("/projects"), { headers: { ...auth, "Idempotency-Key": key }, data: good });
  expect(first.status()).toBe(201);
  const created = (await first.json()).data as { id: number; terms: unknown[] };
  expect(created.terms).toHaveLength(2);
  // The same key and body again: the same project comes back, nothing new is made.
  const again = await page.request.post(API("/projects"), { headers: { ...auth, "Idempotency-Key": key }, data: good });
  expect(again.headers()["idempotent-replay"]).toBe("true");
  expect(((await again.json()).data as { id: number }).id).toBe(created.id);
  await close();
});

test("a payment and its proof are one call: a failure keeps the drawer open with nothing saved, the retry saves once", async ({ browser }, info) => {
  test.skip(!state().storage, "needs object storage (MinIO)");
  const { page, close } = await open(browser, info, "paying");
  await visit(page, "/dashboard");
  const session = await page.evaluate(() => JSON.parse(localStorage.getItem("elproof.session") ?? "{}") as { accessToken: string; principalId: string });
  const auth = { Authorization: `Bearer ${session.accessToken}` };
  const made = await page.request.post(API("/projects"), {
    headers: auth,
    data: {
      name: "Uji Pembayaran", brideName: "Sari Dewi", groomName: "Eko Putra", eventDate: "2027-06-01", venue: "Gedung Uji", prepStartDate: "2026-11-09",
      packageName: "Paket Uji", contractValue: 10_000_000, status: "Draft", picStaffId: Number(session.principalId), picSalesStaffId: 0, description: "",
    },
  });
  expect(made.status()).toBe(201);
  const projectId = ((await made.json()).data as { id: number }).id;

  const posts: Array<{ url: string; method: string; key: string | undefined; body: string | null }> = [];
  page.on("request", (r) => {
    if (r.method() === "POST" || r.method() === "PUT") posts.push({ url: r.url(), method: r.method(), key: r.headers()["idempotency-key"], body: r.postData() });
  });

  await visit(page, `/projects/${projectId}/pembayaran/client`);
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

test("a project can be created with the keyboard alone, from the first field to the last button", async ({ browser }, info) => {
  const { page, close } = await open(browser, info, "paying");
  await visit(page, "/projects/baru");
  const bride = page.getByLabel(/Nama mempelai wanita/);
  await expect(bride).toBeFocused();
  await page.keyboard.type("Rani Wulandari");
  await page.getByLabel(/Nama mempelai pria/).focus();
  await page.keyboard.type("Dimas Aditya");
  await page.getByLabel(/Tanggal acara/).focus();
  await page.keyboard.type("01052027"); // dd/mm/yyyy in the id-ID locale
  await page.getByLabel(/Paket/).focus();
  await page.keyboard.type("Paket Keyboard");
  await page.getByLabel(/Nilai kontrak/).focus();
  await page.keyboard.type("12000000");
  await page.keyboard.press("Enter"); // submits step 1 from the last text field
  await expect(page.getByText(/Langkah 2 dari 2/)).toBeVisible();
  const create = page.getByRole("button", { name: "Buat proyek" });
  await create.focus();
  await page.keyboard.press("Enter");
  await expect(page).toHaveURL(/\/projects\/\d+/, { timeout: 20_000 });
  await close();
});
