import { useOutletContext } from "react-router-dom";
import { ProjectMilestonesSection } from "@/modules/projects/components/detail/ProjectMilestonesSection";
import type { ProjectDetailContext } from "@/modules/projects/pages/ProjectDetailLayout";

export default function ProjectMilestoneTabPage() {
  const { projectId } = useOutletContext<ProjectDetailContext>();
  return <ProjectMilestonesSection projectId={projectId} />;
}
