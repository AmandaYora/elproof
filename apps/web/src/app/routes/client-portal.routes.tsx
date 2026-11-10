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
        { path: "beranda", element: <HomeTabPage />, handle: { title: "Beranda" } },
        { path: "lainnya", element: <MoreTabPage />, handle: { title: "Lainnya" } },
        { path: "ringkasan", element: <Navigate to="../beranda" replace /> },
        { path: "venue", element: <Navigate to="../vendor" replace /> },
        { path: "vendor", element: <VendorTabPage />, handle: { title: "Vendor" } },
        { path: "timeline", element: <TimelineTabPage />, handle: { title: "Jadwal" } },
        { path: "pembayaran", element: <PembayaranTabPage />, handle: { title: "Pembayaran" } },
        { path: "kendala", element: <KendalaTabPage />, handle: { title: "Kendala" } },
        { path: "dokumen", element: <DokumenTabPage />, handle: { title: "Dokumen" } },
      ],
    },
  ],
};
