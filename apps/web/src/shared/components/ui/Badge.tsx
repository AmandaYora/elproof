import type { ReactNode } from "react";
import { cn } from "@/shared/lib/cn";

export type BadgeTone = "neutral" | "success" | "warning" | "danger" | "info" | "navy";

const TONE_CLASSES: Record<BadgeTone, string> = {
  neutral: "bg-neutral-soft text-text-secondary",
  success: "bg-success-soft text-success",
  warning: "bg-warning-soft text-warning",
  danger: "bg-danger-soft text-danger",
  info: "bg-info-soft text-info",
  navy: "bg-navy-900/10 text-navy-900",
};

const DOT_CLASSES: Record<BadgeTone, string> = {
  neutral: "bg-text-secondary",
  success: "bg-success",
  warning: "bg-warning",
  danger: "bg-danger",
  info: "bg-info",
  navy: "bg-navy-900",
};

interface BadgeProps {
  tone?: BadgeTone;
  dot?: boolean;
  children: ReactNode;
  className?: string;
}

export function Badge({ tone = "neutral", dot = true, children, className }: BadgeProps) {
  return (
    <span
      className={cn(
        "inline-flex items-center gap-1.5 rounded-full px-2.5 py-1 text-[11px] font-semibold uppercase tracking-wide whitespace-nowrap",
        TONE_CLASSES[tone],
        className
      )}
    >
      {dot && <span className={cn("h-1.5 w-1.5 rounded-full", DOT_CLASSES[tone])} />}
      {children}
    </span>
  );
}
