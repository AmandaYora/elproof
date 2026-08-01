import { lazy } from "react";
import { Navigate, type RouteObject } from "react-router-dom";
import { AppLayout } from "@/shared/layouts/AppLayout";
import { RequireAuth } from "@/shared/components/RequireAuth";
import { RequireRole } from "@/shared/components/RequireRole";
import { ROUTE_PATHS } from "@/app/routes/route-paths";

const DashboardPage = lazy(() => import("@/modules/dashboard/pages/DashboardPage"));
const ProjectListPage = lazy(() => import("@/modules/projects/pages/ProjectListPage"));
const ProjectDetailLayout = lazy(() => import("@/modules/projects/pages/ProjectDetailLayout"));
const ProjectVendorTabPage = lazy(() => import("@/modules/projects/pages/tabs/ProjectVendorTabPage"));
const ProjectMilestoneTabPage = lazy(() => import("@/modules/projects/pages/tabs/ProjectMilestoneTabPage"));
const ProjectClientTabPage = lazy(() => import("@/modules/projects/pages/tabs/ProjectClientTabPage"));
const ProjectPaymentsTabPage = lazy(() => import("@/modules/projects/pages/tabs/ProjectPaymentsTabPage"));
const PembayaranClientTabPage = lazy(() => import("@/modules/projects/pages/tabs/PembayaranClientTabPage"));
const PembayaranVendorTabPage = lazy(() => import("@/modules/projects/pages/tabs/PembayaranVendorTabPage"));
const PembayaranVenueTabPage = lazy(() => import("@/modules/projects/pages/tabs/PembayaranVenueTabPage"));
const ProjectIssuesTabPage = lazy(() => import("@/modules/projects/pages/tabs/ProjectIssuesTabPage"));
const ProjectEvidenceTabPage = lazy(() => import("@/modules/projects/pages/tabs/ProjectEvidenceTabPage"));
const ProjectActivityTabPage = lazy(() => import("@/modules/projects/pages/tabs/ProjectActivityTabPage"));
const ProjectVenueTabPage = lazy(() => import("@/modules/projects/pages/tabs/ProjectVenueTabPage"));
const ClientListPage = lazy(() => import("@/modules/clients/pages/ClientListPage"));
const VendorCategoryListPage = lazy(() => import("@/modules/vendor-categories/pages/VendorCategoryListPage"));
const MilestoneTemplateListPage = lazy(() => import("@/modules/milestone-templates/pages/MilestoneTemplateListPage"));
const VendorListPage = lazy(() => import("@/modules/vendors/pages/VendorListPage"));
const VenueListPage = lazy(() => import("@/modules/venues/pages/VenueListPage"));
const UserListPage = lazy(() => import("@/modules/users/pages/UserListPage"));
const SubscriptionPage = lazy(() => import("@/modules/subscription/pages/SubscriptionPage"));

export const protectedRoutes: RouteObject = {
  element: <RequireAuth allow={["staff"]} />,
  children: [
    {
      element: <AppLayout />,
      children: [
        // Project stays open to every staff role, including Wedding Planner
        // (scoped server-side to their own PIC'd projects — see PLAN.md's
        // RBAC section) — no RequireRole wrapper needed here.
        { path: ROUTE_PATHS.projects, element: <ProjectListPage /> },
        {
          path: "/projects/:projectId",
          element: <ProjectDetailLayout />,
          children: [
            { index: true, element: <Navigate to="vendor" replace /> },
            { path: "vendor", element: <ProjectVendorTabPage /> },
            { path: "milestone", element: <ProjectMilestoneTabPage /> },
            { path: "client", element: <ProjectClientTabPage /> },
            {
              path: "pembayaran",
              element: <ProjectPaymentsTabPage />,
              children: [
                { index: true, element: <Navigate to="client" replace /> },
                { path: "client", element: <PembayaranClientTabPage /> },
                { path: "vendor", element: <PembayaranVendorTabPage /> },
                { path: "venue", element: <PembayaranVenueTabPage /> },
              ],
            },
            { path: "kendala", element: <ProjectIssuesTabPage /> },
            { path: "dokumen", element: <ProjectEvidenceTabPage /> },
            { path: "aktivitas", element: <ProjectActivityTabPage /> },
            { path: "venue", element: <ProjectVenueTabPage /> },
          ],
        },
        {
          element: <RequireRole allow={["Owner", "Admin"]} />,
          children: [
            { path: ROUTE_PATHS.dashboard, element: <DashboardPage /> },
            { path: ROUTE_PATHS.clients, element: <ClientListPage /> },
            { path: ROUTE_PATHS.vendors, element: <VendorListPage /> },
            { path: ROUTE_PATHS.venues, element: <VenueListPage /> },
          ],
        },
        {
          element: <RequireRole allow={["Owner"]} />,
          children: [
            { path: ROUTE_PATHS.vendorCategories, element: <VendorCategoryListPage /> },
            { path: ROUTE_PATHS.milestoneTemplates, element: <MilestoneTemplateListPage /> },
            { path: ROUTE_PATHS.users, element: <UserListPage /> },
            { path: ROUTE_PATHS.subscription, element: <SubscriptionPage /> },
          ],
        },
      ],
    },
  ],
};
