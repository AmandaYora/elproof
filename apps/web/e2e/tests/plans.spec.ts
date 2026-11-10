import type { Page } from "@playwright/test";
import { expect, visit, test } from "../lib/fixtures";
import { open } from "../lib/suite";
import { apiURL } from "../lib/config";

// T-20261110-16 (ADR-0040): every plan belongs to exactly one App. The Paket page groups and filters by App, an App's
// detail lists the plans it owns and adds one for itself, and the manual-activation picker offers ElProof's plans only.
// These tests WRITE data (an App, plans), so they run in the "mutating" project.

const API = (path: string) => `${apiURL()}/api/v1${path}`;

async function auth(page: Page) {
  const session = await page.evaluate(() => JSON.parse(localStorage.getItem("elproof.session") ?? "{}") as { accessToken: string });
  return { Authorization: `Bearer ${session.accessToken}` };
}

/** Registers an external App and gives it one plan, through the API. */
async function appWithPlan(page: Page, tag: string) {
  const headers = await auth(page);
  const app = await page.request.post(API("/payment/apps"), { headers, data: { name: `Aplikasi ${tag}`, callbackUrl: "https://example.com/hook" } });
  expect(app.status()).toBe(201);
  const appId = (await app.json()).data.appId as string;
  const plan = await page.request.post(API("/plans"), {
    headers,
    data: { appId, name: `Paket ${tag}`, durationMonths: 12, price: 1_500_000, features: ["Fitur khusus"] },
  });
  expect(plan.status()).toBe(201);
  return { appId, appName: `Aplikasi ${tag}`, planName: `Paket ${tag}` };
}

test("Paket: a new plan names its App, and the page filters by App on the server", async ({ browser }, info) => {
  const { page, close } = await open(browser, info, "admin");
  await visit(page, "/platform/paket");
  const tag = `Saring ${Date.now()}`;
  const { appId, appName, planName } = await appWithPlan(page, tag);

  // Creating without an App is refused on that field; choosing the App saves it into the App's own section.
  await visit(page, "/platform/paket");
  await page.getByRole("button", { name: "Tambah Paket", exact: true }).click();
  const dialog = page.getByRole("dialog", { name: "Tambah Paket Langganan" });
  await expect(dialog.getByLabel("Nama Paket")).toBeFocused();
  await dialog.getByLabel("Nama Paket").fill(`Paket Kedua ${tag}`);
  await dialog.getByPlaceholder("cth. Backup data otomatis setiap hari").fill("Dukungan prioritas");
  await dialog.getByRole("button", { name: "Simpan paket" }).click();
  await expect(dialog.getByText("Pilih aplikasi pemilik paket").first()).toBeVisible();
  await dialog.getByRole("button", { name: /^Aplikasi:/ }).click();
  await page.getByRole("option", { name: appName }).click();
  await dialog.getByRole("button", { name: "Simpan paket" }).click();
  await expect(dialog).toBeHidden();
  const section = page.locator(`[data-app="${appId}"]`);
  await expect(section.getByRole("heading", { name: appName })).toBeVisible();
  await expect(section.getByRole("row").filter({ hasText: `Paket Kedua ${tag}` })).toHaveCount(1);
  await expect(page.locator(`[data-app="platform-billing"]`).getByRole("heading", { name: "ElProof (tenant WO)" })).toBeVisible();

  // Filter: the request asks the server for that App only, and only its section is shown.
  const asked = page.waitForRequest((r) => r.url().includes("/api/v1/plans") && new URL(r.url()).searchParams.get("appId") === appId);
  await page.getByRole("button", { name: /^Saring paket per aplikasi:/ }).click();
  await page.getByRole("option", { name: appName }).click();
  await asked;
  await expect(page).toHaveURL(new RegExp(`aplikasi=${appId}`));
  await expect(page.locator("[data-app]")).toHaveCount(1);
  await expect(section.getByRole("row").filter({ hasText: planName })).toHaveCount(1);
  await expect(page.getByText("Paket 1 Tahun")).toHaveCount(0);
  await close();
});

test("Manajemen Aplikasi › Detail › Paket lists the App's own plans and adds one for it", async ({ browser }, info) => {
  const { page, close } = await open(browser, info, "admin");
  await visit(page, "/platform/aplikasi");
  const tag = `Detail ${Date.now()}`;
  const { appId, appName, planName } = await appWithPlan(page, tag);
  await visit(page, "/platform/aplikasi");

  await page.getByRole("row").filter({ hasText: appName }).getByRole("button", { name: "Detail" }).click();
  const detail = page.getByRole("dialog", { name: appName });
  await detail.getByRole("button", { name: "Paket", exact: true }).click();
  await expect(detail.getByRole("row").filter({ hasText: planName })).toHaveCount(1);
  await expect(detail.getByRole("checkbox")).toHaveCount(0); // no more sharing checkboxes
  await expect(detail.getByText("Paket 1 Tahun")).toHaveCount(0); // ElProof's plan is not this App's

  await detail.getByRole("button", { name: "Tambah paket" }).click();
  const form = page.getByRole("dialog", { name: "Tambah Paket Langganan" });
  await expect(form.getByText(appName, { exact: true })).toBeVisible(); // the App is fixed, not a choice
  await expect(form.getByRole("button", { name: /^Aplikasi:/ })).toHaveCount(0);
  await expect(form.getByLabel("Nama Paket")).toBeFocused();
  await form.getByLabel("Nama Paket").fill(`Paket Tambahan ${tag}`);
  await form.getByPlaceholder("cth. Backup data otomatis setiap hari").fill("Dukungan prioritas");
  await form.getByRole("button", { name: "Simpan paket" }).click();
  await expect(form).toBeHidden();
  await expect(detail).toBeVisible(); // the detail stays open under the form
  await expect(detail.getByRole("row").filter({ hasText: `Paket Tambahan ${tag}` })).toHaveCount(1);

  const headers = await auth(page);
  const saved = (await (await page.request.get(API(`/plans?all=true&appId=${appId}`), { headers })).json()).data as Array<{ name: string; appId: string }>;
  expect(saved.map((p) => p.name).sort()).toEqual([planName, `Paket Tambahan ${tag}`].sort());
  await close();
});

test("Aktifkan Langganan offers ElProof's plans only, and a WO sees only ElProof's catalog", async ({ browser }, info) => {
  const { page, close } = await open(browser, info, "admin");
  await visit(page, "/platform/tenant");
  const { planName } = await appWithPlan(page, `Picker ${Date.now()}`);
  await visit(page, "/platform/tenant");

  await page.getByRole("button", { name: "Aktifkan Langganan" }).filter({ visible: true }).first().click();
  const dialog = page.getByRole("dialog", { name: "Aktifkan Langganan Manual" });
  await dialog.locator('[aria-haspopup="listbox"]').first().click();
  await expect(page.getByRole("option", { name: /Paket 1 Tahun/ })).toBeVisible();
  await expect(page.getByRole("option", { name: new RegExp(planName) })).toHaveCount(0);
  await page.keyboard.press("Escape");
  await close();

  const wo = await open(browser, info, "paying");
  await visit(wo.page, "/dashboard");
  const plans = (await (await wo.page.request.get(API("/plans?all=true"), { headers: await auth(wo.page) })).json()).data as Array<{ name: string; appId: string }>;
  expect(plans.length).toBeGreaterThan(0);
  expect(plans.every((p) => p.appId === "platform-billing"), JSON.stringify(plans)).toBe(true);
  expect(plans.map((p) => p.name)).not.toContain(planName);
  await wo.close();
});
