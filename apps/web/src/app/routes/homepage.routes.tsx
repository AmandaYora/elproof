import { lazy } from "react";
import type { RouteObject } from "react-router-dom";
import { ROUTE_PATHS } from "@/app/routes/route-paths";

const MarketingLayout = lazy(() => import("@/modules/homepage/layouts/MarketingLayout"));
const HomePage = lazy(() => import("@/modules/homepage/pages/HomePage"));
const TermsPage = lazy(() => import("@/modules/homepage/pages/TermsPage"));
const ContactPage = lazy(() => import("@/modules/homepage/pages/ContactPage"));

export const homepageRoutes: RouteObject = {
  element: <MarketingLayout />,
  children: [
    { path: ROUTE_PATHS.homepage, element: <HomePage /> },
    { path: ROUTE_PATHS.homepageTerms, element: <TermsPage /> },
    { path: ROUTE_PATHS.homepageContact, element: <ContactPage /> },
  ],
};
