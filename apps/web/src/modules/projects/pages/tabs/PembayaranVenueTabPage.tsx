import { useOutletContext } from "react-router-dom";
import { VenuePaymentsSection } from "@/modules/projects/components/detail/VenuePaymentsSection";
import type { ProjectDetailContext } from "@/modules/projects/pages/ProjectDetailLayout";

export default function PembayaranVenueTabPage() {
  const { projectId } = useOutletContext<ProjectDetailContext>();
  return <VenuePaymentsSection projectId={projectId} />;
}
