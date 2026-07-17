import { useEffect, useState } from "react";
import { cn } from "@/shared/lib/cn";

interface ProofSealProps {
  size?: number;
  /** Plays the one-time "stamp press" entrance. Off by default — use once per page (the hero). */
  animate?: boolean;
  className?: string;
}

const RING_TEXT = "ELPROOF  ·  SATU PINTU, SEMUA TERBUKTI  ·  ";

/**
 * The brand's signature mark: a wax-seal / certification stamp built from the
 * existing --color-warning token, not a new hue — "ElProof" read literally as
 * a mark of proof. Large + animated once in the hero, small + static
 * everywhere else (nav, footer) so the one entrance moment stays singular.
 */
export function ProofSeal({ size = 128, animate = false, className }: ProofSealProps) {
  const [played, setPlayed] = useState(!animate);

  useEffect(() => {
    if (!animate) return;
    const id = requestAnimationFrame(() => setPlayed(true));
    return () => cancelAnimationFrame(id);
  }, [animate]);

  const r = 46;
  const cx = 50;
  const cy = 50;
  const circlePath = `M ${cx - r},${cy} a ${r},${r} 0 1,1 ${r * 2},0 a ${r},${r} 0 1,1 -${r * 2},0`;

  return (
    <svg
      viewBox="0 0 100 100"
      width={size}
      height={size}
      role="img"
      aria-label="Segel ElProof — satu pintu, semua terbukti"
      className={cn(
        "overflow-visible",
        animate && "motion-safe:transition-[transform,opacity] motion-safe:duration-700 motion-safe:ease-out",
        animate && (played ? "motion-safe:scale-100 motion-safe:opacity-100" : "motion-safe:scale-[0.6] motion-safe:opacity-0 motion-safe:rotate-[-14deg]"),
        className
      )}
      style={{ transformOrigin: "50% 50%" }}
    >
      <defs>
        <path id="proof-seal-ring" d={circlePath} fill="none" />
      </defs>

      <circle cx={cx} cy={cy} r={r + 5} fill="none" stroke="currentColor" strokeOpacity="0.35" strokeWidth="0.6" />
      <circle cx={cx} cy={cy} r={r} fill="none" stroke="currentColor" strokeWidth="1.4" />
      <circle cx={cx} cy={cy} r={r - 8} fill="none" stroke="currentColor" strokeOpacity="0.6" strokeWidth="0.8" />

      <text fontSize="6.1" fontWeight="600" letterSpacing="0.5" fill="currentColor">
        <textPath href="#proof-seal-ring" startOffset="0%">
          {RING_TEXT}
        </textPath>
      </text>

      {/* laurel sprigs */}
      {[-1, 1].map((side) => (
        <g key={side} transform={`translate(${cx + side * 24},${cy + 6}) scale(${side},1)`} stroke="currentColor" strokeWidth="1.1" fill="none" strokeLinecap="round">
          <path d="M0,0 C 3,-4 3,-10 0,-16" />
          <path d="M0,-3 C 3,-3.5 5,-5 6,-8" />
          <path d="M0,-8 C 3,-8.5 5,-10 6,-13" />
          <path d="M0,-13 C 2.5,-13.4 4,-14.5 4.5,-16.5" />
        </g>
      ))}

      <path
        d="M35,50.5 L45,60 L66,38"
        fill="none"
        stroke="currentColor"
        strokeWidth="4.2"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  );
}
