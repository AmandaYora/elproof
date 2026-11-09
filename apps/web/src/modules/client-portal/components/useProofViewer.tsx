import { lazy, Suspense, useState, type ReactElement } from "react";
import type { EvidenceType } from "@/modules/projects/types";

// The viewer (dialog, PDF/image handling) is only needed once someone taps a proof.
const EvidenceViewerModal = lazy(() => import("@/shared/components/ui/EvidenceViewerModal").then((m) => ({ default: m.EvidenceViewerModal })));

export interface ProofToOpen {
  id: string;
  type: EvidenceType;
  fileName: string;
}

/**
 * One viewer per page: `open(proof, title)` shows the file in place, `element`
 * goes once in the page. Used by every ProofStamp and "Lihat bukti" button.
 */
export function useProofViewer(projectId: string): { open: (proof: ProofToOpen, title: string) => void; element: ReactElement | null } {
  const [current, setCurrent] = useState<{ proof: ProofToOpen; title: string } | null>(null);
  return {
    open: (proof, title) => setCurrent({ proof, title }),
    element: current ? (
      <Suspense fallback={null}>
        <EvidenceViewerModal
          open
          onClose={() => setCurrent(null)}
          projectId={projectId}
          evidence={{ id: current.proof.id, name: current.title, type: current.proof.type, documentDate: null, fileName: current.proof.fileName }}
        />
      </Suspense>
    ) : null,
  };
}
