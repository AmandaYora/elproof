/**
 * The project page's seven tabs (paket WO DESAIN §3.2): Ringkasan · Acara & rincian · Jadwal · Vendor & venue ·
 * Keuangan · Kendala · Dokumen. Keuangan has three books: the client's payments, the vendors and venues the WO pays
 * (margin permission only), and what the client paid them directly. The clients and the history are panels on Ringkasan.
 */
export type ProjectDetailTab =
  | "ringkasan"
  | "acara-rincian"
  | "jadwal"
  | "vendor"
  | "keuangan/klien"
  | "keuangan/vendor-venue"
  | "keuangan/dibayar-klien"
  | "kendala"
  | "dokumen";

/** The tab paths before 16 Nov 2026, and where each now lives. The router redirects them too (old links keep working). */
export const LEGACY_PROJECT_TABS = {
  milestone: "jadwal",
  venue: "vendor",
  client: "ringkasan",
  aktivitas: "ringkasan",
  pembayaran: "keuangan/klien",
  "pembayaran/client": "keuangan/klien",
  "pembayaran/vendor": "keuangan/vendor-venue",
  "pembayaran/venue": "keuangan/vendor-venue",
} as const satisfies Record<string, ProjectDetailTab>;

/** @deprecated An old tab name; `projectDetail` maps it to its new tab. Use a `ProjectDetailTab`. */
export type LegacyProjectDetailTab = keyof typeof LEGACY_PROJECT_TABS;

function projectTabPath(tab: ProjectDetailTab | LegacyProjectDetailTab): ProjectDetailTab {
  return (LEGACY_PROJECT_TABS as Record<string, ProjectDetailTab>)[tab] ?? (tab as ProjectDetailTab);
}
// The Client Portal's own tab set (DESAIN §3.3): Beranda · Jadwal (timeline) ·
// Pembayaran · Layanan · Lainnya; kendala and dokumen open from Lainnya.
// Layanan holds the package and the vendors' progress; each event's venue is on Jadwal.
export type ClientPortalTab = "beranda" | "lainnya" | "layanan" | "pembayaran" | "kendala" | "timeline" | "dokumen";

export const ROUTE_PATHS = {
  home: "/",
  login: "/login",
  // Self-service WO registration (Fase 2): Google first, then one form.
  signup: "/daftar",
  signupWorkspace: "/daftar/workspace",
  // One-time invitation / password-reset links (Fase 3).
  invitation: (token: string) => `/undangan/${token}`,
  // Marketing site lives at the root since Fase 2 ("/" was a redirect to
  // /login). The old /homepage/* paths redirect here (homepage.routes.tsx).
  homepage: "/",
  homepageAbout: "/tentang-kami",
  homepageTerms: "/syarat-ketentuan",
  homepagePrivacy: "/kebijakan-privasi",
  homepageDataProcessing: "/perjanjian-pemrosesan-data",
  homepageRefund: "/kebijakan-refund",
  homepageFaq: "/faq",
  homepageContact: "/kontak",
  dashboard: "/beranda",
  // Penawaran (paket WO Tahap 5): a quotation before it becomes a project. `?baru=1` opens "Penawaran baru".
  quotations: "/penawaran",
  quotationNew: "/penawaran?baru=1",
  quotationDetail: (id: string) => `/penawaran/${id}`,
  quotationConvert: (id: string) => `/penawaran/${id}/jadikan-proyek`,
  projects: "/proyek",
  projectNew: "/proyek/baru",
  projectDetail: (id: string, tab: ProjectDetailTab | LegacyProjectDetailTab = "ringkasan") => `/proyek/${id}/${projectTabPath(tab)}`,
  clients: "/klien",
  calendar: "/kalender",
  finance: "/keuangan",
  vendorCategories: "/kategori-vendor",
  vendors: "/vendor",
  venues: "/venue",
  users: "/tim",
  subscription: "/langganan",
  subscriptionReceipt: (id: string) => `/langganan/kuitansi/${id}`,
  // Pengaturan → Peran & Akses (Fase 3a).
  roles: "/peran-akses",
  businessProfile: "/profil-usaha",
  milestoneTemplates: "/jadwal-bawaan",
  // Katalog (paket WO Tahap 1): the WO's own packages and event kinds.
  packages: "/paket",
  packageNew: "/paket/baru",
  packageDetail: (id: string) => `/paket/${id}`,
  eventKinds: "/jenis-acara",
  portal: (tab: ClientPortalTab = "beranda") => `/portal/${tab}`,
  platformDashboard: "/platform/dashboard",
  platformTenants: "/platform/tenant",
  platformTenantDetail: (id: string, tab = "profil") => `/platform/tenant/${id}/${tab}`,
  platformPlans: "/platform/paket",
  platformVouchers: "/platform/voucher",
  platformTransactions: "/platform/transaksi",
  platformUsers: "/platform/pengguna",
  platformGatewayConfig: "/platform/pembayaran",
  platformApps: "/platform/aplikasi",
  platformAnnouncements: "/platform/pengumuman",
  platformDataRequests: "/platform/data-privasi",
  platformAudit: "/platform/audit-log",
  platformPolicy: "/platform/kebijakan",
  platformLoginSlides: "/platform/tampilan-login",
  // Two-factor enrolment — outside PlatformLayout, reachable before MFA.
  platformSecurity: "/platform/keamanan",
} as const;
