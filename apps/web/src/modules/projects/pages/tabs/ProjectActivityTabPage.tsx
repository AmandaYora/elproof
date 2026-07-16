import { useOutletContext } from "react-router-dom";
import { ProjectActivitySection } from "@/modules/projects/components/detail/ProjectActivitySection";
import type { ProjectDetailContext } from "@/modules/projects/pages/ProjectDetailLayout";

export default function ProjectActivityTabPage() {
  const { projectId } = useOutletContext<ProjectDetailContext>();
  return <ProjectActivitySection projectId={projectId} />;
}
