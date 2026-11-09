import { useEffect, useState } from "react";
import { BadgeCheck } from "lucide-react";
import { cn } from "@/shared/lib/cn";

/** What a stamp stands on: the file that proves it. */
export interface ProofRef {
  id: string;
}

interface ProofStampProps {
  /** The file behind the claim. Without one nothing is drawn: a stamp is never decoration. */
  proof: ProofRef | null | undefined;
  /** Words next to the seal, e.g. "Terverifikasi 7 Nov oleh Rina". Never the color alone. */
  label: string;
  /** Opens the proof where the person already is. */
  onOpen: () => void;
  /**
   * Identifies this stamp so the "pressed" animation plays once per browser, the
   * first time it is seen, and never again. Omit to never animate.
   */
  seenKey?: string;
  className?: string;
}

const SEEN_STORAGE = "elproof.stamps.seen";

function readSeen(): string[] {
  try {
    const raw = window.localStorage.getItem(SEEN_STORAGE);
    const parsed: unknown = raw ? JSON.parse(raw) : [];
    return Array.isArray(parsed) ? parsed.filter((x): x is string => typeof x === "string") : [];
  } catch {
    return [];
  }
}

function markSeen(key: string) {
  try {
    const seen = readSeen();
    if (!seen.includes(key)) window.localStorage.setItem(SEEN_STORAGE, JSON.stringify([...seen, key].slice(-200)));
  } catch {
    // Storage may be blocked (private window): the stamp then just always draws still.
  }
}

/**
 * A small seal that says something is verified and opens the proof in one tap.
 * Shown only over something that really has a proof file (DESIGN.md §3); the
 * press animation plays once, never with reduced motion.
 */
export function ProofStamp({ proof, label, onOpen, seenKey, className }: ProofStampProps) {
  // Decided once on first render: re-reading storage after markSeen would cancel the animation midway.
  const [animate] = useState(() => (seenKey ? !readSeen().includes(seenKey) : false));
  useEffect(() => {
    if (proof && seenKey) markSeen(seenKey);
  }, [proof, seenKey]);

  if (!proof) return null;
  return (
    <button
      type="button"
      onClick={onOpen}
      className={cn(
        "inline-flex min-h-8 items-center gap-1.5 rounded-full border border-success/30 bg-success-soft py-1 pr-3 pl-2 text-left text-[12.5px] leading-tight font-medium text-success-text pointer-coarse:min-h-11",
        "hover:bg-success-soft/70 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-foreground",
        className,
      )}
    >
      <BadgeCheck
        aria-hidden
        className={cn("size-4 shrink-0", animate && "motion-safe:animate-[stamp-press_400ms_cubic-bezier(0.34,1.56,0.64,1)_both]")}
      />
      <span>
        {label}
        <span className="sr-only">. Buka bukti.</span>
      </span>
    </button>
  );
}
