import { useOutletContext } from "react-router-dom";
import { ProjectIssuesSection } from "@/modules/projects/components/detail/ProjectIssuesSection";
import type { ProjectDetailContext } from "@/modules/projects/pages/ProjectDetailLayout";

export default function ProjectIssuesTabPage() {
  const { projectId } = useOutletContext<ProjectDetailContext>();
  return <ProjectIssuesSection projectId={projectId} />;
}
