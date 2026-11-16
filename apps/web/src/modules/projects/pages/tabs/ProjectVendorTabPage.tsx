import { useOutletContext } from "react-router-dom";
import { ProjectVendorsSection } from "@/modules/projects/components/detail/ProjectVendorsSection";
import { ProjectVenueSection } from "@/modules/projects/components/detail/ProjectVenueSection";
import type { ProjectDetailContext } from "@/modules/projects/pages/ProjectDetailLayout";

/** "Vendor & venue": the venue of the event (one, pinned on top) and every vendor engaged for it. */
export default function ProjectVendorTabPage() {
  const { projectId } = useOutletContext<ProjectDetailContext>();
  return (
    <div className="flex flex-col gap-5">
      <ProjectVenueSection projectId={projectId} />
      <ProjectVendorsSection projectId={projectId} />
    </div>
  );
}
