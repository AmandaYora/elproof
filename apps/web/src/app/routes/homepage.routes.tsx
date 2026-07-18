import { lazy } from "react";
import type { RouteObject } from "react-router-dom";
import { ROUTE_PATHS } from "@/app/routes/route-paths";

const MarketingLayout = lazy(() => import("@/modules/homepage/layouts/MarketingLayout"));
const HomePage = lazy(() => import("@/modules/homepage/pages/HomePage"));
const AboutPage = lazy(() => import("@/modules/homepage/pages/AboutPage"));
const TermsPage = lazy(() => import("@/modules/homepage/pages/TermsPage"));
const PrivacyPage = lazy(() => import("@/modules/homepage/pages/PrivacyPage"));
const RefundPolicyPage = lazy(() => import("@/modules/homepage/pages/RefundPolicyPage"));
const FaqPage = lazy(() => import("@/modules/homepage/pages/FaqPage"));
const ContactPage = lazy(() => import("@/modules/homepage/pages/ContactPage"));

export const homepageRoutes: RouteObject = {
  element: <MarketingLayout />,
  children: [
    { path: ROUTE_PATHS.homepage, element: <HomePage /> },
    { path: ROUTE_PATHS.homepageAbout, element: <AboutPage /> },
    { path: ROUTE_PATHS.homepageTerms, element: <TermsPage /> },
    { path: ROUTE_PATHS.homepagePrivacy, element: <PrivacyPage /> },
    { path: ROUTE_PATHS.homepageRefund, element: <RefundPolicyPage /> },
    { path: ROUTE_PATHS.homepageFaq, element: <FaqPage /> },
    { path: ROUTE_PATHS.homepageContact, element: <ContactPage /> },
  ],
};
