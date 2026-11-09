import { CheckCircle2 } from "lucide-react";
import { ProofStamp } from "@/shared/components/ui/ProofStamp";
import { formatDate } from "@/shared/lib/formatters";
import { jakartaDay, verifiedLabel } from "@/modules/projects/lib/proof";
import type { PortalUpdate } from "@/modules/client-portal/types";
import type { ProofToOpen } from "@/modules/client-portal/components/useProofViewer";

/**
 * "Kabar terbaru", newest first. Anything the WO has proof for carries a stamp
 * that opens the file in place (DESIGN.md §3); an update with no proof file is
 * plain text, never a stamp.
 */
export function RecentUpdates({ updates, onOpenProof }: { updates: PortalUpdate[]; onOpenProof: (proof: ProofToOpen, title: string) => void }) {
  if (updates.length === 0) return null;
  return (
    <section aria-labelledby="news-heading" className="rounded-2xl border border-border bg-card p-5 sm:p-6">
      <h2 id="news-heading" className="text-[15px] font-semibold text-foreground">Kabar terbaru</h2>
      <ul className="mt-3 flex flex-col gap-4">
        {updates.map((u) => {
          const proof = u.proof ? { id: u.proof.evidenceId, type: u.proof.type, fileName: u.proof.fileName } : null;
          return (
            <li key={`${u.kind}-${u.at}-${u.title}`} className="flex items-start gap-3">
              <CheckCircle2 className="mt-0.5 size-4 shrink-0 text-success-text" aria-hidden />
              <div className="min-w-0 flex flex-col gap-1.5">
                <p className="text-[14px] text-foreground">
                  {u.title}
                  <span className="block text-[12.5px] text-muted-foreground">{formatDate(jakartaDay(u.at))}</span>
                </p>
                <ProofStamp
                  proof={proof}
                  label={u.kind === "payment_verified" ? verifiedLabel(u.at, u.verifiedBy) : "Lihat bukti"}
                  seenKey={u.kind === "payment_verified" && proof ? `payment-proof-${proof.id}` : undefined}
                  onOpen={() => proof && onOpenProof(proof, u.title)}
                />
              </div>
            </li>
          );
        })}
      </ul>
    </section>
  );
}
