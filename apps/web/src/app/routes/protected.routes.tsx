import { lazy } from "react";
import { Navigate, type RouteObject } from "react-router-dom";
import { AppLayout } from "@/shared/layouts/AppLayout";
import { RequireAuth } from "@/shared/components/RequireAuth";
import { RequirePermission } from "@/shared/components/RequirePermission";
import { ROUTE_PATHS } from "@/app/routes/route-paths";

const DashboardPage = lazy(() => import("@/modules/dashboard/pages/DashboardPage"));
const ProjectListPage = lazy(() => import("@/modules/projects/pages/ProjectListPage"));
const ProjectCreatePage = lazy(() => import("@/modules/projects/pages/ProjectCreatePage"));
const ProjectDetailLayout = lazy(() => import("@/modules/projects/pages/ProjectDetailLayout"));
const ProjectSummaryTabPage = lazy(() => import("@/modules/projects/pages/tabs/ProjectSummaryTabPage"));
const ProjectIssuesTabPage = lazy(() => import("@/modules/projects/pages/tabs/ProjectIssuesTabPage"));
const ProjectVendorTabPage = lazy(() => import("@/modules/projects/pages/tabs/ProjectVendorTabPage"));
const ProjectMilestoneTabPage = lazy(() => import("@/modules/projects/pages/tabs/ProjectMilestoneTabPage"));
const ProjectClientTabPage = lazy(() => import("@/modules/projects/pages/tabs/ProjectClientTabPage"));
const ProjectPaymentsTabPage = lazy(() => import("@/modules/projects/pages/tabs/ProjectPaymentsTabPage"));
const PembayaranClientTabPage = lazy(() => import("@/modules/projects/pages/tabs/PembayaranClientTabPage"));
const PembayaranVendorTabPage = lazy(() => import("@/modules/projects/pages/tabs/PembayaranVendorTabPage"));
const PembayaranVenueTabPage = lazy(() => import("@/modules/projects/pages/tabs/PembayaranVenueTabPage"));
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
const SubscriptionReceiptPage = lazy(() => import("@/modules/subscription/pages/SubscriptionReceiptPage"));
const BusinessProfilePage = lazy(() => import("@/modules/business-profile/pages/BusinessProfilePage"));
const RolesPage = lazy(() => import("@/modules/access/pages/RolesPage"));

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
        // A static path wins over "/projects/:projectId" below.
        {
          element: <RequirePermission all={["project.create"]} />,
          children: [{ path: ROUTE_PATHS.projectNew, element: <ProjectCreatePage /> }],
        },
        {
          path: "/projects/:projectId",
          element: <ProjectDetailLayout />,
          children: [
            { index: true, element: <Navigate to="ringkasan" replace /> },
            { path: "ringkasan", element: <ProjectSummaryTabPage /> },
            { path: "kendala", element: <ProjectIssuesTabPage /> },
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
            { path: "dokumen", element: <ProjectEvidenceTabPage /> },
            { path: "aktivitas", element: <ProjectActivityTabPage /> },
            { path: "venue", element: <ProjectVenueTabPage /> },
          ],
        },
        // Each page follows the workspace role's permissions (ADR-0026).
        // Open to every staff role: the page itself shows the tenant-wide
        // dashboard to roles that reach every project, "Proyek saya" to the rest.
        { path: ROUTE_PATHS.dashboard, element: <DashboardPage /> },
        {
          element: <RequirePermission all={["client.manage"]} allProjects />,
          children: [{ path: ROUTE_PATHS.clients, element: <ClientListPage /> }],
        },
        {
          element: <RequirePermission all={["vendor.view"]} />,
          children: [
            { path: ROUTE_PATHS.vendors, element: <VendorListPage /> },
            { path: ROUTE_PATHS.venues, element: <VenueListPage /> },
          ],
        },
        { element: <RequirePermission all={["vendor.category.manage"]} />, children: [{ path: ROUTE_PATHS.vendorCategories, element: <VendorCategoryListPage /> }] },
        { element: <RequirePermission all={["template.manage"]} />, children: [{ path: ROUTE_PATHS.milestoneTemplates, element: <MilestoneTemplateListPage /> }] },
        { element: <RequirePermission all={["team.manage"]} />, children: [{ path: ROUTE_PATHS.users, element: <UserListPage /> }] },
        { element: <RequirePermission all={["business_profile.manage"]} />, children: [{ path: ROUTE_PATHS.businessProfile, element: <BusinessProfilePage /> }] },
        { element: <RequirePermission all={["role.manage"]} />, children: [{ path: ROUTE_PATHS.roles, element: <RolesPage /> }] },
        { element: <RequirePermission all={["subscription.manage"]} />, children: [{ path: ROUTE_PATHS.subscription, element: <SubscriptionPage /> }, { path: "/langganan/kuitansi/:id", element: <SubscriptionReceiptPage /> }] },
      ],
    },
  ],
};
