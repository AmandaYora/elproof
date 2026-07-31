import { Navigate, Outlet } from "react-router-dom";
import { useAuthStore } from "@/shared/stores/useAuthStore";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import type { StaffRole } from "@/modules/users/types";

interface RequireRoleProps {
  allow: StaffRole[];
}

// Gates a route subtree by staff role — the frontend half of the
// Owner/Admin/Wedding Planner access boundary (the backend API is the real
// one; every endpoint this guards is independently enforced server-side
// too). Without this, hiding a Sidebar item is purely cosmetic: typing the
// URL directly still renders the page (its data calls then fail, but only
// after the page shell already showed). Redirects to the caller's own
// first-accessible route rather than back to login, since they ARE
// authenticated — just not authorized for this specific subtree.
export function RequireRole({ allow }: RequireRoleProps) {
  const role = useAuthStore((s) => s.session?.role) as StaffRole | undefined;

  if (!role || !allow.includes(role)) {
    const fallback = role === "Staff" ? ROUTE_PATHS.projects : ROUTE_PATHS.dashboard;
    return <Navigate to={fallback} replace />;
  }

  return <Outlet />;
}
