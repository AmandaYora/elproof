import { useOutletContext } from "react-router-dom";
import { ClientPaidBook } from "@/modules/projects/components/supplier-bills/ClientPaidBook";
import type { ProjectDetailContext } from "@/modules/projects/pages/ProjectDetailLayout";

/**
 * Keuangan › Dibayar klien langsung: the vendors and venues the client pays itself, their bills ("Tagihan vendor") and
 * the proofs the couple sent from the portal, checked here ("?periksa=<id>" opens one).
 */
export default function FinanceClientPaidTabPage() {
  const { projectId } = useOutletContext<ProjectDetailContext>();
  return <ClientPaidBook projectId={projectId} />;
}
