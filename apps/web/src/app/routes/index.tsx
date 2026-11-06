import { Suspense } from "react";
import { createBrowserRouter } from "react-router-dom";
import { publicRoutes } from "@/app/routes/public.routes";
import { protectedRoutes } from "@/app/routes/protected.routes";
import { clientPortalRoutes } from "@/app/routes/client-portal.routes";
import { platformRoutes } from "@/app/routes/platform.routes";
import { homepageRoutes, legacyHomepageRedirects } from "@/app/routes/homepage.routes";
import { NotFoundPage } from "@/shared/components/NotFoundPage";

const loadingFallback = <div className="py-20 text-center text-sm text-text-secondary">Memuat halaman...</div>;

export const router = createBrowserRouter([
  ...publicRoutes.map((route) => ({ ...route, element: <Suspense fallback={loadingFallback}>{route.element}</Suspense> })),
  { ...protectedRoutes },
  { ...clientPortalRoutes },
  { ...platformRoutes },
  { ...homepageRoutes },
  ...legacyHomepageRedirects,
  { path: "*", element: <NotFoundPage /> },
]);
