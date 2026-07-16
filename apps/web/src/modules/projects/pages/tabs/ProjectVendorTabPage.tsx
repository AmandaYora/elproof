import { useOutletContext } from "react-router-dom";
import { ProjectVendorsSection } from "@/modules/projects/components/detail/ProjectVendorsSection";
import type { ProjectDetailContext } from "@/modules/projects/pages/ProjectDetailLayout";

export default function ProjectVendorTabPage() {
  const { projectId } = useOutletContext<ProjectDetailContext>();
  return <ProjectVendorsSection projectId={projectId} />;
}
