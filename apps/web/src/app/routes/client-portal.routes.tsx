import { lazy } from "react";
import { Navigate, type RouteObject } from "react-router-dom";
import { RequireAuth } from "@/shared/components/RequireAuth";

const ClientPortalLayout = lazy(() => import("@/modules/client-portal/layouts/ClientPortalLayout"));
const RingkasanTabPage = lazy(() => import("@/modules/client-portal/pages/tabs/RingkasanTabPage"));
const VendorProgressTabPage = lazy(() => import("@/modules/client-portal/pages/tabs/VendorProgressTabPage"));
const PembayaranTabPage = lazy(() => import("@/modules/client-portal/pages/tabs/PembayaranTabPage"));
const KendalaTabPage = lazy(() => import("@/modules/client-portal/pages/tabs/KendalaTabPage"));

export const clientPortalRoutes: RouteObject = {
  path: "/portal",
  element: <RequireAuth allow={["client"]} />,
  children: [
    {
      element: <ClientPortalLayout />,
      children: [
        { index: true, element: <Navigate to="ringkasan" replace /> },
        { path: "ringkasan", element: <RingkasanTabPage /> },
        { path: "vendor", element: <VendorProgressTabPage /> },
        { path: "pembayaran", element: <PembayaranTabPage /> },
        { path: "kendala", element: <KendalaTabPage /> },
      ],
    },
  ],
};
