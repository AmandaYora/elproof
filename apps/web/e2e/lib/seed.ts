import fs from "node:fs";
import path from "node:path";
import { AUTH, AUTH_DIR, BASE_DAY, type Session, type State } from "./config";
import { call, must } from "./api";
import { sql } from "./sql";
import { totp } from "./totp";

const PASSWORD_KLIEN = "Kata-sandi-klien-123";
export const ADMIN_LOGIN = { username: "superadmin", password: "superadmin" };

/** A date `offset` days from the fixed base day, never from the real clock: seeds must not drift. */
function day(offset: number): string {
  const d = new Date(`${BASE_DAY}T00:00:00Z`);
  d.setUTCDate(d.getUTCDate() + offset);
  return d.toISOString().slice(0, 10);
}

interface Seeder {
  api: string;
  google: string;
  s3: boolean;
  db: string;
}

async function googleToken(s: Seeder, email: string, name: string, sub: string): Promise<string> {
  const res = await fetch(`${s.google}/token?email=${encodeURIComponent(email)}&name=${encodeURIComponent(name)}&sub=${sub}`);
  return res.text();
}

async function signUp(s: Seeder, key: string, business: string, owner: string) {
  const cred = await googleToken(s, `${key}@contoh.test`, owner, `g-sub-${key}`);
  must(`signup ${key}`, await call(s.api, null, "POST", "/api/v1/public/signup/workspace", { credential: cred, businessName: business, acceptTerms: true }));
  const login = must(`login ${key}`, await call(s.api, null, "POST", "/api/v1/auth/google", { credential: cred }));
  const session = login.json.data as Session;
  return { session, token: session.accessToken, staffId: Number(session.principalId), tenantId: Number(session.tenantId) };
}

function writeAuth(file: string, webURL: string, session: Session | null) {
  fs.mkdirSync(AUTH_DIR, { recursive: true });
  const origins = session ? [{ origin: webURL, localStorage: [{ name: "elproof.session", value: JSON.stringify(session) }] }] : [];
  fs.writeFileSync(file, JSON.stringify({ cookies: [], origins }, null, 2));
}

async function platformAdminSession(s: Seeder): Promise<Session> {
  const login = must("admin login", await call(s.api, null, "POST", "/api/v1/auth/login", ADMIN_LOGIN));
  const first = login.json.data as Session;
  const setup = must("mfa setup", await call(s.api, first.accessToken, "POST", "/api/v1/auth/mfa/setup"));
  const secret = setup.json.data.secret as string;
  const enable = must("mfa enable", await call(s.api, first.accessToken, "POST", "/api/v1/auth/mfa/enable", { code: totp(secret) }));
  return enable.json.data as Session;
}

const pdf = Buffer.from("%PDF-1.4\n1 0 obj<<>>endobj\ntrailer<<>>\n%%EOF").toString("base64");

/** Builds the deterministic workspaces and writes the browser sessions. Returns ids the specs need. */
export async function seed(s: Seeder, webURL: string): Promise<State["ids"]> {
  // ---- Platform Console: the super admin the `seed` command created, with two-factor on ----
  const admin = await platformAdminSession(s);

  // ---- WO Contoh: the rich workspace ----
  const owner = await signUp(s, "owner", "WO Contoh", "Rina Marlina");
  const t = owner.token;
  const roles = must("staff roles", await call(s.api, t, "GET", "/api/v1/access/roles?kind=staff")).json.data as { id: number; name: string }[];
  const roleOf = (name: string) => (roles.find((r) => r.name === name) ?? roles.find((r) => r.name !== "Owner") ?? roles[0]).id;
  for (const [name, title, role, phone] of [
    ["Bagas Pratama", "Koordinator lapangan", "Wedding Planner", "081234500001"],
    ["Sinta Wulandari", "Admin keuangan", "Admin", "081234500002"],
  ] as const) {
    must(`staff ${name}`, await call(s.api, t, "POST", "/api/v1/staff", { name, title, roleId: roleOf(role), email: `${name.split(" ")[0].toLowerCase()}@contoh.test`, phone }));
  }

  const cats = must("categories", await call(s.api, t, "GET", "/api/v1/vendor-categories?all=true")).json.data as { id: number; name: string }[];
  const cat = (i: number) => cats[i % cats.length].id;
  const vendorIds: number[] = [];
  for (const [i, name] of ["Katering Rasa Nusantara", "Dekorasi Bunga Kasih", "Foto Cahaya Studio"].entries()) {
    const r = must(`vendor ${name}`, await call(s.api, t, "POST", "/api/v1/vendors", {
      name, categoryId: cat(i), picName: ["Bu Wati", "Mas Dimas", "Kak Lia"][i], phone: `08121111000${i}`, email: "", socialMedia: "@" + name.split(" ")[0].toLowerCase(),
      city: "Kota Bandung", address: "Jl. Contoh No. " + (i + 1), priceAkad: 5_000_000 + i * 1_000_000, priceAkadResepsi: 12_000_000, priceResepsi: 9_000_000, notes: "",
    }));
    vendorIds.push(r.json.data.id);
  }
  const venueIds: number[] = [];
  for (const [i, name] of ["Gedung Graha Sabuga", "Hotel Savoy Homann"].entries()) {
    const r = must(`venue ${name}`, await call(s.api, t, "POST", "/api/v1/venues", {
      name, picName: "Pak Hendra", phonePic: `08133330000${i}`, phoneVenue: "0221234567", email: "", address: "Jl. Gedung No. " + (i + 1), city: "Kota Bandung",
      rentalPrice: 25_000_000 + i * 10_000_000, charge: 1_500_000, capacity: 300 + i * 100, facilities: "AC, parkir luas", socialMedia: "@" + name.split(" ")[1].toLowerCase(), notes: "",
    }));
    venueIds.push(r.json.data.id);
  }

  async function project(name: string, bride: string, groom: string, event: string, contract: number, status: string, pic = owner.staffId) {
    const r = await call(s.api, t, "POST", "/api/v1/projects", {
      name, brideName: bride, groomName: groom, eventDate: event, venue: "Gedung Graha Sabuga", prepStartDate: day(-30), packageName: "Paket Premium",
      contractValue: contract, status, picStaffId: pic, description: "Resepsi adat Sunda",
    });
    return must(`project ${name}`, r).json.data.id as number;
  }
  const p1 = await project("Citra & Bima", "Citra Lestari", "Bima Pratama", "2027-03-14", 150_000_000, "Preparation");
  const p2 = await project("Dewi & Arga", "Dewi Anggraini", "Arga Wijaya", "2026-09-20", 90_000_000, "Draft");
  const p3 = await project("Sari & Doni", "Sari Puspita", "Doni Saputra", "2027-06-20", 120_000_000, "Draft");

  // Projects are created as Draft whatever the body says; the lifecycle moves through explicit actions.
  const step = async (id: number, action: string) => must(`${action} ${id}`, await call(s.api, t, "POST", `/api/v1/projects/${id}/transition`, { action }));
  await step(p1, "start_preparation");
  for (const a of ["start_preparation", "mark_ready", "complete"]) await step(p2, a);

  // P1: timeline progress, vendors, money, an issue
  const ms = must("milestones", await call(s.api, t, "GET", `/api/v1/projects/${p1}/milestones`)).json.data as { id: number; targetDate: string }[];
  for (const m of ms.slice(0, 2)) {
    must("milestone done", await call(s.api, t, "PATCH", `/api/v1/projects/${p1}/milestones/${m.id}`, { status: "Completed", targetDate: m.targetDate, completedDate: day(-20) }));
  }
  const engagement = must("engagement catering", await call(s.api, t, "POST", `/api/v1/projects/${p1}/vendors`, {
    vendorId: vendorIds[0], categoryId: cat(0), scope: "Katering 500 pax", contractValue: 45_000_000, pricingTier: "Akad & Resepsi", engagementStatus: "Booked",
    eventDate: "2027-03-14", dpAmount: 15_000_000, dueDate: day(40), picStaffId: owner.staffId, notes: "",
  })).json.data.id as number;
  must("engagement decor", await call(s.api, t, "POST", `/api/v1/projects/${p1}/vendors`, {
    vendorId: vendorIds[1], categoryId: cat(1), scope: "Dekorasi pelaminan", contractValue: 30_000_000, pricingTier: "Resepsi", engagementStatus: "Negotiation",
    eventDate: "2027-03-14", dpAmount: 0, dueDate: day(60), picStaffId: owner.staffId, notes: "",
  }));
  must("vendor milestone", await call(s.api, t, "POST", `/api/v1/projects/${p1}/vendors/${engagement}/milestones`, {
    name: "Tasting menu", description: "Cicip menu bersama klien", targetDate: day(20), picStaffId: owner.staffId,
  }));
  must("payment terms", await call(s.api, t, "PUT", `/api/v1/projects/${p1}/payment-terms`, {
    terms: [
      { label: "DP 30%", amount: 45_000_000, dueDate: day(-39) },
      { label: "Termin 2", amount: 40_000_000, dueDate: day(-15) }, // overdue
      { label: "Pelunasan", amount: 65_000_000, dueDate: "2027-02-14" },
    ],
  }));
  must("client payment", await call(s.api, t, "POST", `/api/v1/projects/${p1}/client-payments`, {
    type: "DP", amount: 45_000_000, paymentDate: day(-38), method: "Transfer", referenceNumber: "TRF-0001", notes: "Transfer BCA",
  }));
  must("vendor payment", await call(s.api, t, "POST", `/api/v1/projects/${p1}/payments`, {
    projectVendorId: engagement, type: "DP", amount: 15_000_000, paymentDate: day(-35), method: "Transfer", referenceNumber: "VND-0001", notes: "",
  }));
  must("issue", await call(s.api, t, "POST", `/api/v1/projects/${p1}/issues`, {
    projectVendorId: engagement, title: "Menu vegetarian belum dikonfirmasi", description: "Katering belum memastikan 30 porsi vegetarian.", impact: "Medium",
    foundDate: day(-3), resolutionPlan: "Telepon katering dan minta konfirmasi tertulis.", picStaffId: owner.staffId, targetResolutionDate: day(4),
  }));
  if (s.s3) {
    must("document", await call(s.api, t, "POST", `/api/v1/projects/${p1}/evidence`, {
      name: "Kontrak katering", type: "Contract", fileName: "kontrak.pdf", mimeType: "application/pdf", base64Data: pdf, documentDate: day(-40), description: "",
      relatedKind: "general", relatedId: 0, isClientVisible: true,
    }));
  }

  // P1's clients; the Bride gets a real login
  const clientRoles = must("client roles", await call(s.api, t, "GET", "/api/v1/access/roles?kind=client")).json.data as { id: number; name: string }[];
  const clientRole = (name: string) => (clientRoles.find((r) => r.name === name) ?? clientRoles[0]).id;
  const bride = must("client bride", await call(s.api, t, "POST", "/api/v1/clients", {
    projectId: p1, role: "Bride", relationNote: "", name: "Citra Lestari", phone: "081298760001", username: "citra.contoh", email: "", accessRoleId: clientRole("Pengantin"),
  }));
  const groom = must("client groom", await call(s.api, t, "POST", "/api/v1/clients", {
    projectId: p1, role: "Groom", relationNote: "", name: "Bima Pratama", phone: "081298760002", username: "bima.contoh", email: "", accessRoleId: clientRole("Pengantin"),
  }));
  void groom;
  const link = bride.json.data.accessLink.token as string;
  const clientLogin = must("client password", await call(s.api, null, "POST", `/api/v1/auth/links/${link}/password`, { password: PASSWORD_KLIEN }));
  const clientSession = clientLogin.json.data as Session;

  // ---- WO Berbayar: a paying tenant for the Platform Console ----
  const paying = await signUp(s, "berbayar", "WO Berbayar", "Hadi Susanto");

  // ---- WO Kosong: nothing in it, for the empty states ----
  const empty = await signUp(s, "kosong", "WO Kosong", "Maya Sari");

  // ---- Platform: an announcement and a subscription ----
  must("announcement", await call(s.api, admin.accessToken, "POST", "/api/v1/console/announcements", {
    title: "Pemeliharaan sistem Sabtu malam", body: "ElProof tidak dapat dibuka Sabtu pukul 01.00–03.00 WIB.", kind: "maintenance", audience: "all", audienceValue: "",
    startsAt: "2026-11-01T00:00:00+07:00", endsAt: "2027-12-31T00:00:00+07:00",
  }));
  must("data request", await call(s.api, admin.accessToken, "POST", `/api/v1/console/tenants/${owner.tenantId}/data-requests`, { kind: "export", requestedBy: "owner@contoh.test", note: "Minta salinan data usaha" }));

  // paid subscription for WO Berbayar (no API creates one without a payment gateway)
  sql(`UPDATE tenants SET plan_id = (SELECT MIN(id) FROM subscription_plans), subscription_status = 'active', subscription_expires_at = '2027-10-31 00:00:00' WHERE id = ${paying.tenantId}`, s.db);
  sql(`INSERT INTO subscription_transactions (tenant_id, type, amount, payment_method, payment_reference, status, created_at, paid_at) VALUES (${paying.tenantId}, 'new', 2000000, 'QRIS', 'ELP-CONTOH-0001', 'paid', NOW(), NOW())`, s.db);
  const txId = Number(sql("SELECT MAX(id) FROM subscription_transactions", s.db));

  normalizeTimes(s.db, owner.tenantId, paying.tenantId, empty.tenantId);

  writeAuth(AUTH.owner, webURL, owner.session);
  writeAuth(AUTH.client, webURL, clientSession);
  writeAuth(AUTH.admin, webURL, admin);
  writeAuth(AUTH.empty, webURL, empty.session);
  writeAuth(AUTH.anon, webURL, null);

  return {
    project: p1, projectDone: p2, projectDraft: p3, tenantContoh: owner.tenantId, tenantBerbayar: paying.tenantId, tenantKosong: empty.tenantId,
    vendor: vendorIds[0], engagement, transaction: txId,
  };
}

/**
 * Real timestamps (created_at, paid_at, audit times) differ on every run. Pin them to the fixed instant
 * 2026-11-09 10:00 WIB plus the row id in seconds, so ordering is kept and text on screen never drifts.
 * Server-computed relative text (trial days left, days overdue) still follows the real clock: see README.
 */
function normalizeTimes(db: string, ...tenants: number[]) {
  const cols = sql(
    `SELECT table_name, column_name FROM information_schema.columns WHERE table_schema = '${db}'
       AND data_type IN ('timestamp','datetime')
       AND column_name IN ('created_at','updated_at','uploaded_at','occurred_at','paid_at','joined_at','granted_at','started_at','terms_accepted_at','accepted_at','completed_at')
       AND table_name NOT IN ('refresh_tokens','auth_tokens','credentials','schema_migrations')`
  );
  for (const line of cols.split("\n").filter(Boolean)) {
    const [table, column] = line.split("\t");
    const hasId = sql(`SELECT COUNT(*) FROM information_schema.columns WHERE table_schema='${db}' AND table_name='${table}' AND column_name='id'`) === "1";
    const value = hasId ? `TIMESTAMP('2026-11-09 03:00:00') + INTERVAL id SECOND` : `TIMESTAMP('2026-11-09 03:00:00')`;
    sql(`UPDATE \`${table}\` SET \`${column}\` = ${value} WHERE \`${column}\` IS NOT NULL`, db);
  }
  const trialEnds = sql(
    `SELECT COUNT(*) FROM information_schema.columns WHERE table_schema='${db}' AND table_name='tenants' AND column_name='trial_ends_at'`
  );
  if (trialEnds === "1") {
    sql(`UPDATE tenants SET trial_ends_at = '2026-11-23 16:59:59' WHERE id IN (${tenants.join(",")})`, db);
  }
}

export function writeState(file: string, state: State) {
  fs.mkdirSync(path.dirname(file), { recursive: true });
  fs.writeFileSync(file, JSON.stringify(state, null, 2));
}
