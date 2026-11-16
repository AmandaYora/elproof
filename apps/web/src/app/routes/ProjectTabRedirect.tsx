import { Navigate, useLocation, useParams } from "react-router-dom";
import { ROUTE_PATHS, type ProjectDetailTab } from "@/app/routes/route-paths";

/** Sends an old project tab address (`/proyek/7/pembayaran/client`) to the tab that holds it now, keeping the query and hash. */
export function ProjectTabRedirect({ to }: { to: ProjectDetailTab }) {
  const { projectId = "" } = useParams<{ projectId: string }>();
  const { search, hash } = useLocation();
  return <Navigate to={`${ROUTE_PATHS.projectDetail(projectId, to)}${search}${hash}`} replace />;
}
