import { useOutletContext } from "react-router-dom";
import { ClientPaymentsSection } from "@/modules/projects/components/detail/ClientPaymentsSection";
import { ProjectPaymentsSection } from "@/modules/projects/components/detail/ProjectPaymentsSection";
import type { ProjectDetailContext } from "@/modules/projects/pages/ProjectDetailLayout";

// Two sections stacked, money in then money out (PLAN.md "Uang Masuk dari
// Client") — matches the left-to-right order on ProjectHeaderCard (Nilai
// Kontrak -> Sisa Tagihan Client -> Margin).
export default function ProjectPaymentsTabPage() {
  const { projectId } = useOutletContext<ProjectDetailContext>();
  return (
    <div className="flex flex-col gap-6">
      <ClientPaymentsSection projectId={projectId} />
      <ProjectPaymentsSection projectId={projectId} />
    </div>
  );
}
