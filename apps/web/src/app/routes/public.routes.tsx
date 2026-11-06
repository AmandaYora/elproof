import { lazy } from "react";
import type { RouteObject } from "react-router-dom";
import { ROUTE_PATHS } from "@/app/routes/route-paths";

const LoginPage = lazy(() => import("@/modules/auth/pages/LoginPage"));
const SignupPage = lazy(() => import("@/modules/auth/pages/SignupPage"));
const WorkspaceSetupPage = lazy(() => import("@/modules/auth/pages/WorkspaceSetupPage"));
const InvitationPage = lazy(() => import("@/modules/auth/pages/InvitationPage"));

export const publicRoutes: RouteObject[] = [
  { path: ROUTE_PATHS.login, element: <LoginPage /> },
  // Self-service WO registration (Fase 2).
  { path: ROUTE_PATHS.signup, element: <SignupPage /> },
  { path: ROUTE_PATHS.signupWorkspace, element: <WorkspaceSetupPage /> },
  // Staff/Owner invitations and client password links (Fase 3).
  { path: "/undangan/:token", element: <InvitationPage /> },
];
