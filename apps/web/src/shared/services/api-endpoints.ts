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
    // Two-factor (TOTP) for Platform Console admins.
    mfaVerify: "/api/v1/auth/mfa/verify",
    mfaSetup: "/api/v1/auth/mfa/setup",
    mfaEnable: "/api/v1/auth/mfa/enable",
    // Sign in with Google for WO Owners and staff (Fase 2).
    google: "/api/v1/auth/google",
    // Link Google to an existing staff login by confirming its password once.
    googleLink: "/api/v1/auth/google/link",
    // One-time invitation / password-reset links (Fase 3).
    link: (token: string) => `/api/v1/auth/links/${encodeURIComponent(token)}`,
    linkGoogle: (token: string) => `/api/v1/auth/links/${encodeURIComponent(token)}/google`,
    linkPassword: (token: string) => `/api/v1/auth/links/${encodeURIComponent(token)}/password`,
  },
  // Workspace roles and permissions (Fase 3a, ADR-0026).
  access: {
    me: "/api/v1/access/me",
    catalog: "/api/v1/access/catalog",
    roles: "/api/v1/access/roles",
    role: (id: number | string) => `/api/v1/access/roles/${id}`,
  },
  billing: {
    plans: "/api/v1/plans",
    plan: (id: string) => `/api/v1/plans/${id}`,
    planToggleActive: (id: string) => `/api/v1/plans/${id}/toggle-active`,
    transactions: "/api/v1/subscription-transactions",
    transaction: (id: string) => `/api/v1/subscription-transactions/${id}`,
    // Subscription ledger of external Apps (D1) — owned by `billing` even
    // though it is read from Manajemen Aplikasi. A plan's App is a field of
    // the plan itself (ADR-0040): `plans?appId=` lists one App's plans.
    appSubscriptions: "/api/v1/app-subscriptions",
    // Discount codes on subscription payments (ADR-0041): Platform Console only; a WO checks one before paying.
    vouchers: "/api/v1/vouchers",
    voucher: (id: number) => `/api/v1/vouchers/${id}`,
    voucherRedemptions: (id: number) => `/api/v1/vouchers/${id}/redemptions`,
    voucherCheck: "/api/v1/subscriptions/voucher-check",
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
  // Platform Console back office (Fase 6): metrics, tenant 360, notes,
  // data requests, announcements, subscription policy, support sessions.
  console: {
    metrics: "/api/v1/platform/metrics",
    trends: "/api/v1/platform/trends",
    tenantOverview: (id: string) => `/api/v1/console/tenants/${id}/overview`,
    tenantNotes: (id: string) => `/api/v1/console/tenants/${id}/notes`,
    tenantDataRequests: (id: string) => `/api/v1/console/tenants/${id}/data-requests`,
    tenantSupport: (id: string) => `/api/v1/console/tenants/${id}/support`,
    tenantImpersonate: (id: string) => `/api/v1/console/tenants/${id}/impersonate`,
    supportSessionEnd: (id: number | string) => `/api/v1/console/support-sessions/${id}/end`,
    dataRequests: "/api/v1/console/data-requests",
    dataRequestComplete: (id: number | string) => `/api/v1/console/data-requests/${id}/complete`,
    announcements: "/api/v1/console/announcements",
    announcement: (id: number | string) => `/api/v1/console/announcements/${id}`,
    subscriptionPolicy: "/api/v1/console/subscription-policy",
    tenantsExport: "/api/v1/tenants/export",
    loginSlide: (index: number) => `/api/v1/console/login-slides/${index}`,
    auditLog: "/api/v1/audit-log",
    auditLogExport: "/api/v1/audit-log/export",
  },
  // Owner side of the same features.
  workspace: {
    supportAccess: "/api/v1/tenants/me/support-access",
    activeAnnouncements: "/api/v1/announcements/active",
  },
  platform: {
    tenants: "/api/v1/tenants",
    tenant: (id: string) => `/api/v1/tenants/${id}`,
    tenantMe: "/api/v1/tenants/me",
    // Lifecycle summary for every staff role (console banner).
    tenantMeSubscription: "/api/v1/tenants/me/subscription",
    tenantMeBranding: "/api/v1/tenants/me/branding",
    tenantMeLogo: "/api/v1/tenants/me/logo",
    tenantMeBusinessDetails: "/api/v1/tenants/me/business-details",
    tenantMeContact: "/api/v1/tenants/me/contact",
    // Owner-only self-service (Fase 2): "Lengkapi Profil" and the
    // "Mulai di sini" checklist with trial usage.
    tenantMeProfile: "/api/v1/tenants/me/profile",
    tenantMeOnboarding: "/api/v1/tenants/me/onboarding",
    tenantMeOnboardingDismiss: "/api/v1/tenants/me/onboarding/dismiss",
    // Free-trial caps, edited in the Platform Console.
    trialLimits: "/api/v1/trial-limits",
    tenantLogo: (id: string) => `/api/v1/tenants/${id}/logo`,
    tenantToggleSuspension: (id: string) => `/api/v1/tenants/${id}/toggle-suspension`,
    // New Owner invitation link (Fase 3) — the team never sets a password.
    tenantOwnerInvite: (id: string) => `/api/v1/tenants/${id}/owner-invite`,
    tenantActivateSubscription: (id: string) => `/api/v1/tenants/${id}/activate-subscription`,
    subscriptionsPay: "/api/v1/subscriptions/pay",
    subscriptionsPendingCharge: "/api/v1/subscriptions/pending-charge",
    subscriptionsCancelPendingCharge: "/api/v1/subscriptions/pending-charge/cancel",
    platformAdmins: "/api/v1/platform-admins",
    platformAdmin: (id: string) => `/api/v1/platform-admins/${id}`,
    platformAdminToggleActive: (id: string) => `/api/v1/platform-admins/${id}/toggle-active`,
    platformAdminResetPassword: (id: string) => `/api/v1/platform-admins/${id}/reset-password`,
    platformAdminResetMfa: (id: string) => `/api/v1/platform-admins/${id}/reset-mfa`,
  },
  // Pre-auth endpoints: no session exists yet.
  public: {
    // LoginPage's left-panel photo slider — a fixed, global set of marketing
    // photos (not tenant-scoped).
    loginSlides: "/api/v1/public/login-slides",
    loginSlide: (index: number) => `/api/v1/public/login-slides/${index}`,
    // Non-secret settings needed before sign-in (Google client ID).
    config: "/api/v1/public/config",
    // Self-service WO registration (Fase 2).
    signupWorkspace: "/api/v1/public/signup/workspace",
    // ElProof's active public plans for the Beranda price section (ADR-0038).
    plans: "/api/v1/public/plans",
  },
  staff: {
    base: "/api/v1/staff",
    item: (id: string) => `/api/v1/staff/${id}`,
    // New one-time invitation link (Fase 3).
    invite: (id: string) => `/api/v1/staff/${id}/invite`,
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
    bulk: "/api/v1/vendors/bulk",
    base: "/api/v1/vendors",
    item: (id: string) => `/api/v1/vendors/${id}`,
    projectHistory: (id: string) => `/api/v1/vendors/${id}/project-history`,
    attachment: (id: string) => `/api/v1/vendors/${id}/attachment`,
    template: "/api/v1/vendors/template",
    import: "/api/v1/vendors/import",
    export: "/api/v1/vendors/export",
    // Public-safe (staff AND client) — {id, name} only, powers Client
    // Portal's Vendor Progress tab. Every other vendor endpoint above is
    // staff-only now that commercial fields (harga akad, lampiran) exist.
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
    bulk: "/api/v1/venues/bulk",
    base: "/api/v1/venues",
    item: (id: string) => `/api/v1/venues/${id}`,
    attachment: (id: string) => `/api/v1/venues/${id}/attachment`,
    deleteImpact: (id: string) => `/api/v1/venues/${id}/delete-impact`,
    template: "/api/v1/venues/template",
    import: "/api/v1/venues/import",
    export: "/api/v1/venues/export",
  },
  projects: {
    base: "/api/v1/projects",
    meHome: "/api/v1/projects/me/home",
    overview: (id: string) => `/api/v1/projects/${id}/overview`,
    item: (id: string) => `/api/v1/projects/${id}`,
    cancel: (id: string) => `/api/v1/projects/${id}/cancel`,
    transition: (id: string) => `/api/v1/projects/${id}/transition`,
    closingChecklist: (id: string) => `/api/v1/projects/${id}/closing-checklist`,
    paymentTerms: (id: string) => `/api/v1/projects/${id}/payment-terms`,
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
    paymentRestore: (id: string, paymentId: string) => `/api/v1/projects/${id}/payments/${paymentId}/restore`,
    clientPayments: (id: string) => `/api/v1/projects/${id}/client-payments`,
    clientPayment: (id: string, paymentId: string) => `/api/v1/projects/${id}/client-payments/${paymentId}`,
    clientPaymentRestore: (id: string, paymentId: string) => `/api/v1/projects/${id}/client-payments/${paymentId}/restore`,
    uploads: (id: string) => `/api/v1/projects/${id}/uploads`,
    clientPaymentVerify: (id: string, paymentId: string) => `/api/v1/projects/${id}/client-payments/${paymentId}/verify`,
    venuePayments: (id: string) => `/api/v1/projects/${id}/venue-payments`,
    venuePayment: (id: string, paymentId: string) => `/api/v1/projects/${id}/venue-payments/${paymentId}`,
    venuePaymentRestore: (id: string, paymentId: string) => `/api/v1/projects/${id}/venue-payments/${paymentId}/restore`,
    issues: (id: string) => `/api/v1/projects/${id}/issues`,
    issue: (id: string, issueId: string) => `/api/v1/projects/${id}/issues/${issueId}`,
    evidence: (id: string) => `/api/v1/projects/${id}/evidence`,
    evidenceFile: (id: string, evidenceId: string) => `/api/v1/projects/${id}/evidence/${evidenceId}/file`,
    // Direct, two-minute link to the stored file (501 when the deployment has none).
    evidenceLink: (id: string, evidenceId: string) => `/api/v1/projects/${id}/evidence/${evidenceId}/link`,
    evidenceItem: (id: string, evidenceId: string) => `/api/v1/projects/${id}/evidence/${evidenceId}`,
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
  dashboardMine: "/api/v1/dashboard/mine",
  clients: {
    base: "/api/v1/clients",
    byProject: (projectId: string) => `/api/v1/clients?projectId=${projectId}`,
    item: (id: string) => `/api/v1/clients/${id}`,
    restore: (id: string) => `/api/v1/clients/${id}/restore`,
    // New invitation or password-reset link (Fase 3); the WO never sets a client password.
    accessLink: (id: string) => `/api/v1/clients/${id}/access-link`,
    replaceRepresentative: (id: string) => `/api/v1/clients/${id}/replace-representative`,
  },
} as const;
