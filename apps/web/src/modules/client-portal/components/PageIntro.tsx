import type { ReactNode } from "react";

/** Every portal page opens the same way: a 20/600 title without an icon and, when it adds something, one line under it. */
export function PageIntro({ title, children }: { title: string; children?: ReactNode }) {
  return (
    <header>
      <h1 className="text-xl font-semibold text-foreground">{title}</h1>
      {children && <p className="mt-1.5 max-w-prose text-[14px] leading-relaxed text-muted-foreground">{children}</p>}
    </header>
  );
}
