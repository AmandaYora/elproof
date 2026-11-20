import fs from "node:fs";
import path from "node:path";
import { AUTH, AUTH_DIR, BASE_DAY, type Session, type State } from "./config";
import { call, must } from "./api";
import { sql } from "./sql";
import { totp } from "./totp";
import { sessionOf, storageStateFor } from "./session";

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
  const session = sessionOf(login);
  return { session, token: session.accessToken, staffId: Number(session.principalId), tenantId: Number(session.tenantId) };
}

function writeAuth(file: string, webURL: string, session: Session | null) {
  fs.mkdirSync(AUTH_DIR, { recursive: true });
  fs.writeFileSync(file, JSON.stringify(storageStateFor(webURL, session), null, 2));
}

async function platformAdminSession(s: Seeder): Promise<Session> {
  const login = must("admin login", await call(s.api, null, "POST", "/api/v1/auth/login", ADMIN_LOGIN));
  const first = sessionOf(login);
  const setup = must("mfa setup", await call(s.api, first.accessToken, "POST", "/api/v1/auth/mfa/setup"));
  const secret = setup.json.data.secret as string;
  const enable = must("mfa enable", await call(s.api, first.accessToken, "POST", "/api/v1/auth/mfa/enable", { code: totp(secret) }));
  return sessionOf(enable);
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
      priceAkadResepsi: 25_000_000 + i * 10_000_000, charge: 1_500_000, capacityStanding: 300 + i * 100, capacitySeated: 200 + i * 50, facilities: "AC, parkir luas", socialMedia: "@" + name.split(" ")[1].toLowerCase(), notes: "",
    }));
    venueIds.push(r.json.data.id);
  }
  // The catalog (paket WO Tahap 1): the six built-in event kinds came with the sign-up; one package uses them.
  const kinds = must("event kinds", await call(s.api, t, "GET", "/api/v1/event-kinds")).json.data as { id: number; name: string }[];
  const kind = (name: string) => kinds.find((k) => k.name === name)!.id;
  const packageId = must("package", await call(s.api, t, "POST", "/api/v1/packages", {
    name: "Paket Resepsi 300 tamu", price: 120_000_000, eventKindIds: [kind("Akad"), kind("Resepsi")],
    includedGuests: 300, extraGuestStep: 50, extraGuestStepPrice: 4_000_000, defaultVenueId: venueIds[0],
    components: [
      { description: "Katering buffet 300 porsi", vendorId: vendorIds[0], estimatedCost: 45_000_000 },
      // Only the vendor a project already uses: the others stay free to delete in the list tests.
      { description: "Dekorasi pelaminan dan gerbang", categoryId: cat(1), estimatedCost: 25_000_000 },
      { description: "Dokumentasi foto dan video", categoryId: cat(2), estimatedCost: 12_000_000 },
      { description: "Koordinator acara 6 orang", estimatedCost: 8_000_000 },
    ],
  })).json.data.id as number;

  /** One event without a venue: the dashboard and list projects only need a date. */
  const resepsi = (date: string) => ({ eventKindId: kind("Resepsi"), eventDate: date, startTime: "11:00", endTime: "14:00", venueId: null, layout: "Standing", sessions: [{ label: "Sesi 1", startTime: "11:00", endTime: "14:00", guests: 300 }] });
  /** A project and its Rincian: the contract value is the Rincian's total (the workspace is not PKP, so no PPN). */
  async function project(name: string, bride: string, groom: string, events: object[], lines: object[], status: string, pic = owner.staffId) {
    const r = await call(s.api, t, "POST", "/api/v1/projects", {
      name, brideName: bride, groomName: groom, prepStartDate: day(-30), status, picStaffId: pic, description: "Resepsi adat Sunda", events,
    });
    const created = must(`project ${name}`, r).json.data as { id: number; events: { id: number; kindName: string }[] };
    const current = must(`lines ${name}`, await call(s.api, t, "GET", `/api/v1/projects/${created.id}/lines`)).json.data as { linesVersion: string };
    must(`rincian ${name}`, await call(s.api, t, "PUT", `/api/v1/projects/${created.id}/lines`, { lines }, { "If-Match": `"${current.linesVersion}"` }));
    return created;
  }
  const paketPremium = (amount: number) => [{ kind: "Item", description: "Paket Premium", quantity: 1, unit: "paket", unitPrice: amount }];
  // P1 has two events: Akad in the morning before, Resepsi on Hari H (made the main event below).
  const p1Created = await project("Citra & Bima", "Citra Lestari", "Bima Pratama", [
    { eventKindId: kind("Akad"), eventDate: "2027-03-13", startTime: "08:00", endTime: "10:00", venueId: venueIds[0], layout: "Seated",
      sessions: [{ label: "Sesi 1", startTime: "08:00", endTime: "10:00", guests: 120 }] },
    { eventKindId: kind("Resepsi"), eventDate: "2027-03-14", startTime: "11:00", endTime: "15:00", venueId: venueIds[1], layout: "Standing",
      sessions: [{ label: "Sesi 1", startTime: "11:00", endTime: "13:00", guests: 350 }, { label: "Sesi 2", startTime: "13:00", endTime: "15:00", guests: 350 }] },
  ], [
    // The catalog package (Rp120.000.000, 300 guests included) with its components copied in; the Resepsi's 700
    // guests add 8 steps of 50 (Rp32.000.000), and a discount brings the contract to Rp150.000.000 as before.
    { kind: "Package", packageId },
    { kind: "Discount", description: "Potongan harga", amount: 2_000_000 },
  ], "Preparation");
  const p1 = p1Created.id;
  const resepsiP1 = p1Created.events.find((e) => e.kindName === "Resepsi")!.id;
  must("main event", await call(s.api, t, "POST", `/api/v1/projects/${p1}/events/${resepsiP1}/set-main`));
  const p2 = (await project("Dewi & Arga", "Dewi Anggraini", "Arga Wijaya", [resepsi("2026-09-20")], paketPremium(90_000_000), "Draft")).id;
  const p3 = (await project("Sari & Doni", "Sari Puspita", "Doni Saputra", [resepsi("2027-06-20")], paketPremium(120_000_000), "Draft")).id;

  // Penawaran (paket WO Tahap 5): a draft without a revision, one shared as Revisi 1 that runs out within a week, and
  // one accepted. The business clock is pinned to 9 Nov 2026, so their numbers and "segera kedaluwarsa" never drift.
  async function quotation(couple: string, whatsapp: string, validUntil: string, eventDate: string, price: number) {
    const id = must(`quotation ${couple}`, await call(s.api, t, "POST", "/api/v1/quotations", {
      coupleNames: couple, contactWhatsapp: whatsapp, picStaffId: 0, validUntil, notes: "",
    })).json.data.quotation.id as number;
    const detail = async () => must(`quotation ${couple}`, await call(s.api, t, "GET", `/api/v1/quotations/${id}`)).json.data.quotation as { updatedAt: string; linesUpdatedAt: string; termsUpdatedAt: string };
    // A new quotation takes its validity from Profil usaha (14 days); the header's own date is set by an update.
    must(`quotation validity ${couple}`, await call(s.api, t, "PUT", `/api/v1/quotations/${id}`, {
      coupleNames: couple, contactWhatsapp: whatsapp, picStaffId: owner.staffId, validUntil, notes: "",
    }, { "If-Match": `"${(await detail()).updatedAt}"` }));
    must(`quotation event ${couple}`, await call(s.api, t, "POST", `/api/v1/quotations/${id}/events`, resepsi(eventDate)));
    must(`quotation lines ${couple}`, await call(s.api, t, "PUT", `/api/v1/quotations/${id}/lines`, {
      lines: [{ kind: "Item", description: "Paket Pernikahan Lengkap", quantity: 1, unit: "paket", unitPrice: price }],
    }, { "If-Match": `"${(await detail()).linesUpdatedAt}"` }));
    must(`quotation terms ${couple}`, await call(s.api, t, "PUT", `/api/v1/quotations/${id}/terms`, {
      terms: [
        { label: "DP", percent: 30, amount: null, dueBasis: "Accepted", dueOffsetDays: 0, dueEventId: null },
        { label: "Pelunasan", percent: 70, amount: null, dueBasis: "Event", dueOffsetDays: -14, dueEventId: null },
      ],
    }, { "If-Match": `"${(await detail()).termsUpdatedAt}"` }));
    return id;
  }
  await quotation("Nadia & Reza", "081277700001", "2026-11-23", "2027-08-15", 85_000_000);
  const sent = await quotation("Putri & Fajar", "081277700002", "2026-11-13", "2027-05-02", 110_000_000);
  must("revision Putri & Fajar", await call(s.api, t, "POST", `/api/v1/quotations/${sent}/revisions`));
  const accepted = await quotation("Ayu & Rizky", "081277700003", "2026-11-30", "2027-07-11", 95_000_000);
  must("revision Ayu & Rizky", await call(s.api, t, "POST", `/api/v1/quotations/${accepted}/revisions`));
  must("accept Ayu & Rizky", await call(s.api, t, "POST", `/api/v1/quotations/${accepted}/status`, {
    status: "Accepted", revisionN: 1, acceptedOn: "2026-11-08", channel: "WhatsApp",
  }));

  // Projects are created as Draft whatever the body says; the lifecycle moves through explicit actions.
  const step = async (id: number, action: string) => must(`${action} ${id}`, await call(s.api, t, "POST", `/api/v1/projects/${id}/transition`, { action }));
  await step(p1, "start_preparation");
  for (const a of ["start_preparation", "mark_ready", "complete"]) await step(p2, a);

  // P1: timeline progress, vendors, money, an issue
  const ms = must("milestones", await call(s.api, t, "GET", `/api/v1/projects/${p1}/milestones`)).json.data as { id: number; targetDate: string; updatedAt: string }[];
  for (const m of ms.slice(0, 2)) {
    must("milestone done", await call(s.api, t, "PATCH", `/api/v1/projects/${p1}/milestones/${m.id}`, { status: "Completed", targetDate: m.targetDate, completedDate: day(-20) }, { "If-Match": `"${m.updatedAt}"` }));
  }
  // Vendor & venue: the catering is paid by the WO (with its agreed cost), the decoration by the couple directly.
  const supplier = must("supplier catering", await call(s.api, t, "POST", `/api/v1/projects/${p1}/suppliers`, {
    vendorId: vendorIds[0], paidBy: "WO", showName: true, agreedCost: 45_000_000, engagementStatus: "Booked",
    bookingDate: day(-40), dueDate: day(40), picStaffId: owner.staffId, scope: "Katering 500 pax", notes: "",
  })).json.data.id as number;
  must("supplier decor", await call(s.api, t, "POST", `/api/v1/projects/${p1}/suppliers`, {
    vendorId: vendorIds[1], paidBy: "Client", showName: true, agreedCost: null, engagementStatus: "Negotiation",
    bookingDate: "", dueDate: day(60), picStaffId: owner.staffId, scope: "Dekorasi pelaminan", notes: "",
  }));
  must("vendor milestone", await call(s.api, t, "POST", `/api/v1/projects/${p1}/suppliers/${supplier}/milestones`, {
    name: "Tasting menu", description: "Cicip menu bersama klien", targetDate: day(20), picStaffId: owner.staffId,
  }));
  must("payment terms", await call(s.api, t, "PUT", `/api/v1/projects/${p1}/payment-terms`, {
    terms: [
      { label: "DP 30%", amount: 45_000_000, dueDate: day(-39) },
      { label: "Termin 2", amount: 40_000_000, dueDate: day(-15) }, // overdue
      { label: "Pelunasan", amount: 65_000_000, dueDate: "2027-02-14" },
    ],
  }));
  const dp = must("client payment", await call(s.api, t, "POST", `/api/v1/projects/${p1}/client-payments`, {
    type: "DP", amount: 45_000_000, paymentDate: day(-38), method: "Transfer", referenceNumber: "TRF-0001", notes: "Transfer BCA",
  })).json.data.id as number;
  must("supplier payment", await call(s.api, t, "POST", `/api/v1/projects/${p1}/supplier-payments`, {
    projectSupplierId: supplier, type: "DP", amount: 15_000_000, paymentDate: day(-35), method: "Transfer", referenceNumber: "VND-0001", notes: "",
  }));
  must("issue", await call(s.api, t, "POST", `/api/v1/projects/${p1}/issues`, {
    projectSupplierId: supplier, title: "Menu vegetarian belum dikonfirmasi", description: "Katering belum memastikan 30 porsi vegetarian.", impact: "Medium",
    foundDate: day(-3), resolutionPlan: "Telepon katering dan minta konfirmasi tertulis.", picStaffId: owner.staffId, targetResolutionDate: day(4),
  }));
  if (s.s3) {
    // Proof (E4): the DP has its transfer receipt and the WO has verified it; the first finished step has a shared report.
    must("dp proof", await call(s.api, t, "POST", `/api/v1/projects/${p1}/evidence`, {
      name: "Bukti transfer DP", type: "Transfer Proof", fileName: "bukti-dp.pdf", mimeType: "application/pdf", base64Data: pdf, documentDate: day(-38), description: "",
      relatedKind: "clientPayment", relatedId: dp, isClientVisible: false,
    }));
    must("verify dp", await call(s.api, t, "POST", `/api/v1/projects/${p1}/client-payments/${dp}/verify`));
    must("step proof", await call(s.api, t, "POST", `/api/v1/projects/${p1}/evidence`, {
      name: "Laporan survei lokasi", type: "Document", fileName: "survei-lokasi.pdf", mimeType: "application/pdf", base64Data: pdf, documentDate: day(-20), description: "",
      relatedKind: "projectMilestone", relatedId: ms[0].id, isClientVisible: true,
    }));
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
  const clientSession = sessionOf(clientLogin);

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
  writeAuth(AUTH.paying, webURL, paying.session);
  writeAuth(AUTH.anon, webURL, null);

  return {
    project: p1, projectDone: p2, projectDraft: p3, tenantContoh: owner.tenantId, tenantBerbayar: paying.tenantId, tenantKosong: empty.tenantId,
    vendor: vendorIds[0], supplier, transaction: txId,
  };
}

/**
 * Real timestamps (created_at, paid_at, audit times) differ on every run. Pin them to the fixed instant
 * 2026-11-09 10:00 WIB plus the row id in seconds, so ordering is kept and text on screen never drifts.
 * The trial end is pinned too, so "Masa coba: 14 hari lagi" counts against the business clock (FIXED_NOW) on every run.
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
