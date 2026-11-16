/**
 * The project page's six tabs (RISET WO Console §3 #4): Ringkasan · Jadwal · Vendor & venue · Keuangan · Kendala ·
 * Dokumen. Keuangan has three ledgers; the clients and the history are panels on Ringkasan.
 */
export type ProjectDetailTab =
  | "ringkasan"
  | "jadwal"
  | "vendor"
  | "keuangan/klien"
  | "keuangan/vendor"
  | "keuangan/venue"
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
  "pembayaran/vendor": "keuangan/vendor",
  "pembayaran/venue": "keuangan/venue",
} as const satisfies Record<string, ProjectDetailTab>;

/** @deprecated An old tab name; `projectDetail` maps it to its new tab. Use a `ProjectDetailTab`. */
export type LegacyProjectDetailTab = keyof typeof LEGACY_PROJECT_TABS;

function projectTabPath(tab: ProjectDetailTab | LegacyProjectDetailTab): ProjectDetailTab {
  return (LEGACY_PROJECT_TABS as Record<string, ProjectDetailTab>)[tab] ?? (tab as ProjectDetailTab);
}
// The Client Portal's own tab set (ClientPortalTab below) has no "ringkasan":
// that content is its persistent header. Venue is a pinned card inside its
// "vendor" tab, as it now is in the WO Console's "Vendor & venue" tab.
export type ClientPortalTab = "beranda" | "lainnya" | "vendor" | "pembayaran" | "kendala" | "timeline" | "dokumen";

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
