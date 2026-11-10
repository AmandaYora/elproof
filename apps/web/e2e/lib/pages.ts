import type { Page } from "@playwright/test";
import { AUTH } from "./config";

export type AuthKey = keyof typeof AUTH;

/** A golden page: one stable route of one console, with what to break for its error state and how to open its dialog. */
export interface GoldenPage {
  id: string;
  auth: AuthKey;
  route: string;
  /** API paths (prefix of the pathname) to abort for the error-state screenshot. */
  fail?: string[];
  /** Shown also with an empty workspace (auth "empty"). */
  empty?: boolean;
}

export interface GoldenDialog {
  id: string;
  auth: AuthKey;
  route: string;
  open: (page: Page) => Promise<void>;
}

const P = 1; // the rich project ("Citra & Bima")

export const WO_PAGES: GoldenPage[] = [
  { id: "beranda", auth: "owner", route: "/dashboard", fail: ["/api/v1/dashboard"], empty: true },
  { id: "proyek", auth: "owner", route: "/projects", fail: ["/api/v1/projects"], empty: true },
  { id: "proyek-ringkasan", auth: "owner", route: `/projects/${P}/ringkasan`, fail: [`/api/v1/projects/${P}`] },
  { id: "proyek-vendor", auth: "owner", route: `/projects/${P}/vendor` },
  { id: "proyek-keuangan", auth: "owner", route: `/projects/${P}/pembayaran/client` },
  { id: "proyek-kendala", auth: "owner", route: `/projects/${P}/kendala` },
  { id: "proyek-dokumen", auth: "owner", route: `/projects/${P}/dokumen` },
  { id: "klien", auth: "owner", route: "/clients", fail: ["/api/v1/clients"], empty: true },
  { id: "vendor", auth: "owner", route: "/vendors", fail: ["/api/v1/vendors"], empty: true },
  { id: "venue", auth: "owner", route: "/venues", fail: ["/api/v1/venues"], empty: true },
  { id: "tim", auth: "owner", route: "/pengguna", fail: ["/api/v1/staff"] },
  { id: "peran-akses", auth: "owner", route: "/peran-akses", fail: ["/api/v1/access/roles"] },
  { id: "profil-usaha", auth: "owner", route: "/profil-usaha" },
  { id: "langganan", auth: "owner", route: "/langganan" },
];

export const WO_DIALOGS: GoldenDialog[] = [
  {
    id: "tambah-vendor",
    auth: "owner",
    route: "/vendors",
    open: async (page) => {
      await page.getByRole("button", { name: "Tambah Vendor" }).click();
    },
  },
  {
    id: "tambah-pembayaran",
    auth: "owner",
    route: `/projects/${P}/pembayaran/client`,
    open: async (page) => {
      await page.getByRole("button", { name: "Tambah Pembayaran" }).click();
    },
  },
];

export const PORTAL_PAGES: GoldenPage[] = [
  { id: "beranda", auth: "client", route: "/portal/beranda", fail: ["/api/v1/projects/me"] },
  { id: "jadwal", auth: "client", route: "/portal/timeline", fail: ["/api/v1/projects/me"] },
  { id: "pembayaran", auth: "client", route: "/portal/pembayaran", fail: ["/api/v1/projects/"] },
  { id: "vendor", auth: "client", route: "/portal/vendor", fail: ["/api/v1/projects/"] },
  { id: "kendala", auth: "client", route: "/portal/kendala", fail: ["/api/v1/projects/"] },
  { id: "dokumen", auth: "client", route: "/portal/dokumen" },
  { id: "lainnya", auth: "client", route: "/portal/lainnya" },
];

export const PLATFORM_PAGES: GoldenPage[] = [
  { id: "ringkasan", auth: "admin", route: "/platform/dashboard", fail: ["/api/v1/platform/metrics"] },
  { id: "tenant", auth: "admin", route: "/platform/tenant", fail: ["/api/v1/tenants"] },
  { id: "tenant-detail", auth: "admin", route: "/platform/tenant/1/langganan", fail: ["/api/v1/console/tenants/1/overview"] },
  { id: "tagihan", auth: "admin", route: "/platform/transaksi", fail: ["/api/v1/subscription-transactions"] },
  { id: "pengumuman", auth: "admin", route: "/platform/pengumuman", fail: ["/api/v1/console/announcements"] },
  { id: "audit-log", auth: "admin", route: "/platform/audit-log", fail: ["/api/v1/audit-log"] },
  { id: "kebijakan", auth: "admin", route: "/platform/kebijakan", fail: ["/api/v1/console/subscription-policy"] },
];

export const PLATFORM_DIALOGS: GoldenDialog[] = [
  {
    id: "tambah-tenant",
    auth: "admin",
    route: "/platform/tenant",
    open: async (page) => {
      await page.getByRole("button", { name: "Tambah Tenant" }).click();
    },
  },
];

// The twelve pages before sign-in share the public theme (ADR-0038): Masuk lives here, not with the WO console.
export const MARKETING_PAGES: GoldenPage[] = [
  { id: "beranda", auth: "anon", route: "/", fail: ["/api/v1/public/plans"] },
  { id: "tentang-kami", auth: "anon", route: "/tentang-kami" },
  { id: "faq", auth: "anon", route: "/faq" },
  { id: "kontak", auth: "anon", route: "/kontak" },
  { id: "syarat-ketentuan", auth: "anon", route: "/syarat-ketentuan" },
  { id: "masuk", auth: "anon", route: "/login" },
  { id: "daftar", auth: "anon", route: "/daftar" },
];
