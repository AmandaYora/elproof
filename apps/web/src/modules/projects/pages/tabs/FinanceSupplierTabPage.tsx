import { Navigate, useOutletContext } from "react-router-dom";
import { useCan } from "@/shared/stores/usePermissionStore";
import { ProjectPaymentsSection } from "@/modules/projects/components/detail/ProjectPaymentsSection";
import type { ProjectDetailContext } from "@/modules/projects/pages/ProjectDetailLayout";

/** Keuangan › Vendor & venue dibayar WO: the "Biaya rekanan" book. Without finance.margin.view it is not there at all. */
export default function FinanceSupplierTabPage() {
  const { projectId } = useOutletContext<ProjectDetailContext>();
  const canMargin = useCan("finance.margin.view");
  if (!canMargin) return <Navigate to="../klien" replace />;
  return <ProjectPaymentsSection projectId={projectId} />;
}
