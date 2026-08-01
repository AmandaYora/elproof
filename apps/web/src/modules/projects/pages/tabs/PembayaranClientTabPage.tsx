import { useOutletContext } from "react-router-dom";
import { ClientPaymentsSection } from "@/modules/projects/components/detail/ClientPaymentsSection";
import type { ProjectDetailContext } from "@/modules/projects/pages/ProjectDetailLayout";

export default function PembayaranClientTabPage() {
  const { projectId } = useOutletContext<ProjectDetailContext>();
  return <ClientPaymentsSection projectId={projectId} />;
}
