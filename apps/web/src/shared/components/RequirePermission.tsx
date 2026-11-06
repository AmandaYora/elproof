import { Navigate, Outlet } from "react-router-dom";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import { hasPermission, usePermissionStore } from "@/shared/stores/usePermissionStore";

interface RequirePermissionProps {
  /** Every listed permission is required. */
  all: string[];
  /** Also require a role that reaches every project (tenant-wide pages). */
  allProjects?: boolean;
}

/**
 * Gates a route subtree by the workspace role's permissions (ADR-0026) —
 * the frontend half; every endpoint behind these pages enforces the same
 * permission. A person without it lands on Project, which every staff role
 * can open (scoped to their own projects when their role says so).
 */
export function RequirePermission({ all, allProjects = false }: RequirePermissionProps) {
  const grants = usePermissionStore((s) => s.grants);
  const loaded = usePermissionStore((s) => s.loaded);
  if (!loaded) {
    return <div className="py-20 text-center text-sm text-text-secondary">Memeriksa akses...</div>;
  }
  const ok = all.every((p) => hasPermission(grants, p)) && (!allProjects || grants?.projectScope === "all");
  if (!ok) return <Navigate to={ROUTE_PATHS.projects} replace />;
  return <Outlet />;
}
