import type { ReactNode } from "react";
import { cn } from "@/shared/lib/cn";

interface PageHeaderProps {
  title: ReactNode;
  /** One sentence, only when it tells something the title does not. */
  description?: ReactNode;
  /** The page's actions, primary last (it sits at the right edge). */
  actions?: ReactNode;
  /** Above the title: breadcrumbs. */
  eyebrow?: ReactNode;
  className?: string;
}

/**
 * The top of every console page (DESIGN.md §4.5 type scale): the page title at 24/600, an optional sentence, and the
 * page's actions at the right. One component so every page starts the same way.
 */
export function PageHeader({ title, description, actions, eyebrow, className }: PageHeaderProps) {
  return (
    <div className={cn("mb-6 flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between", className)}>
      <div className="min-w-0">
        {eyebrow && <div className="mb-2">{eyebrow}</div>}
        <h1 className="text-[24px] leading-tight font-semibold tracking-[-0.02em] text-foreground">{title}</h1>
        {description && <p className="mt-1.5 max-w-[70ch] text-[14px] text-muted-foreground">{description}</p>}
      </div>
      {actions && <div className="flex flex-wrap items-center gap-2 sm:shrink-0 sm:justify-end">{actions}</div>}
    </div>
  );
}
