import { Check } from "lucide-react";
import { cn } from "@/shared/lib/cn";
import { ProofStamp } from "@/shared/components/ui/ProofStamp";
import { formatDate } from "@/shared/lib/formatters";
import type { ProjectMilestone } from "@/modules/projects/types";
import type { ProofToOpen } from "@/modules/client-portal/components/useProofViewer";

/** The part of a step the rail draws: project steps and a vendor's steps share it. */
export type RailStep = Pick<ProjectMilestone, "id" | "name" | "status" | "targetDate" | "completedDate" | "evidence">;

/** What a step says about itself, in plain words that never blame anyone. */
function stepNote(m: RailStep): string {
  switch (m.status) {
    case "Completed":
      return `Selesai ${formatDate(m.completedDate)}`;
    case "Blocked":
      return "Sedang terhambat. Tim kami sedang menindaklanjuti.";
    case "In Progress":
      return "Sedang dikerjakan oleh tim";
    default:
      return `Target ${formatDate(m.targetDate)}`;
  }
}

/**
 * A vertical rail of steps. The marker says the state by shape as well as color
 * (check, ring with dot, plain ring) and the note says it in words. A finished
 * step that has a shared file shows a stamp that opens it.
 */
export function MilestoneRail({ steps, onOpenProof }: { steps: RailStep[]; onOpenProof: (proof: ProofToOpen, title: string) => void }) {
  return (
    <ol className="flex flex-col">
      {steps.map((m, i) => {
        const done = m.status === "Completed";
        const active = m.status === "In Progress";
        const blocked = m.status === "Blocked";
        const proof = done ? m.evidence[0] : undefined;
        const last = i === steps.length - 1;
        return (
          <li key={m.id} className="relative flex gap-4 pb-6 last:pb-0" aria-current={active ? "step" : undefined}>
            {!last && <span aria-hidden className={cn("absolute top-7 left-[13px] h-[calc(100%-1.75rem)] w-0.5", done ? "bg-navy-900" : "bg-border")} />}
            <span
              aria-hidden
              className={cn(
                "z-10 flex size-7 shrink-0 items-center justify-center rounded-full border-2",
                done && "border-navy-900 bg-navy-900 text-white",
                active && "border-navy-900 bg-card",
                blocked && "border-warning bg-card",
                !done && !active && !blocked && "border-border bg-card",
              )}
            >
              {done ? <Check className="size-4" /> : active ? <span className="size-2.5 rounded-full bg-navy-900" /> : null}
            </span>
            <div className="min-w-0 flex-1 pt-0.5">
              <p className={cn("text-[15px]", done || active ? "font-semibold text-foreground" : "font-medium text-muted-foreground")}>{m.name}</p>
              <p className={cn("mt-0.5 text-[13px]", blocked ? "font-medium text-warning-text" : "text-muted-foreground")}>{stepNote(m)}</p>
              {proof && (
                <ProofStamp className="mt-2" proof={proof} label="Lihat bukti" onOpen={() => onOpenProof(proof, m.name)} />
              )}
            </div>
          </li>
        );
      })}
    </ol>
  );
}
