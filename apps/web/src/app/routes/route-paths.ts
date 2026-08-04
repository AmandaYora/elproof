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
export type ClientPortalTab = "ringkasan" | "vendor" | "pembayaran" | "kendala" | "venue";

export const ROUTE_PATHS = {
  home: "/",
  login: "/login",
  homepage: "/homepage",
  homepageAbout: "/homepage/tentang-kami",
  homepageTerms: "/homepage/syarat-ketentuan",
  homepagePrivacy: "/homepage/kebijakan-privasi",
  homepageRefund: "/homepage/kebijakan-refund",
  homepageFaq: "/homepage/faq",
  homepageContact: "/homepage/kontak",
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
  portal: (tab: ClientPortalTab = "ringkasan") => `/portal/${tab}`,
  platformDashboard: "/platform/dashboard",
  platformTenants: "/platform/tenant",
  platformPlans: "/platform/paket",
  platformTransactions: "/platform/transaksi",
  platformUsers: "/platform/pengguna",
  platformGatewayConfig: "/platform/pembayaran",
  platformApps: "/platform/aplikasi",
} as const;
