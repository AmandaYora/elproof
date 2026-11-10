import type { ReactNode } from "react";
import { Check } from "lucide-react";
import { cn } from "@/shared/lib/cn";

/*
 * Small building blocks of the product mocks on the Beranda (hero phone, the
 * "Cara kerja" screens, the try-it preview). Example data only, drawn with
 * the public theme tokens; nothing here is a real screen of the app.
 */

/** A plum phone with a white screen. Width and screen height come from the caller. */
export function PhoneFrame({ children, className, screenClassName }: { children: ReactNode; className?: string; screenClassName?: string }) {
  return (
    <div className={cn("rounded-[2.5rem] bg-primary p-3 shadow-2xl shadow-primary/30", className)}>
      <div className={cn("relative overflow-hidden rounded-[1.9rem] bg-background px-[18px] py-[22px] text-[14px] leading-snug text-foreground", screenClassName)}>
        {children}
      </div>
    </div>
  );
}

/** One row of a mock screen: a bordered block with a bold first line and a quiet second line. */
export function MockItem({ title, detail, className, children }: { title: ReactNode; detail?: ReactNode; className?: string; children?: ReactNode }) {
  return (
    <div className={cn("mt-2.5 rounded-2xl border border-border px-3.5 py-3", className)}>
      <p className="text-[14px] font-bold">{title}</p>
      {detail && <p className="mt-0.5 text-[12.5px] text-muted-foreground">{detail}</p>}
      {children}
    </div>
  );
}

type Tone = "success" | "warning" | "neutral";

const TONE: Record<Tone, string> = {
  success: "bg-success-soft text-success-text",
  warning: "bg-warning-soft text-warning-text",
  neutral: "bg-muted text-muted-foreground",
};

/** A status chip: always words, plus a check when it is done. */
export function StatusChip({ tone, children, className }: { tone: Tone; children: ReactNode; className?: string }) {
  return (
    <span className={cn("inline-flex shrink-0 items-center gap-1 rounded-full px-2.5 py-1 text-[12px] font-semibold whitespace-nowrap", TONE[tone], className)}>
      {tone === "success" && <Check className="size-3.5" aria-hidden strokeWidth={2.6} />}
      {children}
    </span>
  );
}

/** Progress ring for "18 dari 24 tahapan". */
export function ProgressRing({ done, total, size = 60 }: { done: number; total: number; size?: number }) {
  const r = 15.5;
  const c = 2 * Math.PI * r;
  return (
    <svg width={size} height={size} viewBox="0 0 36 36" aria-hidden className="shrink-0">
      <circle cx="18" cy="18" r={r} fill="none" stroke="currentColor" strokeWidth="4" className="text-accent" />
      <circle
        cx="18"
        cy="18"
        r={r}
        fill="none"
        stroke="currentColor"
        strokeWidth="4"
        strokeLinecap="round"
        strokeDasharray={`${(done / total) * c} ${c}`}
        transform="rotate(-90 18 18)"
        className="text-primary"
      />
    </svg>
  );
}

/** The quiet "Contoh data" tag every mock carries. */
export function SampleTag({ className }: { className?: string }) {
  return <span className={cn("rounded-full bg-muted px-2.5 py-1 text-[12px] font-semibold text-muted-foreground", className)}>Contoh data</span>;
}
