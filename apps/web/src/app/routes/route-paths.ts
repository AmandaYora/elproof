export type ProjectDetailTab =
  | "vendor"
  | "milestone"
  | "client"
  | "pembayaran/client"
  | "pembayaran/vendor"
  | "pembayaran/venue"
  | "dokumen"
  | "aktivitas"
  | "venue";
// "ringkasan" removed (PLAN.md's Client Portal restructure) — that content
// is now the persistent header, not a tab; "timeline" added, completing
// parity with WO Console's own tab set. "venue" removed separately (Vendor/
// Venue merge, client-facing only) — Venue is now a pinned card inside the
// "vendor" tab itself, not its own route; ProjectDetailTab above is WO
// Console's own (unaffected) tab set and keeps its separate "venue" entry.
export type ClientPortalTab = "vendor" | "pembayaran" | "kendala" | "timeline" | "dokumen";

export const ROUTE_PATHS = {
  home: "/",
  login: "/login",
  // Self-service WO registration (Fase 2): Google first, then one form.
  signup: "/daftar",
  signupWorkspace: "/daftar/workspace",
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
  dashboard: "/dashboard",
  projects: "/projects",
  projectDetail: (id: string, tab: ProjectDetailTab = "vendor") => `/projects/${id}/${tab}`,
  clients: "/clients",
  vendorCategories: "/vendor-categories",
  vendors: "/vendors",
  venues: "/venues",
  users: "/pengguna",
  subscription: "/langganan",
  milestoneTemplates: "/timeline-default",
  portal: (tab: ClientPortalTab = "vendor") => `/portal/${tab}`,
  platformDashboard: "/platform/dashboard",
  platformTenants: "/platform/tenant",
  platformPlans: "/platform/paket",
  platformTransactions: "/platform/transaksi",
  platformUsers: "/platform/pengguna",
  platformGatewayConfig: "/platform/pembayaran",
  platformApps: "/platform/aplikasi",
  // Two-factor enrolment — outside PlatformLayout, reachable before MFA.
  platformSecurity: "/platform/keamanan",
} as const;
