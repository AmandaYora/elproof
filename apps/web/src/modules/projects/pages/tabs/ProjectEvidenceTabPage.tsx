import { useOutletContext } from "react-router-dom";
import { ProjectEvidenceSection } from "@/modules/projects/components/detail/ProjectEvidenceSection";
import type { ProjectDetailContext } from "@/modules/projects/pages/ProjectDetailLayout";

export default function ProjectEvidenceTabPage() {
  const { projectId } = useOutletContext<ProjectDetailContext>();
  return <ProjectEvidenceSection projectId={projectId} />;
}
