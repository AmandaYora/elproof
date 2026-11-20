import { readFileSync } from "node:fs";
import path from "node:path";
import type { Browser, TestInfo } from "@playwright/test";
import { expect, prepareContext, state, test, visit } from "../lib/fixtures";
import { open } from "../lib/suite";
import { apiURL, E2E_ROOT, webURL, type Session } from "../lib/config";
import { call, must } from "../lib/api";
import { browserSession, sessionOf, storageStateFor } from "../lib/session";
import { resepsiOn, withContract } from "../lib/project";

// PLAN paket WO Tahap 4 "Vendor dibayar klien", in WO Berbayar: the photographer is paid by the couple directly. The WO
// writes its bill ("Tagihan vendor"); the Pengantin sees it on the portal (375 px), sends the transfer proof with
// "Kirim bukti bayar"; the Owner finds it in Keuangan › Verifikasi bukti and marks it as matching ("Tandai sesuai"), so
// the bill is paid. None of it is WO cash: Keuangan's and Beranda's money figures stay exactly as they were. The
// Keluarga (no portal.finance.view) sees none of it.
// This test WRITES data, so it runs in the "mutating" project (after every screenshot project).

const PROOF = readFileSync(path.join(E2E_ROOT, "lib/files/bukti-transfer.jpg"));
const MOBILE = { width: 375, height: 812 };
const BILL = 3_500_000;

interface BillReply {
  projectSupplierId: number;
  name: string;
  total: number;
  paid: number;
  remaining: number;
  pending: number;
  terms: Array<{ id: number; label: string; status: string; paid: number; pending: number; remaining: number }>;
  payments: Array<{ id: number; source: string; status: string; billTermId: number | null; amount: number }>;
}

/** A client of the project with the workspace's portal role `roleName`, signed in (password set through the access link). */
async function clientSession(token: string, projectId: number, roleName: "Pengantin" | "Keluarga", tag: string): Promise<Session> {
  const roles = must("client roles", await call(apiURL(), token, "GET", "/api/v1/access/roles?kind=client")).json.data as { id: number; name: string }[];
  const role = roles.find((r) => r.name === roleName)!;
  const bride = roleName === "Pengantin";
  const client = must(`client ${roleName}`, await call(apiURL(), token, "POST", "/api/v1/clients", {
    projectId, role: bride ? "Bride" : "Family Representative", relationNote: bride ? "" : "Ibu mempelai wanita",
    name: bride ? "Sekar Ayu" : "Ibu Wulan", phone: bride ? "081298764001" : "081298764002",
    username: `${bride ? "sekar" : "wulan"}.${tag}`, email: "", accessRoleId: role.id,
  }));
  const link = client.json.data.accessLink.token as string;
  return sessionOf(must(`${roleName} password`, await call(apiURL(), null, "POST", `/api/v1/auth/links/${link}/password`, { password: "Kata-sandi-klien-123" })));
}

/** A phone-sized browser context signed in as `session` (a client of the project). */
async function openAs(browser: Browser, info: TestInfo, session: Session) {
  const context = await browser.newContext({ ...info.project.use, viewport: MOBILE, storageState: storageStateFor(webURL(), session) });
  const page = await context.newPage();
  await prepareContext(page);
  return { page, close: () => context.close() };
}

test("tagihan vendor: Pengantin kirim bukti bayar fotografer di portal, Owner tandai sesuai, kas WO tetap", async ({ browser }, info) => {
  test.skip(!state().storage, "needs object storage (MinIO)");
  test.setTimeout(150_000);
  const { page, close } = await open(browser, info, "paying");
  await visit(page, "/beranda");
  const session = await browserSession(page);
  const token = session.accessToken;
  const staffId = Number(session.principalId);
  const tag = Date.now().toString(36).slice(-5);

  // WO cash, read before and after: what this project brings the WO (the project's Keuangan header: paid, remaining,
  // margin) and the WO cash Beranda and Keuangan add up from it. Workspace-wide totals are not compared: other
  // mutating tests write payments in the same workspace at the same time.
  const cashNow = async () => {
    const overview = must("overview", await call(apiURL(), token, "GET", `/api/v1/projects/${projectId}/overview`)).json.data as { finance: Record<string, unknown> };
    const { paid, remaining, overdueTotal, contractValue, supplierCost, margin } = overview.finance;
    const received = must("client payments", await call(apiURL(), token, "GET", `/api/v1/projects/${projectId}/client-payments`)).json.data as unknown[];
    return { paid, remaining, overdueTotal, contractValue, supplierCost, margin, clientPayments: received.length };
  };

  // A project with a contract, and a photographer the couple pays directly, with one bill.
  const projectName = `Sekar & Bima ${tag}`;
  const made = must("project", await call(apiURL(), token, "POST", "/api/v1/projects", {
    name: projectName, brideName: "Sekar Ayu", groomName: "Bima Pratama", events: await resepsiOn(token, "2027-09-18"), prepStartDate: "2026-11-09",
    status: "Preparation", picStaffId: staffId, picSalesStaffId: 0, description: "",
  }));
  const projectId = made.json.data.id as number;
  await withContract(token, projectId, 80_000_000);
  const cats = must("categories", await call(apiURL(), token, "GET", "/api/v1/vendor-categories?all=true")).json.data as { id: number; name: string }[];
  const category = cats.find((c) => /foto|dokumentasi/i.test(c.name)) ?? cats[0];
  const vendorName = `Lensa Senja Foto ${tag}`;
  const vendorId = must("vendor", await call(apiURL(), token, "POST", "/api/v1/vendors", {
    name: vendorName, categoryId: category.id, picName: "Mas Arif", phone: "081211110288", email: "", socialMedia: "", city: "Kota Bandung", address: "", notes: "",
  })).json.data.id as number;
  const supplierId = must("client-paid supplier", await call(apiURL(), token, "POST", `/api/v1/projects/${projectId}/suppliers`, {
    vendorId, paidBy: "Client", showName: true, agreedCost: null, engagementStatus: "Booked",
    bookingDate: "", dueDate: "2027-08-01", picStaffId: staffId, scope: "Dokumentasi foto", notes: "",
  })).json.data.id as number;
  const cashBefore = await cashNow();
  must("bill term", await call(apiURL(), token, "POST", `/api/v1/projects/${projectId}/supplier-bills/terms`, {
    projectSupplierId: supplierId, label: "DP fotografer", amount: BILL, dueDate: "2026-12-01",
  }, { "Idempotency-Key": `e2e-bill-${projectId}` }));
  const billsNow = async () =>
    (must("supplier bills", await call(apiURL(), token, "GET", `/api/v1/projects/${projectId}/supplier-bills`)).json.data as BillReply[]).find(
      (b) => b.projectSupplierId === supplierId,
    )!;

  // The WO sees the bill in Keuangan › Dibayar klien langsung.
  await visit(page, `/proyek/${projectId}/keuangan/dibayar-klien`);
  const terms = page.getByRole("table", { name: "Tagihan vendor", exact: true });
  await expect(terms.getByRole("row").filter({ hasText: "DP fotografer" })).toContainText(vendorName);

  // Keluarga (no portal.finance.view): the API refuses the bills; the portal has no Pembayaran and no vendor bill.
  const famSession = await clientSession(token, projectId, "Keluarga", tag);
  const refused = await call(apiURL(), famSession.accessToken, "GET", `/api/v1/projects/${projectId}/supplier-bills`);
  expect(refused.status, "Keluarga may not read the vendor bills").toBe(403);
  const keluarga = await openAs(browser, info, famSession);
  await visit(keluarga.page, "/portal/beranda");
  await expect(keluarga.page.getByRole("main")).toBeVisible();
  await expect(keluarga.page.getByRole("link", { name: "Pembayaran" })).toHaveCount(0);
  await expect(keluarga.page.getByText("Tagihan vendor berikutnya")).toHaveCount(0);
  await expect(keluarga.page.getByText(/Tagihan vendor/i)).toHaveCount(0);
  await keluarga.close();

  // Pengantin, on a phone: Beranda shows the next vendor bill; Pembayaran › Tagihan vendor sends the proof.
  const pengantin = await openAs(browser, info, await clientSession(token, projectId, "Pengantin", tag));
  const portal = pengantin.page;
  await visit(portal, "/portal/beranda");
  const next = portal.getByRole("region", { name: "Tagihan vendor berikutnya" });
  await expect(next).toContainText(vendorName);
  await expect(next).toContainText("DP fotografer");
  await expect(next).toContainText("Rp3.500.000");
  await next.getByRole("link", { name: "Lihat tagihan vendor" }).click();
  await expect(portal).toHaveURL(/\/portal\/pembayaran#tagihan-vendor$/);
  const section = portal.getByRole("region", { name: "Tagihan vendor", exact: true });
  await expect(section).toContainText(vendorName);
  await expect(section).toContainText("di luar nilai kontrak");
  await section.getByRole("button", { name: `Kirim bukti bayar DP fotografer ${vendorName}` }).click();
  const drawer = portal.getByRole("dialog", { name: "Kirim bukti bayar" });
  await expect(drawer).toBeVisible();
  await drawer.locator('input[type="file"]').setInputFiles({ name: "bukti-dp-foto.jpg", mimeType: "image/jpeg", buffer: PROOF });
  await drawer.getByLabel("No. referensi").fill(`TRF-FOTO-${tag}`);
  const sent = portal.waitForResponse((r) => r.url().endsWith(`/projects/${projectId}/supplier-bills/submissions`) && r.request().method() === "POST");
  await drawer.getByRole("button", { name: "Kirim bukti bayar" }).click();
  const submission = await sent;
  expect(submission.status()).toBe(201);
  expect(submission.request().headers()["idempotency-key"], "a retry never sends the proof twice").toBeTruthy();
  await expect(portal.getByText("Bukti pembayaran terkirim.").first()).toBeVisible();
  await expect(drawer).toBeHidden();
  await expect(section.getByText("Menunggu diperiksa wedding organizer Anda", { exact: true })).toBeVisible();
  // Sent once: no second "Kirim bukti bayar" while the whole bill waits to be checked.
  await expect(section.getByRole("button", { name: `Kirim bukti bayar DP fotografer ${vendorName}` })).toHaveCount(0);

  let bill = await billsNow();
  expect(bill.terms[0]).toMatchObject({ paid: 0, pending: BILL });
  expect(bill.terms[0].status, "a proof not checked yet does not pay the bill").not.toBe("paid");
  expect(bill.payments).toHaveLength(1);
  expect(bill.payments[0]).toMatchObject({ source: "client", status: "pending", amount: BILL, billTermId: bill.terms[0].id });

  // Owner: Keuangan › Verifikasi bukti lists it as a vendor proof; Dibayar klien langsung checks it.
  await visit(page, "/keuangan?tab=verifikasi");
  const queued = page.getByRole("row").filter({ hasText: projectName });
  await expect(queued).toBeVisible();
  await expect(queued.getByText("Bukti bayar vendor")).toBeVisible();
  await expect(queued).toContainText(vendorName);

  await visit(page, `/proyek/${projectId}/keuangan/dibayar-klien`);
  const waiting = page.getByRole("status").filter({ hasText: "1 bukti bayar vendor menunggu diperiksa" });
  await expect(waiting).toContainText(vendorName);
  await waiting.getByRole("button", { name: "Periksa bukti" }).click();
  await expect(page).toHaveURL(/[?&]periksa=/);
  const review = page.getByRole("dialog", { name: "Periksa bukti bayar vendor" });
  await expect(review).toContainText(vendorName);
  const verified = page.waitForResponse((r) => /\/supplier-payments\/\d+\/verify$/.test(r.url()) && r.request().method() === "POST");
  await review.getByRole("button", { name: "Tandai sesuai" }).click();
  expect((await verified).status()).toBe(200);
  await expect(page.getByText("Bukti pembayaran ditandai sesuai").first()).toBeVisible();
  await expect(review).toBeHidden();
  await expect(page.getByRole("status").filter({ hasText: "bukti bayar vendor menunggu diperiksa" })).toHaveCount(0);

  // The bill is paid; it left the queue.
  bill = await billsNow();
  expect(bill).toMatchObject({ total: BILL, paid: BILL, remaining: 0, pending: 0 });
  expect(bill.terms[0]).toMatchObject({ status: "paid", paid: BILL, remaining: 0, pending: 0 });
  expect(bill.payments[0].status).toBe("verified");
  await visit(page, "/keuangan?tab=verifikasi");
  await expect(page.getByRole("row").filter({ hasText: projectName })).toHaveCount(0);

  // WO cash is untouched: no payment to the WO, the same contract money on the project.
  expect(await cashNow(), "a vendor proof is never WO cash").toEqual(cashBefore);
  expect(cashBefore.clientPayments).toBe(0);

  // The couple sees it counted.
  await portal.reload();
  await expect(section.getByText("Sesuai", { exact: true })).toBeVisible();
  await expect(section.getByRole("button", { name: /^Kirim bukti bayar/ })).toHaveCount(0);
  await pengantin.close();
  await close();
});
