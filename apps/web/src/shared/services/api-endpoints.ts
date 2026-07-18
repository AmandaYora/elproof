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
    tenantToggleSuspension: (id: string) => `/api/v1/tenants/${id}/toggle-suspension`,
    tenantResetCredential: (id: string) => `/api/v1/tenants/${id}/reset-credential`,
    tenantActivateSubscription: (id: string) => `/api/v1/tenants/${id}/activate-subscription`,
    subscriptionsPay: "/api/v1/subscriptions/pay",
    platformAdmins: "/api/v1/platform-admins",
    platformAdmin: (id: string) => `/api/v1/platform-admins/${id}`,
    platformAdminToggleActive: (id: string) => `/api/v1/platform-admins/${id}/toggle-active`,
    platformAdminResetPassword: (id: string) => `/api/v1/platform-admins/${id}/reset-password`,
  },
  staff: {
    base: "/api/v1/staff",
    item: (id: string) => `/api/v1/staff/${id}`,
    toggleActive: (id: string) => `/api/v1/staff/${id}/toggle-active`,
  },
  vendors: {
    categories: "/api/v1/vendor-categories",
    category: (id: string) => `/api/v1/vendor-categories/${id}`,
    categoryToggleActive: (id: string) => `/api/v1/vendor-categories/${id}/toggle-active`,
    base: "/api/v1/vendors",
    item: (id: string) => `/api/v1/vendors/${id}`,
    toggleActive: (id: string) => `/api/v1/vendors/${id}/toggle-active`,
    projectHistory: (id: string) => `/api/v1/vendors/${id}/project-history`,
  },
  projects: {
    base: "/api/v1/projects",
    me: "/api/v1/projects/me",
    item: (id: string) => `/api/v1/projects/${id}`,
    cancel: (id: string) => `/api/v1/projects/${id}/cancel`,
    milestones: (id: string) => `/api/v1/projects/${id}/milestones`,
    milestone: (id: string, milestoneId: string) => `/api/v1/projects/${id}/milestones/${milestoneId}`,
    vendors: (id: string) => `/api/v1/projects/${id}/vendors`,
    vendor: (id: string, pvId: string) => `/api/v1/projects/${id}/vendors/${pvId}`,
    vendorCancel: (id: string, pvId: string) => `/api/v1/projects/${id}/vendors/${pvId}/cancel`,
    vendorMilestones: (id: string, pvId: string) => `/api/v1/projects/${id}/vendors/${pvId}/milestones`,
    vendorMilestone: (id: string, pvId: string, milestoneId: string) =>
      `/api/v1/projects/${id}/vendors/${pvId}/milestones/${milestoneId}`,
    payments: (id: string) => `/api/v1/projects/${id}/payments`,
    issues: (id: string) => `/api/v1/projects/${id}/issues`,
    issue: (id: string, issueId: string) => `/api/v1/projects/${id}/issues/${issueId}`,
    evidence: (id: string) => `/api/v1/projects/${id}/evidence`,
    evidenceFile: (id: string, evidenceId: string) => `/api/v1/projects/${id}/evidence/${evidenceId}/file`,
    activity: (id: string) => `/api/v1/projects/${id}/activity`,
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
