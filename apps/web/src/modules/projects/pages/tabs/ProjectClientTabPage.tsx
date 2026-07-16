import { useOutletContext } from "react-router-dom";
import { ProjectClientsSection } from "@/modules/projects/components/detail/ProjectClientsSection";
import type { ProjectDetailContext } from "@/modules/projects/pages/ProjectDetailLayout";

export default function ProjectClientTabPage() {
  const { projectId } = useOutletContext<ProjectDetailContext>();
  return <ProjectClientsSection projectId={projectId} />;
}
