import { Suspense, lazy } from "react";
import type { RouteObject } from "react-router-dom";
import { RequireAuth } from "@/shared/components/RequireAuth";
import { ROUTE_PATHS } from "@/app/routes/route-paths";

const PlatformLayout = lazy(() => import("@/modules/platform-admin/layouts/PlatformLayout"));
const PlatformDashboardPage = lazy(() => import("@/modules/platform-admin/pages/PlatformDashboardPage"));
const TenantListPage = lazy(() => import("@/modules/platform-admin/pages/TenantListPage"));
const PlanListPage = lazy(() => import("@/modules/platform-admin/pages/PlanListPage"));
const VoucherListPage = lazy(() => import("@/modules/platform-admin/pages/VoucherListPage"));
const PlatformTransactionsPage = lazy(() => import("@/modules/platform-admin/pages/PlatformTransactionsPage"));
const PlatformUserListPage = lazy(() => import("@/modules/platform-admin/pages/PlatformUserListPage"));
const GatewayConfigPage = lazy(() => import("@/modules/platform-admin/pages/GatewayConfigPage"));
const AppListPage = lazy(() => import("@/modules/platform-admin/pages/AppListPage"));
const TenantDetailPage = lazy(() => import("@/modules/platform-admin/pages/TenantDetailPage"));
const AnnouncementsPage = lazy(() => import("@/modules/platform-admin/pages/AnnouncementsPage"));
const DataRequestsPage = lazy(() => import("@/modules/platform-admin/pages/DataRequestsPage"));
const AuditLogPage = lazy(() => import("@/modules/platform-admin/pages/AuditLogPage"));
const SubscriptionPolicyPage = lazy(() => import("@/modules/platform-admin/pages/SubscriptionPolicyPage"));
const LoginSlidesPage = lazy(() => import("@/modules/platform-admin/pages/LoginSlidesPage"));
const MfaSetupPage = lazy(() => import("@/modules/platform-admin/pages/MfaSetupPage"));

export const platformRoutes: RouteObject = {
  element: <RequireAuth allow={["platform_admin"]} />,
  children: [
    {
      path: ROUTE_PATHS.platformSecurity,
      element: (
        <Suspense fallback={<div className="py-20 text-center text-sm text-text-secondary">Memuat halaman...</div>}>
          <MfaSetupPage />
        </Suspense>
      ),
    },
    {
      element: <PlatformLayout />,
      children: [
        { path: ROUTE_PATHS.platformDashboard, element: <PlatformDashboardPage />, handle: { title: "Ringkasan" } },
        { path: ROUTE_PATHS.platformTenants, element: <TenantListPage />, handle: { title: "Tenant" } },
        { path: `${ROUTE_PATHS.platformTenants}/:id/:tab?`, element: <TenantDetailPage />, handle: { title: "Detail tenant" } },
        { path: ROUTE_PATHS.platformAnnouncements, element: <AnnouncementsPage />, handle: { title: "Pengumuman" } },
        { path: ROUTE_PATHS.platformDataRequests, element: <DataRequestsPage />, handle: { title: "Permintaan data" } },
        { path: ROUTE_PATHS.platformAudit, element: <AuditLogPage />, handle: { title: "Audit log" } },
        { path: ROUTE_PATHS.platformPolicy, element: <SubscriptionPolicyPage />, handle: { title: "Kebijakan langganan" } },
        { path: ROUTE_PATHS.platformLoginSlides, element: <LoginSlidesPage />, handle: { title: "Slide halaman masuk" } },
        { path: ROUTE_PATHS.platformPlans, element: <PlanListPage />, handle: { title: "Paket" } },
        { path: ROUTE_PATHS.platformVouchers, element: <VoucherListPage />, handle: { title: "Voucher" } },
        { path: ROUTE_PATHS.platformTransactions, element: <PlatformTransactionsPage />, handle: { title: "Transaksi" } },
        { path: ROUTE_PATHS.platformUsers, element: <PlatformUserListPage />, handle: { title: "Pengguna platform" } },
        { path: ROUTE_PATHS.platformGatewayConfig, element: <GatewayConfigPage />, handle: { title: "Gateway pembayaran" } },
        { path: ROUTE_PATHS.platformApps, element: <AppListPage />, handle: { title: "Aplikasi" } },
      ],
    },
  ],
};
