import { useOutletContext } from "react-router-dom";
import { ClientPaymentsSection } from "@/modules/projects/components/detail/ClientPaymentsSection";
import { PaymentTermsSection } from "@/modules/projects/components/detail/PaymentTermsSection";
import type { ProjectDetailContext } from "@/modules/projects/pages/ProjectDetailLayout";

export default function PembayaranClientTabPage() {
  const { projectId } = useOutletContext<ProjectDetailContext>();
  return (
    <div className="flex flex-col gap-5">
      <PaymentTermsSection projectId={projectId} />
      <ClientPaymentsSection projectId={projectId} />
    </div>
  );
}
