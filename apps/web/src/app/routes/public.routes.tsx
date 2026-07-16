import { lazy } from "react";
import type { RouteObject } from "react-router-dom";
import { ROUTE_PATHS } from "@/app/routes/route-paths";

const LoginPage = lazy(() => import("@/modules/auth/pages/LoginPage"));

export const publicRoutes: RouteObject[] = [{ path: ROUTE_PATHS.login, element: <LoginPage /> }];
