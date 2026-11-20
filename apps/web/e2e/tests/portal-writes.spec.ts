import { readFileSync } from "node:fs";
import path from "node:path";
import type { APIRequestContext, Page } from "@playwright/test";
import { expect, state, test, visit } from "../lib/fixtures";
import { open } from "../lib/suite";
import { apiURL, BASE_DAY, E2E_ROOT } from "../lib/config";
import { browserSession } from "../lib/session";
import { call, must } from "../lib/api";
import { resepsiOn, withContract } from "../lib/project";

// ADR-0047: the Client Portal gains a cover photo (uploaded by the WO) and two write paths for the couple: a transfer
// proof that waits in Keuangan "Verifikasi bukti", and marking a task done. The portal screens for the couple are built
// separately, so the couple's side is driven through the API here; the WO side through the console. These tests WRITE
// data, so they run in the "mutating" project, in the "paying" workspace.

const API = (p: string) => `${apiURL()}/api/v1${p}`;
const PHOTO = readFileSync(path.join(E2E_ROOT, "lib/files/bukti-transfer.jpg"));

interface Staff {
  auth: Record<string, string>;
  principalId: number;
}

async function staffOf(page: Page): Promise<Staff> {
  await visit(page, "/beranda");
  const s = await browserSession(page);
  return { auth: { Authorization: `Bearer ${s.accessToken}` }, principalId: Number(s.principalId) };
}

async function newProject(req: APIRequestContext, staff: Staff, name: string): Promise<number> {
  const made = await req.post(API("/projects"), {
    headers: staff.auth,
    data: {
      name, brideName: "Laras Ayu", groomName: "Dimas Putra", events: await resepsiOn(staff.auth.Authorization.slice(7), "2027-07-10"), prepStartDate: "2026-11-09",
      status: "Preparation", picStaffId: staff.principalId, picSalesStaffId: 0, description: "",
    },
  });
  expect(made.status()).toBe(201);
  const id = ((await made.json()).data as { id: number }).id;
  await withContract(staff.auth.Authorization.slice(7), id, 20_000_000);
  return id;
}

/** Adds the bride with the workspace's "Pengantin" portal role and returns her access token. */
async function brideToken(staff: Staff, projectId: number): Promise<string> {
  const token = staff.auth.Authorization.slice("Bearer ".length);
  const roles = must("client roles", await call(apiURL(), token, "GET", "/api/v1/access/roles?kind=client")).json.data as { id: number; name: string }[];
  const role = roles.find((r) => r.name === "Pengantin") ?? roles[0];
  const username = `laras.${projectId}.${Date.now() % 1_000_000}`;
  const client = must("bride", await call(apiURL(), token, "POST", "/api/v1/clients", {
    projectId, role: "Bride", relationNote: "", name: "Laras Ayu", phone: "081298761234", username, email: "", accessRoleId: role.id,
  }));
  const link = client.json.data.accessLink.token as string;
  const login = must("bride password", await call(apiURL(), null, "POST", `/api/v1/auth/links/${link}/password`, { password: "Kata-sandi-klien-123" }));
  return login.json.data.accessToken as string;
}

test("the WO uploads, replaces and removes the portal cover photo on Ringkasan", async ({ browser }, info) => {
  const { page, close } = await open(browser, info, "paying");
  const staff = await staffOf(page);
  const projectId = await newProject(page.request, staff, "Uji Sampul");
  const bride = await brideToken(staff, projectId);

  await visit(page, `/proyek/${projectId}/ringkasan`);
  const card = page.getByRole("region", { name: "Foto sampul portal" });
  await expect(card.getByText("Tampil di Portal Klien sebagai latar hero.")).toBeVisible();
  await expect(card.getByRole("figure")).toHaveAccessibleName(/tanpa foto sampul/);

  await card.locator('input[type="file"]').setInputFiles({ name: "sampul.jpg", mimeType: "image/jpeg", buffer: PHOTO });
  await expect(page.getByText("Foto sampul disimpan")).toBeVisible();
  await expect(card.getByRole("figure")).toHaveAccessibleName(/dengan foto sampul/);
  await expect(card.locator("figure img")).toHaveAttribute("src", /^blob:/);
  await expect(card.getByRole("button", { name: "Ganti foto" })).toBeVisible();

  // Every read the console and the portal use carries the same versioned URL, and the couple can open it.
  const project = (await (await page.request.get(API(`/projects/${projectId}`), { headers: staff.auth })).json()).data as { coverUrl: string | null };
  expect(project.coverUrl).toMatch(new RegExp(`^/api/v1/projects/${projectId}/cover\\?v=\\d+$`));
  const home = must("home", await call(apiURL(), bride, "GET", "/api/v1/projects/me/home"));
  expect(home.json.data.project.coverUrl).toBe(project.coverUrl);
  const img = await fetch(apiURL() + project.coverUrl, { headers: { Authorization: `Bearer ${bride}` } });
  expect(img.status).toBe(200);
  expect(img.headers.get("content-type")).toBe("image/jpeg");
  expect(img.headers.get("cache-control")).toBe("private, max-age=300");
  // The couple cannot change it.
  expect((await call(apiURL(), bride, "DELETE", `/api/v1/projects/${projectId}/cover`)).status).toBe(403);

  // Replacing it: a new version in the URL, so no browser shows the old photo from its cache.
  await page.waitForTimeout(1100);
  await card.locator('input[type="file"]').setInputFiles({ name: "sampul-baru.jpg", mimeType: "image/jpeg", buffer: PHOTO });
  await expect(page.getByText("Foto sampul diganti")).toBeVisible();
  const replaced = (await (await page.request.get(API(`/projects/${projectId}`), { headers: staff.auth })).json()).data as { coverUrl: string | null };
  expect(replaced.coverUrl).not.toBe(project.coverUrl);
  await expect(card.locator("figure img")).toHaveAttribute("src", /^blob:/);

  await card.getByRole("button", { name: "Hapus foto" }).click();
  await expect(page.getByText("Foto sampul dihapus")).toBeVisible();
  await expect(card.getByRole("figure")).toHaveAccessibleName(/tanpa foto sampul/);
  const after = must("home", await call(apiURL(), bride, "GET", "/api/v1/projects/me/home"));
  expect(after.json.data.project.coverUrl).toBeNull();
  await close();
});

test("the couple sends a transfer proof: it waits in Verifikasi bukti as Dikirim pasangan, and the WO verifies it", async ({ browser }, info) => {
  test.skip(!state().storage, "needs object storage (MinIO)");
  const { page, close } = await open(browser, info, "paying");
  const staff = await staffOf(page);
  const projectName = `Uji Bukti Pasangan ${Date.now() % 100_000}`;
  const projectId = await newProject(page.request, staff, projectName);
  const bride = await brideToken(staff, projectId);

  // The couple: a signed upload of the photo, then the submission (what the portal will do).
  const grant = must("grant", await call(apiURL(), bride, "POST", `/api/v1/projects/${projectId}/uploads`, { fileName: "bukti.jpg", mimeType: "image/jpeg", size: PHOTO.length }));
  const { uploadToken, url, direct } = grant.json.data as { uploadToken: string; url: string; direct: boolean };
  const put = await fetch(direct ? url : apiURL() + url, {
    method: "PUT",
    headers: direct ? { "Content-Type": "image/jpeg" } : { "Content-Type": "image/jpeg", Authorization: `Bearer ${bride}` },
    body: PHOTO,
  });
  expect(put.status).toBe(200);
  const submitted = await call(apiURL(), bride, "POST", `/api/v1/projects/${projectId}/client-payments/submissions`, {
    amount: 5_000_000, paymentDate: BASE_DAY, method: "Transfer Bank", referenceNumber: "TRF-LARAS-1", evidence: [{ uploadToken, type: "Transfer Proof" }],
  }, { "Idempotency-Key": `e2e-submit-${projectId}` });
  expect(submitted.status).toBe(201);
  expect(submitted.json.data).toMatchObject({ source: "client", verifiedAt: null, recordedBy: null, type: "DP", amount: 5_000_000 });

  // The WO: Keuangan, "Verifikasi bukti".
  await visit(page, "/keuangan?tab=verifikasi");
  const row = page.getByRole("row").filter({ hasText: projectName });
  await expect(row).toBeVisible();
  await expect(row.getByText("Dikirim pasangan")).toBeVisible();

  // The project ledger shows the same badge; verifying works as for any payment.
  await visit(page, `/proyek/${projectId}/keuangan/klien`);
  const ledgerRow = page.getByRole("row").filter({ hasText: "TRF-LARAS-1" });
  await expect(ledgerRow.getByText("Dikirim pasangan")).toBeVisible();
  await ledgerRow.getByRole("button", { name: /^Verifikasi DP/ }).click();
  await page.getByRole("alertdialog").getByRole("button", { name: "Verifikasi pembayaran" }).click();
  await expect(ledgerRow.getByRole("button", { name: /Diverifikasi/ })).toBeVisible();

  // The couple sees it verified; it has left the queue.
  const list = must("client list", await call(apiURL(), bride, "GET", `/api/v1/projects/${projectId}/client-payments`)).json.data as Array<{ source: string; verifiedAt: string | null }>;
  expect(list).toHaveLength(1);
  expect(list[0].source).toBe("client");
  expect(list[0].verifiedAt).not.toBeNull();
  await visit(page, "/keuangan?tab=verifikasi");
  await expect(page.getByRole("row").filter({ hasText: projectName })).toHaveCount(0);
  await close();
});

test("the WO flags a step as a task for the couple, and the couple marks it done", async ({ browser }, info) => {
  const { page, close } = await open(browser, info, "paying");
  const staff = await staffOf(page);
  const projectId = await newProject(page.request, staff, "Uji Tugas Pasangan");
  const bride = await brideToken(staff, projectId);
  const step = await page.request.post(API(`/projects/${projectId}/milestones`), { headers: staff.auth, data: { name: "Kirim daftar tamu", targetDate: "2026-12-01" } });
  expect(step.status()).toBe(201);
  const stepId = ((await step.json()).data as { id: number }).id;

  // Before it is flagged, the couple cannot complete it.
  const refused = await call(apiURL(), bride, "POST", `/api/v1/projects/${projectId}/milestones/${stepId}/complete`);
  expect(refused.status).toBe(403);
  expect(refused.json.errors.code).toBe("not_client_task");

  await visit(page, `/proyek/${projectId}/jadwal`);
  const row = page.getByRole("row").filter({ hasText: "Kirim daftar tamu" });
  await row.getByRole("button", { name: "Ubah tahapan" }).click();
  const dialog = page.getByRole("dialog");
  await dialog.getByRole("checkbox", { name: /Tugas untuk pasangan/ }).check();
  await dialog.getByRole("button", { name: "Simpan perubahan" }).click();
  await expect(dialog).toBeHidden();
  await expect(row.getByText("Tugas pasangan")).toBeVisible();

  const home = must("home", await call(apiURL(), bride, "GET", "/api/v1/projects/me/home"));
  expect(home.json.data.clientTasks).toEqual([{ id: stepId, name: "Kirim daftar tamu", targetDate: "2026-12-01", status: "Not Started", completedDate: null }]);

  const done = await call(apiURL(), bride, "POST", `/api/v1/projects/${projectId}/milestones/${stepId}/complete`);
  expect(done.status).toBe(200);
  expect(done.json.data).toMatchObject({ status: "Completed", forClient: true, completedByClient: true });
  // Again: unchanged.
  const again = await call(apiURL(), bride, "POST", `/api/v1/projects/${projectId}/milestones/${stepId}/complete`);
  expect(again.status).toBe(200);
  expect(again.json.data.updatedAt).toBe(done.json.data.updatedAt);

  await page.reload();
  await expect(page.getByRole("row").filter({ hasText: "Kirim daftar tamu" }).getByText("Diselesaikan pasangan")).toBeVisible();
  await close();
});
