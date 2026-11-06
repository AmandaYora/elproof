import { lazy } from "react";
import { Navigate, type RouteObject } from "react-router-dom";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import { isPlatformHost } from "@/shared/lib/platform-host";

const MarketingLayout = lazy(() => import("@/modules/homepage/layouts/MarketingLayout"));
const HomePage = lazy(() => import("@/modules/homepage/pages/HomePage"));
const AboutPage = lazy(() => import("@/modules/homepage/pages/AboutPage"));
const TermsPage = lazy(() => import("@/modules/homepage/pages/TermsPage"));
const PrivacyPage = lazy(() => import("@/modules/homepage/pages/PrivacyPage"));
const DataProcessingPage = lazy(() => import("@/modules/homepage/pages/DataProcessingPage"));
const RefundPolicyPage = lazy(() => import("@/modules/homepage/pages/RefundPolicyPage"));
const FaqPage = lazy(() => import("@/modules/homepage/pages/FaqPage"));
const ContactPage = lazy(() => import("@/modules/homepage/pages/ContactPage"));

// On a WO's custom domain "/" goes straight to that WO's branded login, as
// before Fase 2 — never to ElProof's marketing site.
const marketingElement = isPlatformHost() ? <MarketingLayout /> : <Navigate to={ROUTE_PATHS.login} replace />;

export const homepageRoutes: RouteObject = {
  element: marketingElement,
  children: [
    { path: ROUTE_PATHS.homepage, element: <HomePage /> },
    { path: ROUTE_PATHS.homepageAbout, element: <AboutPage /> },
    { path: ROUTE_PATHS.homepageTerms, element: <TermsPage /> },
    { path: ROUTE_PATHS.homepagePrivacy, element: <PrivacyPage /> },
    { path: ROUTE_PATHS.homepageDataProcessing, element: <DataProcessingPage /> },
    { path: ROUTE_PATHS.homepageRefund, element: <RefundPolicyPage /> },
    { path: ROUTE_PATHS.homepageFaq, element: <FaqPage /> },
    { path: ROUTE_PATHS.homepageContact, element: <ContactPage /> },
  ],
};

// The marketing site moved from /homepage/* to the root in Fase 2; old links
// (shared before the move) keep working.
const LEGACY = [
  ["/homepage", ROUTE_PATHS.homepage],
  ["/homepage/tentang-kami", ROUTE_PATHS.homepageAbout],
  ["/homepage/syarat-ketentuan", ROUTE_PATHS.homepageTerms],
  ["/homepage/kebijakan-privasi", ROUTE_PATHS.homepagePrivacy],
  ["/homepage/kebijakan-refund", ROUTE_PATHS.homepageRefund],
  ["/homepage/faq", ROUTE_PATHS.homepageFaq],
  ["/homepage/kontak", ROUTE_PATHS.homepageContact],
] as const;

export const legacyHomepageRedirects: RouteObject[] = LEGACY.map(([from, to]) => ({
  path: from,
  element: <Navigate to={to} replace />,
}));
