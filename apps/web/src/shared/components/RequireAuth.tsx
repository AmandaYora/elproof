import { Navigate, Outlet, useLocation } from "react-router-dom";
import { useAuthStore, type PrincipalType } from "@/shared/stores/useAuthStore";
import { ROUTE_PATHS } from "@/app/routes/route-paths";

interface RequireAuthProps {
  allow: PrincipalType[];
}

// Gates an entire route subtree by principal type — WO Console (`staff`),
// Client Portal (`client`), and Platform Console (`platform_admin`) are
// otherwise indistinguishable at the router level (all three share one
// `useAuthStore` session shape), so without this a client with a valid token
// could navigate straight to `/platform/dashboard` and see the page shell
// render before any API call 401s. Role-specific gates (e.g. Owner-only
// pages) are handled locally in the page itself, not here.
export function RequireAuth({ allow }: RequireAuthProps) {
  const session = useAuthStore((s) => s.session);
  const location = useLocation();

  if (!session || !allow.includes(session.principalType)) {
    return <Navigate to={ROUTE_PATHS.login} replace state={{ from: location }} />;
  }

  return <Outlet />;
}
