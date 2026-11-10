import type { Locator, Page } from "@playwright/test";
import { expect, state, visit, test } from "../lib/fixtures";
import { open } from "../lib/suite";
import { apiURL, config } from "../lib/config";
import { sql } from "../lib/sql";

// T-20261110-17 (ADR-0041): vouchers. The Platform Console creates and switches them off; a WO Owner applies one in the
// "Bayar Sekarang" dialog and sees the price before and after it, or why it was refused. No payment gateway runs here, so
// nothing is paid. These tests WRITE data (vouchers, a plan), so they run in the "mutating" project.

const API = (path: string) => `${apiURL()}/api/v1`.concat(path);
const ELPROOF = "platform-billing";
const PLAN_NAME = "Paket 1 Tahun"; // ElProof's seeded plan, Rp2.000.000 (migration 000045)

async function auth(page: Page) {
  const session = await page.evaluate(() => JSON.parse(localStorage.getItem("elproof.session") ?? "{}") as { accessToken: string });
  return { Authorization: `Bearer ${session.accessToken}` };
}

/** A code no other test (or run) has used: A–Z, 0–9, at most 32 characters. */
function uniqueCode(prefix: string) {
  return `${prefix}${Date.now().toString(36)}${Math.random().toString(36).slice(2, 5)}`.toUpperCase();
}

/** A calendar day in WIB, the zone the API counts voucher days in, `offset` days from today. */
function wibDay(offset: number) {
  return new Date(Date.now() + 7 * 3_600_000 + offset * 86_400_000).toISOString().slice(0, 10);
}

interface VoucherInput {
  code: string;
  planId?: number | null;
  discountType?: "percent" | "amount";
  discountValue?: number;
  maxDiscount?: number | null;
  validFrom?: string;
  validUntil?: string;
  quota?: number | null;
}

/** Creates an ElProof voucher through the API, as the signed-in admin. */
async function createVoucher(page: Page, input: VoucherInput) {
  const res = await page.request.post(API("/vouchers"), {
    headers: await auth(page),
    data: {
      appId: ELPROOF, planId: null, discountType: "percent", discountValue: 10, maxDiscount: null,
      validFrom: "", validUntil: "", quota: null, oncePerCustomer: false, ...input,
    },
  });
  expect(res.status(), await res.text()).toBe(201);
  return (await res.json()).data as { id: number; code: string };
}

async function elproofPlanId(page: Page) {
  const res = await page.request.get(API(`/plans?all=true&appId=${ELPROOF}`), { headers: await auth(page) });
  const plans = (await res.json()).data as Array<{ id: number; name: string }>;
  const plan = plans.find((p) => p.name === PLAN_NAME);
  expect(plan, JSON.stringify(plans)).toBeTruthy();
  return plan!.id;
}

/** The Owner's "Bayar Sekarang" dialog for ElProof's seeded plan, with its voucher field focused. */
async function openPayDialog(page: Page) {
  await visit(page, "/langganan");
  await page
    .getByText(PLAN_NAME, { exact: true })
    .locator("xpath=ancestor::div[.//button][1]")
    .getByRole("button", { name: /Pilih Paket Ini|Perpanjang Paket Ini/ })
    .click();
  const dialog = page.getByRole("dialog");
  const field = dialog.getByLabel("Kode voucher (opsional)");
  await expect(field).toBeFocused(); // the dialog's first field; typing before focus lands would be lost
  return { dialog, field };
}

/** The price block of the pay dialog: label → amount. */
function priceLine(dialog: Locator, label: string) {
  return dialog.getByText(label, { exact: true }).locator("xpath=ancestor::div[1]");
}

test("Voucher: admin creates an ElProof 10% voucher in the form, and an empty code is refused on its field", async ({ browser }, info) => {
  const { page, close } = await open(browser, info, "admin");
  await visit(page, "/platform/voucher");
  const code = uniqueCode("E2EA");

  await page.getByRole("button", { name: "Tambah voucher" }).first().click();
  const dialog = page.getByRole("dialog", { name: "Tambah Voucher" });
  const codeInput = dialog.getByRole("textbox", { name: /^Kode/ });
  await expect(codeInput).toBeFocused();

  // Nothing filled in: the form says what is missing, on each field.
  await dialog.getByRole("button", { name: "Simpan voucher" }).click();
  await expect(dialog.getByText("Isi kode voucher").first()).toBeVisible();
  await expect(dialog.getByText("Pilih aplikasi voucher ini").first()).toBeVisible();
  await expect(dialog).toBeVisible();

  // Typed in lower case, kept in upper case.
  await codeInput.fill(code.toLowerCase());
  await expect(codeInput).toHaveValue(code);
  await dialog.getByRole("button", { name: /^Aplikasi:/ }).click();
  await page.getByRole("option", { name: "ElProof (tenant WO)" }).click();
  await dialog.getByRole("spinbutton", { name: /^Nilai potongan/ }).fill("10");
  await dialog.getByRole("textbox", { name: /^Batas maks/ }).fill("50000");
  await dialog.getByRole("button", { name: "Simpan voucher" }).click();
  await expect(dialog).toBeHidden();

  const row = page.getByRole("row").filter({ hasText: code });
  await expect(row).toHaveCount(1);
  for (const text of ["ElProof (tenant WO)", "Semua paket", "10% · maks Rp50.000", "Tanpa batas", "0 / tanpa batas", "Aktif"]) {
    await expect(row).toContainText(text);
  }

  const saved = (await (await page.request.get(API(`/vouchers?search=${code}`), { headers: await auth(page) })).json()).data as Array<Record<string, unknown>>;
  expect(saved).toHaveLength(1);
  expect(saved[0]).toMatchObject({ code, appId: ELPROOF, planId: null, discountType: "percent", discountValue: 10, maxDiscount: 50000, quota: null, isActive: true });
  await close();
});

test("Voucher: a WO Owner applies a code in Bayar Sekarang and sees the price before and after it", async ({ browser }, info) => {
  const admin = await open(browser, info, "admin");
  await visit(admin.page, "/platform/voucher");
  const { code } = await createVoucher(admin.page, { code: uniqueCode("E2EB"), discountValue: 10 });
  await admin.close();

  const { page, close } = await open(browser, info, "owner");
  const { dialog, field } = await openPayDialog(page);
  await expect(priceLine(dialog, "Biaya")).toContainText("Rp2.000.000");

  await field.fill(code.toLowerCase());
  await dialog.getByRole("button", { name: "Pakai" }).click();
  await expect(priceLine(dialog, "Harga awal")).toContainText("Rp2.000.000");
  await expect(dialog.getByText(code, { exact: true })).toBeVisible();
  await expect(dialog.getByText("−Rp200.000")).toBeVisible();
  await expect(priceLine(dialog, "Harga akhir")).toContainText("Rp1.800.000");
  await expect(field).toHaveValue(code);
  await expect(dialog.getByRole("button", { name: "Hapus" })).toBeVisible();

  // Editing the code drops the voucher: it has to be applied again before it counts.
  await field.press("End");
  await field.press("X");
  await expect(dialog.getByText("Harga akhir", { exact: true })).toHaveCount(0);
  await expect(priceLine(dialog, "Biaya")).toContainText("Rp2.000.000");
  await expect(dialog.getByRole("button", { name: "Pakai" })).toBeVisible();

  // Applied again, then removed with "Hapus": back to the normal price, field empty.
  await field.fill(code);
  await dialog.getByRole("button", { name: "Pakai" }).click();
  await expect(priceLine(dialog, "Harga akhir")).toContainText("Rp1.800.000");
  await dialog.getByRole("button", { name: "Hapus" }).click();
  await expect(field).toHaveValue("");
  await expect(priceLine(dialog, "Biaya")).toContainText("Rp2.000.000");
  await close();
});

test("Voucher: a refused code says why in Indonesian, under the field, and the price stays normal", async ({ browser }, info) => {
  const admin = await open(browser, info, "admin");
  await visit(admin.page, "/platform/voucher");
  const headers = await auth(admin.page);
  const planId = await elproofPlanId(admin.page);

  const expired = await createVoucher(admin.page, { code: uniqueCode("E2EEXP"), validUntil: wibDay(-1) });
  const inactive = await createVoucher(admin.page, { code: uniqueCode("E2EOFF") });
  const off = await admin.page.request.patch(API(`/vouchers/${inactive.id}`), { headers, data: { isActive: false } });
  expect(off.status()).toBe(200);

  // A voucher for another ElProof plan. That plan is switched off again right away, so no WO ever sees it.
  const otherName = `Paket Lain ${Date.now()}`;
  const otherPlan = await admin.page.request.post(API("/plans"), {
    headers,
    data: { appId: ELPROOF, name: otherName, durationMonths: 6, price: 1_200_000, features: ["Uji voucher"] },
  });
  expect(otherPlan.status()).toBe(201);
  const otherPlanId = (await otherPlan.json()).data.id as number;
  const wrongPlan = await createVoucher(admin.page, { code: uniqueCode("E2EPLN"), planId: otherPlanId });
  expect((await admin.page.request.post(API(`/plans/${otherPlanId}/toggle-active`), { headers })).status()).toBe(200);

  // Quota 1, already used by another workspace (written straight into billing's ledger: paying needs a gateway).
  const exhausted = await createVoucher(admin.page, { code: uniqueCode("E2EFUL"), quota: 1 });
  const orderRef = `E2E-${exhausted.code}`;
  sql(
    `INSERT INTO subscription_voucher_redemptions (voucher_id, voucher_code, order_ref, customer_type, customer_ref, plan_id, original_amount, discount_amount, final_amount, status, resolved_at)
     VALUES (${exhausted.id}, '${exhausted.code}', '${orderRef}', 'tenant', '${state().ids.tenantBerbayar}', ${planId}, 2000000, 200000, 1800000, 'used', NOW(6))`,
    config.dbName
  );

  const { page, close } = await open(browser, info, "owner");
  const { dialog, field } = await openPayDialog(page);
  const cases: Array<[string, RegExp]> = [
    [uniqueCode("E2ENONE"), /Kode voucher tidak ditemukan/],
    [expired.code, /Voucher ini sudah kedaluwarsa/],
    [inactive.code, /Voucher ini sudah tidak berlaku/],
    [wrongPlan.code, new RegExp(`Voucher ini hanya berlaku untuk paket ${otherName}`)],
    [exhausted.code, /Kuota voucher ini sudah habis/],
  ];
  for (const [code, message] of cases) {
    await field.fill(code);
    await dialog.getByRole("button", { name: "Pakai" }).click();
    const alert = dialog.getByRole("alert").filter({ hasText: message });
    await expect(alert, code).toBeVisible();
    await expect(field).toHaveAttribute("aria-invalid", "true");
    await expect(field).toHaveAttribute("aria-describedby", (await alert.getAttribute("id")) ?? "missing");
    await expect(dialog.getByText("Harga akhir", { exact: true })).toHaveCount(0);
    await expect(priceLine(dialog, "Biaya")).toContainText("Rp2.000.000");
  }
  await close();

  // The console shows the quota as used up, and who used it.
  const consoleAdmin = await open(browser, info, "admin");
  await visit(consoleAdmin.page, `/platform/voucher?q=${exhausted.code}`);
  const row = consoleAdmin.page.getByRole("row").filter({ hasText: exhausted.code });
  await expect(row).toContainText("1 / 1");
  await expect(row).toContainText("Habis");
  await row.getByRole("button", { name: `Pemakaian ${exhausted.code}` }).click();
  const usage = consoleAdmin.page.getByRole("dialog", { name: `Pemakaian voucher ${exhausted.code}` });
  const use = usage.getByRole("row").filter({ hasText: orderRef });
  await expect(use).toContainText("WO Berbayar");
  await expect(use).toContainText("Terpakai");
  await expect(use).toContainText("Rp1.800.000");
  await consoleAdmin.close();
});

test("Voucher: admin switches a voucher off after confirming, and it reads Nonaktif", async ({ browser }, info) => {
  const { page, close } = await open(browser, info, "admin");
  await visit(page, "/platform/voucher");
  const { id, code } = await createVoucher(page, { code: uniqueCode("E2ED") });
  await visit(page, `/platform/voucher?q=${code}`);

  const row = page.getByRole("row").filter({ hasText: code });
  await expect(row).toContainText("Aktif");
  await row.getByRole("button", { name: `Nonaktifkan ${code}` }).click();
  const confirm = page.getByRole("alertdialog", { name: `Nonaktifkan voucher ${code}?` });
  await expect(confirm.getByRole("button", { name: "Batal" })).toBeFocused();
  await confirm.getByRole("button", { name: "Nonaktifkan voucher" }).click();
  await expect(confirm).toBeHidden();
  await expect(row).toContainText("Nonaktif");
  await expect(row.getByRole("button", { name: `Aktifkan ${code}` })).toBeVisible();

  const saved = (await (await page.request.get(API(`/vouchers/${id}`), { headers: await auth(page) })).json()).data as { isActive: boolean; state: string };
  expect(saved).toMatchObject({ isActive: false, state: "inactive" });
  await close();
});
