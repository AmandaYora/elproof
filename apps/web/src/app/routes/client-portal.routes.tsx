import { lazy } from "react";
import { Navigate, type RouteObject } from "react-router-dom";
import { RequireAuth } from "@/shared/components/RequireAuth";

const ClientPortalLayout = lazy(() => import("@/modules/client-portal/layouts/ClientPortalLayout"));
const VendorTabPage = lazy(() => import("@/modules/client-portal/pages/tabs/VendorTabPage"));
const TimelineTabPage = lazy(() => import("@/modules/client-portal/pages/tabs/TimelineTabPage"));
const PembayaranTabPage = lazy(() => import("@/modules/client-portal/pages/tabs/PembayaranTabPage"));
const KendalaTabPage = lazy(() => import("@/modules/client-portal/pages/tabs/KendalaTabPage"));
const DokumenTabPage = lazy(() => import("@/modules/client-portal/pages/tabs/DokumenTabPage"));

// "ringkasan" no longer exists (PLAN.md's Client Portal restructure — its
// summary content is now the persistent ClientProjectHeaderCard, visible on
// every tab). /portal and /portal/ringkasan both land on "vendor" now,
// matching the reference screenshot's default-active tab.
//
// "venue" no longer has its own tab either (client-facing Vendor/Venue
// merge) -- Venue is now a pinned card inside VendorTabPage's own grid, same
// redirect precedent as "ringkasan" above so an old bookmark/shared link
// still lands somewhere real instead of 404ing.
export const clientPortalRoutes: RouteObject = {
  path: "/portal",
  element: <RequireAuth allow={["client"]} />,
  children: [
    {
      element: <ClientPortalLayout />,
      children: [
        { index: true, element: <Navigate to="vendor" replace /> },
        { path: "ringkasan", element: <Navigate to="../vendor" replace /> },
        { path: "venue", element: <Navigate to="../vendor" replace /> },
        { path: "vendor", element: <VendorTabPage /> },
        { path: "timeline", element: <TimelineTabPage /> },
        { path: "pembayaran", element: <PembayaranTabPage /> },
        { path: "kendala", element: <KendalaTabPage /> },
        { path: "dokumen", element: <DokumenTabPage /> },
      ],
    },
  ],
};
