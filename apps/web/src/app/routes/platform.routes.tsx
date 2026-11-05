import { Suspense, lazy } from "react";
import type { RouteObject } from "react-router-dom";
import { RequireAuth } from "@/shared/components/RequireAuth";
import { ROUTE_PATHS } from "@/app/routes/route-paths";

const PlatformLayout = lazy(() => import("@/modules/platform-admin/layouts/PlatformLayout"));
const PlatformDashboardPage = lazy(() => import("@/modules/platform-admin/pages/PlatformDashboardPage"));
const TenantListPage = lazy(() => import("@/modules/platform-admin/pages/TenantListPage"));
const PlanListPage = lazy(() => import("@/modules/platform-admin/pages/PlanListPage"));
const PlatformTransactionsPage = lazy(() => import("@/modules/platform-admin/pages/PlatformTransactionsPage"));
const PlatformUserListPage = lazy(() => import("@/modules/platform-admin/pages/PlatformUserListPage"));
const GatewayConfigPage = lazy(() => import("@/modules/platform-admin/pages/GatewayConfigPage"));
const AppListPage = lazy(() => import("@/modules/platform-admin/pages/AppListPage"));
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
        { path: ROUTE_PATHS.platformDashboard, element: <PlatformDashboardPage /> },
        { path: ROUTE_PATHS.platformTenants, element: <TenantListPage /> },
        { path: ROUTE_PATHS.platformPlans, element: <PlanListPage /> },
        { path: ROUTE_PATHS.platformTransactions, element: <PlatformTransactionsPage /> },
        { path: ROUTE_PATHS.platformUsers, element: <PlatformUserListPage /> },
        { path: ROUTE_PATHS.platformGatewayConfig, element: <GatewayConfigPage /> },
        { path: ROUTE_PATHS.platformApps, element: <AppListPage /> },
      ],
    },
  ],
};
