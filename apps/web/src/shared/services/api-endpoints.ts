// Endpoint paths for apps/api, grouped by backend module — see docs/API_CONTRACT.md.
// `auth`, `billing`, and `platform` are wired to a real backend (Fase 1/2); the
// rest are declared ahead of time so later fases only have to implement the
// module, not invent the path.
export const API = {
  base: "/api/v1",
  auth: {
    login: "/api/v1/auth/login",
    refresh: "/api/v1/auth/refresh",
    logout: "/api/v1/auth/logout",
    me: "/api/v1/auth/me",
  },
  billing: {
    plans: "/api/v1/plans",
    plan: (id: string) => `/api/v1/plans/${id}`,
    planToggleActive: (id: string) => `/api/v1/plans/${id}/toggle-active`,
    transactions: "/api/v1/subscription-transactions",
  },
  // `payment` module — see knowledge/MODULE_PAYMENT.md. Internal-mode
  // (Fase 9) config + external-mode App registry (Fase 10, Platform
  // Console's "Manajemen Aplikasi"). `/auth/app/token` and
  // `/external/payments/*` are consumed by external SaaS callers directly,
  // never by this frontend.
  payment: {
    gatewayConfig: "/api/v1/payment/gateway-config",
    apps: "/api/v1/payment/apps",
    appResetSecret: (appId: string) => `/api/v1/payment/apps/${appId}/reset-secret`,
    appToggleActive: (appId: string) => `/api/v1/payment/apps/${appId}/toggle-active`,
  },
  platform: {
    tenants: "/api/v1/tenants",
    tenant: (id: string) => `/api/v1/tenants/${id}`,
    tenantMe: "/api/v1/tenants/me",
    tenantMeBranding: "/api/v1/tenants/me/branding",
    tenantMeLogo: "/api/v1/tenants/me/logo",
    tenantLogo: (id: string) => `/api/v1/tenants/${id}/logo`,
    tenantToggleSuspension: (id: string) => `/api/v1/tenants/${id}/toggle-suspension`,
    tenantResetCredential: (id: string) => `/api/v1/tenants/${id}/reset-credential`,
    tenantActivateSubscription: (id: string) => `/api/v1/tenants/${id}/activate-subscription`,
    subscriptionsPay: "/api/v1/subscriptions/pay",
    subscriptionsPendingCharge: "/api/v1/subscriptions/pending-charge",
    subscriptionsCancelPendingCharge: "/api/v1/subscriptions/pending-charge/cancel",
    platformAdmins: "/api/v1/platform-admins",
    platformAdmin: (id: string) => `/api/v1/platform-admins/${id}`,
    platformAdminToggleActive: (id: string) => `/api/v1/platform-admins/${id}/toggle-active`,
    platformAdminResetPassword: (id: string) => `/api/v1/platform-admins/${id}/reset-password`,
  },
  // Pre-auth, Host-header-resolved tenant branding (ADR-0015) — powers
  // LoginPage for a tenant's own custom domain. Unlike every other group
  // above, these carry no auth and resolve the tenant from the request's
  // Host header server-side, not from a JWT or an :id param.
  public: {
    branding: "/api/v1/public/branding",
    logo: "/api/v1/public/logo",
    // LoginPage's left-panel photo slider — a fixed, global set of marketing
    // photos (not tenant-scoped), unlike branding/logo above.
    loginSlides: "/api/v1/public/login-slides",
    loginSlide: (index: number) => `/api/v1/public/login-slides/${index}`,
  },
  staff: {
    base: "/api/v1/staff",
    item: (id: string) => `/api/v1/staff/${id}`,
    toggleActive: (id: string) => `/api/v1/staff/${id}/toggle-active`,
    // Public-safe within the tenant (any staff role) — {id, name, title}
    // only, powers every PIC picker/label across the `projects` module.
    // `base` above is Owner-only now (Pengguna management).
    summary: "/api/v1/staff/summary",
    // Hard delete (PLAN.md) — `deleteImpact` names every project this staff
    // member is still a live PIC assignment on, for the confirmation dialog;
    // `item(id)` above is reused with the DELETE verb for the delete itself.
    deleteImpact: (id: string) => `/api/v1/staff/${id}/delete-impact`,
  },
  vendors: {
    categories: "/api/v1/vendor-categories",
    category: (id: string) => `/api/v1/vendor-categories/${id}`,
    categoryToggleActive: (id: string) => `/api/v1/vendor-categories/${id}/toggle-active`,
    base: "/api/v1/vendors",
    item: (id: string) => `/api/v1/vendors/${id}`,
    toggleActive: (id: string) => `/api/v1/vendors/${id}/toggle-active`,
    projectHistory: (id: string) => `/api/v1/vendors/${id}/project-history`,
    attachment: (id: string) => `/api/v1/vendors/${id}/attachment`,
    template: "/api/v1/vendors/template",
    import: "/api/v1/vendors/import",
    export: "/api/v1/vendors/export",
    // Public-safe (staff AND client) — {id, name} only, powers Client
    // Portal's Vendor Progress tab. Every other vendor endpoint above is
    // staff-only now that commercial fields (harga akad, lampiran) exist.
    summary: "/api/v1/vendors/summary",
    // Hard delete (PLAN.md) — `deleteImpact` names every project this vendor
    // is still engaged on, for the confirmation dialog; `item(id)` above is
    // reused with the DELETE verb for the delete itself. Vendor Category's
    // delete has no impact endpoint — it's a hard block, not a dialog (see
    // PLAN.md's carve-out reasoning).
    deleteImpact: (id: string) => `/api/v1/vendors/${id}/delete-impact`,
  },
  // `venues` — its own directory, not a vendor category (ADR-0016). One
  // attachment slot per venue (no more photo gallery), plus a bulk Excel
  // import/template flow (ADR-0016's Revisi).
  venues: {
    base: "/api/v1/venues",
    item: (id: string) => `/api/v1/venues/${id}`,
    toggleActive: (id: string) => `/api/v1/venues/${id}/toggle-active`,
    attachment: (id: string) => `/api/v1/venues/${id}/attachment`,
    deleteImpact: (id: string) => `/api/v1/venues/${id}/delete-impact`,
    template: "/api/v1/venues/template",
    import: "/api/v1/venues/import",
    export: "/api/v1/venues/export",
  },
  projects: {
    base: "/api/v1/projects",
    me: "/api/v1/projects/me",
    item: (id: string) => `/api/v1/projects/${id}`,
    cancel: (id: string) => `/api/v1/projects/${id}/cancel`,
    toggleArchive: (id: string) => `/api/v1/projects/${id}/toggle-archive`,
    duplicate: (id: string) => `/api/v1/projects/${id}/duplicate`,
    milestones: (id: string) => `/api/v1/projects/${id}/milestones`,
    milestone: (id: string, milestoneId: string) => `/api/v1/projects/${id}/milestones/${milestoneId}`,
    vendors: (id: string) => `/api/v1/projects/${id}/vendors`,
    vendor: (id: string, pvId: string) => `/api/v1/projects/${id}/vendors/${pvId}`,
    vendorCancel: (id: string, pvId: string) => `/api/v1/projects/${id}/vendors/${pvId}/cancel`,
    vendorMilestones: (id: string, pvId: string) => `/api/v1/projects/${id}/vendors/${pvId}/milestones`,
    vendorMilestone: (id: string, pvId: string, milestoneId: string) =>
      `/api/v1/projects/${id}/vendors/${pvId}/milestones/${milestoneId}`,
    payments: (id: string) => `/api/v1/projects/${id}/payments`,
    payment: (id: string, paymentId: string) => `/api/v1/projects/${id}/payments/${paymentId}`,
    clientPayments: (id: string) => `/api/v1/projects/${id}/client-payments`,
    clientPayment: (id: string, paymentId: string) => `/api/v1/projects/${id}/client-payments/${paymentId}`,
    venuePayments: (id: string) => `/api/v1/projects/${id}/venue-payments`,
    venuePayment: (id: string, paymentId: string) => `/api/v1/projects/${id}/venue-payments/${paymentId}`,
    issues: (id: string) => `/api/v1/projects/${id}/issues`,
    issue: (id: string, issueId: string) => `/api/v1/projects/${id}/issues/${issueId}`,
    evidence: (id: string) => `/api/v1/projects/${id}/evidence`,
    evidenceFile: (id: string, evidenceId: string) => `/api/v1/projects/${id}/evidence/${evidenceId}/file`,
    evidenceToggleClientVisible: (id: string, evidenceId: string) =>
      `/api/v1/projects/${id}/evidence/${evidenceId}/toggle-client-visible`,
    // Client Portal's own "Dokumen" tab — always only general-kind,
    // client-visible documents, unconditionally (see backend listDocuments).
    documents: (id: string) => `/api/v1/projects/${id}/documents`,
    activity: (id: string) => `/api/v1/projects/${id}/activity`,
    venue: (id: string) => `/api/v1/projects/${id}/venue`,
  },
  // Timeline Default Template (PLAN.md) -- a tenant's own configurable
  // checklist seeded into every new project's Timeline tab, managed from
  // Pengaturan -> Timeline Default. Un-paginated, so no page/search params.
  milestoneTemplates: {
    base: "/api/v1/milestone-templates",
    item: (id: string) => `/api/v1/milestone-templates/${id}`,
  },
  dashboard: "/api/v1/dashboard",
  clients: {
    base: "/api/v1/clients",
    byProject: (projectId: string) => `/api/v1/clients?projectId=${projectId}`,
    item: (id: string) => `/api/v1/clients/${id}`,
    toggleActive: (id: string) => `/api/v1/clients/${id}/toggle-active`,
    resetCredential: (id: string) => `/api/v1/clients/${id}/reset-credential`,
    replaceRepresentative: (id: string) => `/api/v1/clients/${id}/replace-representative`,
  },
} as const;
