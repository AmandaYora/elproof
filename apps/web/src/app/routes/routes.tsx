import { Suspense } from "react";
import type { RouteObject } from "react-router-dom";
import { publicRoutes } from "@/app/routes/public.routes";
import { protectedRoutes } from "@/app/routes/protected.routes";
import { clientPortalRoutes } from "@/app/routes/client-portal.routes";
import { platformRoutes } from "@/app/routes/platform.routes";
import { homepageRoutes, legacyHomepageRedirects } from "@/app/routes/homepage.routes";
import { NotFoundPage } from "@/shared/components/NotFoundPage";

const loadingFallback = <div className="py-20 text-center text-sm text-text-secondary">Memuat halaman...</div>;

/**
 * The whole route table. The browser router (index.tsx) and the build-time prerender (entry-server.tsx) both read it, so a
 * prerendered page is the same tree the browser hydrates, and the route list the Go server uses to answer 404 (ADR-0042)
 * cannot drift from the router.
 */
export const routes: RouteObject[] = [
  ...publicRoutes.map((route) => ({ ...route, element: <Suspense fallback={loadingFallback}>{route.element}</Suspense> })),
  { ...protectedRoutes },
  { ...clientPortalRoutes },
  { ...platformRoutes },
  { ...homepageRoutes },
  ...legacyHomepageRedirects,
  { path: "*", element: <NotFoundPage /> },
];
