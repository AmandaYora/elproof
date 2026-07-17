import type { ReactNode } from "react";
import { useRevealOnScroll } from "@/modules/homepage/hooks/useRevealOnScroll";
import { cn } from "@/shared/lib/cn";

export function Reveal({ children, delay = 0, className }: { children: ReactNode; delay?: number; className?: string }) {
  const { ref, visible } = useRevealOnScroll<HTMLDivElement>();
  return (
    <div
      ref={ref}
      className={cn(
        "motion-safe:transition-[opacity,transform] motion-safe:duration-700 motion-safe:ease-out",
        visible ? "motion-safe:translate-y-0 motion-safe:opacity-100" : "motion-safe:translate-y-4 motion-safe:opacity-0",
        className
      )}
      style={{ transitionDelay: visible ? `${delay}ms` : "0ms" }}
    >
      {children}
    </div>
  );
}
