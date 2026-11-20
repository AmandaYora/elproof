import { useOutletContext } from "react-router-dom";
import { ProjectVendorsSection } from "@/modules/projects/components/detail/ProjectVendorsSection";
import type { ProjectDetailContext } from "@/modules/projects/pages/ProjectDetailLayout";

/** "Vendor & venue": every vendor and venue of the project in one table; a row opens its drawer. */
export default function ProjectVendorTabPage() {
  const { projectId } = useOutletContext<ProjectDetailContext>();
  return <ProjectVendorsSection projectId={projectId} />;
}
