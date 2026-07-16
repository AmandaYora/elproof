import { useOutletContext } from "react-router-dom";
import { ProjectPaymentsSection } from "@/modules/projects/components/detail/ProjectPaymentsSection";
import type { ProjectDetailContext } from "@/modules/projects/pages/ProjectDetailLayout";

export default function ProjectPaymentsTabPage() {
  const { projectId } = useOutletContext<ProjectDetailContext>();
  return <ProjectPaymentsSection projectId={projectId} />;
}
