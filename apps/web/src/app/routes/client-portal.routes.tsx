import { lazy } from "react";
import { Navigate, type RouteObject } from "react-router-dom";
import { RequireAuth } from "@/shared/components/RequireAuth";

const ClientPortalLayout = lazy(() => import("@/modules/client-portal/layouts/ClientPortalLayout"));
const HomeTabPage = lazy(() => import("@/modules/client-portal/pages/tabs/HomeTabPage"));
const MoreTabPage = lazy(() => import("@/modules/client-portal/pages/tabs/MoreTabPage"));
const VendorTabPage = lazy(() => import("@/modules/client-portal/pages/tabs/VendorTabPage"));
const TimelineTabPage = lazy(() => import("@/modules/client-portal/pages/tabs/TimelineTabPage"));
const PembayaranTabPage = lazy(() => import("@/modules/client-portal/pages/tabs/PembayaranTabPage"));
const KendalaTabPage = lazy(() => import("@/modules/client-portal/pages/tabs/KendalaTabPage"));
const DokumenTabPage = lazy(() => import("@/modules/client-portal/pages/tabs/DokumenTabPage"));

// Beranda is the landing page (Fase 5). "ringkasan" and "venue" are old
// bookmarks and shared links: they redirect so they still land somewhere real.
export const clientPortalRoutes: RouteObject = {
  path: "/portal",
  element: <RequireAuth allow={["client"]} />,
  children: [
    {
      element: <ClientPortalLayout />,
      children: [
        { index: true, element: <Navigate to="beranda" replace /> },
        { path: "beranda", element: <HomeTabPage /> },
        { path: "lainnya", element: <MoreTabPage /> },
        { path: "ringkasan", element: <Navigate to="../beranda" replace /> },
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
