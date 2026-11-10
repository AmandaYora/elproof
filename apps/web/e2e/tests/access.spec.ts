import type { Browser, Page, TestInfo } from "@playwright/test";
import { expect, prepareContext, test, visit } from "../lib/fixtures";
import { call, must } from "../lib/api";
import { apiURL, config, webURL, type Session } from "../lib/config";

// T-20261110-12: what a role's screen offers matches what the API lets it do (ADR-0026). Each test signs up its own
// workspace (no golden data is touched) and signs in as staff of the role under test, so it WRITES data: "mutating".

const google = () => `http://127.0.0.1:${config.googlePort}`;
let seq = 0;

async function googleCredential(email: string, name: string): Promise<string> {
  const res = await fetch(`${google()}/token?email=${encodeURIComponent(email)}&name=${encodeURIComponent(name)}&sub=g-sub-${email}`);
  return res.text();
}

interface Workspace {
  owner: Session;
  key: string;
  roles: { id: number; name: string; permissions: string[]; projectScope: string }[];
}

async function workspace(): Promise<Workspace> {
  // Two sign-ups at the same instant can deadlock in the API (MySQL 1213, a 500): the workers sign up at once, so a
  // refused sign-up is tried again under a new name. Reported as its own follow-up (T-20261110-12).
  for (let attempt = 1; ; attempt++) {
    const key = `akses${Date.now().toString(36)}${Math.random().toString(36).slice(2, 7)}${(seq += 1)}`;
    const cred = await googleCredential(`${key}@akses.test`, "Pemilik Akses");
    const signup = await call(apiURL(), null, "POST", "/api/v1/public/signup/workspace", { credential: cred, businessName: `WO ${key}`, acceptTerms: true }, {
      // Its own address, so the per-address sign-up limit (10 an hour) is never what a run measures.
      "X-Real-IP": `10.99.${Math.floor(Math.random() * 250)}.${Math.floor(Math.random() * 250) + 1}`,
    });
    if (signup.status >= 500 && attempt < 4) {
      await new Promise((r) => setTimeout(r, 200 * attempt));
      continue;
    }
    must("signup", signup);
    const owner = must("owner login", await call(apiURL(), null, "POST", "/api/v1/auth/google", { credential: cred })).json.data as Session;
    const roles = must("roles", await call(apiURL(), owner.accessToken, "GET", "/api/v1/access/roles?kind=staff")).json.data as Workspace["roles"];
    return { owner, key, roles };
  }
}

const roleId = (ws: Workspace, name: string) => {
  const r = ws.roles.find((x) => x.name === name);
  if (!r) throw new Error(`no role ${name}`);
  return r.id;
};

/** Invites a staff member with this role and accepts the invitation with Google: their session. */
async function staff(ws: Workspace, name: string, role: number): Promise<Session> {
  const email = `${name.toLowerCase().replace(/\s+/g, ".")}.${ws.key}@akses.test`;
  const made = must(`staff ${name}`, await call(apiURL(), ws.owner.accessToken, "POST", "/api/v1/staff", { name, title: name, roleId: role, email, phone: "081200000000" }));
  const token = made.json.data.invitation.token as string;
  const cred = await googleCredential(email, name);
  return must(`accept ${name}`, await call(apiURL(), null, "POST", `/api/v1/auth/links/${token}/google`, { credential: cred })).json.data as Session;
}

async function as(browser: Browser, info: TestInfo, session: Session): Promise<{ page: Page; close: () => Promise<void> }> {
  const context = await browser.newContext({
    ...info.project.use,
    storageState: { cookies: [], origins: [{ origin: webURL(), localStorage: [{ name: "elproof.session", value: JSON.stringify(session) }] }] },
  });
  const page = await context.newPage();
  await prepareContext(page);
  return { page, close: () => context.close() };
}

test.describe("role screens follow the API", () => {
  test("Keuangan (vendor.view, no vendor.manage) reads the vendor and venue lists without write buttons", async ({ browser }, info) => {
    const ws = await workspace();
    must("vendor", await call(apiURL(), ws.owner.accessToken, "POST", "/api/v1/venues", {
      name: "Gedung Uji Akses", picName: "Pak Uji", phonePic: "081300000000", phoneVenue: "", email: "", address: "Jl. Uji", city: "Kota Bandung",
      rentalPrice: 1, charge: 0, capacity: 100, facilities: "", socialMedia: "", notes: "",
    }));
    const finance = await staff(ws, "Uji Keuangan", roleId(ws, "Keuangan"));
    const { page, close } = await as(browser, info, finance);
    for (const route of ["/vendors", "/venues"]) {
      await visit(page, route);
      await expect(page.getByRole("button", { name: "Ekspor ke Excel" })).toBeVisible();
      await expect(page.getByRole("button", { name: /^Tambah (vendor|venue)$/i })).toHaveCount(0);
      await expect(page.getByRole("button", { name: "Impor bulk" })).toHaveCount(0);
    }
    await expect(page.getByText("Gedung Uji Akses").first()).toBeVisible();
    await expect(page.getByRole("button", { name: /^Ubah venue$/i })).toHaveCount(0);
    await expect(page.getByRole("button", { name: "Nonaktifkan" })).toHaveCount(0);
    await close();
  });

  test("Keuangan may add a document but not show it on the Client Portal (the API refuses that too)", async ({ browser }, info) => {
    const ws = await workspace();
    const project = must("project", await call(apiURL(), ws.owner.accessToken, "POST", "/api/v1/projects", {
      name: "Uji Dokumen", brideName: "Lina", groomName: "Reza", eventDate: "2027-07-01", venue: "Gedung Uji", prepStartDate: "2026-11-09",
      packageName: "Paket Uji", contractValue: 10_000_000, status: "Draft", picStaffId: Number(ws.owner.principalId), picSalesStaffId: 0, description: "",
    })).json.data.id as number;
    const finance = await staff(ws, "Uji Dokumen Keuangan", roleId(ws, "Keuangan"));
    const refused = await call(apiURL(), finance.accessToken, "POST", `/api/v1/projects/${project}/evidence`, {
      name: "Kontrak", type: "Contract", fileName: "k.pdf", mimeType: "application/pdf", base64Data: "JVBERi0xLjQK", documentDate: "2026-11-09",
      description: "", relatedKind: "general", relatedId: 0, isClientVisible: true,
    });
    expect(refused.status).toBe(403);
    const { page, close } = await as(browser, info, finance);
    await visit(page, `/projects/${project}/dokumen`);
    await page.getByRole("button", { name: "Tambah dokumen" }).click();
    const dialog = page.getByRole("dialog", { name: "Tambah dokumen" });
    await dialog.getByLabel("Konteks").click();
    await page.getByRole("option", { name: "Umum (tidak terkait spesifik)" }).click();
    await expect(dialog.getByLabel("Konteks")).toContainText("Umum");
    await expect(dialog.getByText("Tampilkan dokumen ini ke portal klien")).toHaveCount(0);
    await close();
  });

  test("a team manager below Admin cannot open Admin's row or role for editing", async ({ browser }, info) => {
    const ws = await workspace();
    const lead = must("custom role", await call(apiURL(), ws.owner.accessToken, "POST", "/api/v1/access/roles", {
      kind: "staff", name: "Kepala Tim", description: "", projectScope: "assigned", permissions: ["team.manage", "role.manage", "project.edit"],
    })).json.data.id as number;
    await staff(ws, "Uji Admin", roleId(ws, "Admin"));
    const manager = await staff(ws, "Uji Kepala", lead);
    const { page, close } = await as(browser, info, manager);

    await visit(page, "/pengguna");
    const adminRow = page.getByRole("row", { name: /Uji Admin/ });
    await expect(adminRow.getByText("Hanya dapat diubah oleh Owner")).toBeVisible();
    await expect(adminRow.getByRole("button", { name: "Ubah anggota tim" })).toHaveCount(0);

    await visit(page, "/peran-akses");
    await page.getByRole("button", { name: "Ubah", exact: true }).nth(roleIndex(ws, "Admin")).click();
    const drawer = page.getByRole("dialog");
    await expect(drawer.getByText("Peran ini memberi akses yang tidak Anda miliki. Hanya Owner yang dapat mengubahnya.")).toBeVisible();
    await close();
  });

  test("the palette of a Wedding Planner searches projects only, never the vendor or client lists it may not open", async ({ browser }, info) => {
    const ws = await workspace();
    const planner = await staff(ws, "Uji Planner", roleId(ws, "Wedding Planner"));
    const { page, close } = await as(browser, info, planner);
    await visit(page, "/dashboard");
    const asked: string[] = [];
    page.on("request", (r) => {
      const url = new URL(r.url());
      if (url.searchParams.get("search") === "Citra") asked.push(url.pathname);
    });
    await page.keyboard.press("Control+k");
    const dialog = page.getByRole("dialog", { name: "Pencarian" });
    await expect(dialog).toBeVisible();
    await dialog.getByRole("combobox").fill("Citra");
    await expect.poll(() => asked.includes("/api/v1/projects"), { timeout: 10_000 }).toBe(true);
    await page.waitForTimeout(500);
    asked.splice(asked.indexOf("/api/v1/projects"), 1);
    expect(asked).toEqual([]);
    await close();
  });
});

/** The editable roles on Peran & Akses are listed in the API's order, the Owner's first (shown as "Lihat"). */
function roleIndex(ws: Workspace, name: string) {
  return ws.roles.filter((r) => r.name !== "Owner").findIndex((r) => r.name === name);
}
