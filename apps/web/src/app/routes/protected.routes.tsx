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
        { path: ROUTE_PATHS.projects, element: <ProjectListPage />, handle: { title: "Proyek" } },
        // A static path wins over "/projects/:projectId" below.
        {
          element: <RequirePermission all={["project.create"]} />,
          children: [{ path: ROUTE_PATHS.projectNew, element: <ProjectCreatePage />, handle: { title: "Proyek baru" } }],
        },
        {
          path: "/projects/:projectId",
          element: <ProjectDetailLayout />,
          children: [
            { index: true, element: <Navigate to="ringkasan" replace /> },
            { path: "ringkasan", element: <ProjectSummaryTabPage />, handle: { title: "Ringkasan proyek" } },
            { path: "kendala", element: <ProjectIssuesTabPage />, handle: { title: "Kendala proyek" } },
            { path: "vendor", element: <ProjectVendorTabPage />, handle: { title: "Vendor proyek" } },
            { path: "milestone", element: <ProjectMilestoneTabPage />, handle: { title: "Jadwal proyek" } },
            { path: "client", element: <ProjectClientTabPage />, handle: { title: "Klien proyek" } },
            {
              path: "pembayaran",
              element: <ProjectPaymentsTabPage />,
              children: [
                { index: true, element: <Navigate to="client" replace /> },
                { path: "client", element: <PembayaranClientTabPage />, handle: { title: "Pembayaran klien" } },
                { path: "vendor", element: <PembayaranVendorTabPage />, handle: { title: "Pembayaran vendor" } },
                { path: "venue", element: <PembayaranVenueTabPage />, handle: { title: "Pembayaran venue" } },
              ],
            },
            { path: "dokumen", element: <ProjectEvidenceTabPage />, handle: { title: "Dokumen proyek" } },
            { path: "aktivitas", element: <ProjectActivityTabPage />, handle: { title: "Aktivitas proyek" } },
            { path: "venue", element: <ProjectVenueTabPage />, handle: { title: "Venue proyek" } },
          ],
        },
        // Each page follows the workspace role's permissions (ADR-0026).
        // Open to every staff role: the page itself shows the tenant-wide
        // dashboard to roles that reach every project, "Proyek saya" to the rest.
        { path: ROUTE_PATHS.dashboard, element: <DashboardPage />, handle: { title: "Beranda" } },
        {
          element: <RequirePermission all={["client.manage"]} allProjects />,
          children: [{ path: ROUTE_PATHS.clients, element: <ClientListPage />, handle: { title: "Klien" } }],
        },
        {
          element: <RequirePermission all={["vendor.view"]} />,
          children: [
            { path: ROUTE_PATHS.vendors, element: <VendorListPage />, handle: { title: "Vendor" } },
            { path: ROUTE_PATHS.venues, element: <VenueListPage />, handle: { title: "Venue" } },
          ],
        },
        { element: <RequirePermission all={["vendor.category.manage"]} />, children: [{ path: ROUTE_PATHS.vendorCategories, element: <VendorCategoryListPage />, handle: { title: "Kategori vendor" } }] },
        { element: <RequirePermission all={["template.manage"]} />, children: [{ path: ROUTE_PATHS.milestoneTemplates, element: <MilestoneTemplateListPage />, handle: { title: "Jadwal bawaan" } }] },
        { element: <RequirePermission all={["team.manage"]} />, children: [{ path: ROUTE_PATHS.users, element: <UserListPage />, handle: { title: "Tim" } }] },
        { element: <RequirePermission all={["business_profile.manage"]} />, children: [{ path: ROUTE_PATHS.businessProfile, element: <BusinessProfilePage />, handle: { title: "Profil usaha" } }] },
        { element: <RequirePermission all={["role.manage"]} />, children: [{ path: ROUTE_PATHS.roles, element: <RolesPage />, handle: { title: "Peran dan akses" } }] },
        { element: <RequirePermission all={["subscription.manage"]} />, children: [{ path: ROUTE_PATHS.subscription, element: <SubscriptionPage />, handle: { title: "Langganan" } }, { path: "/langganan/kuitansi/:id", element: <SubscriptionReceiptPage />, handle: { title: "Kuitansi langganan" } }] },
      ],
    },
  ],
};
