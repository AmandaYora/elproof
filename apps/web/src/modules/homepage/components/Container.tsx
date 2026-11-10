import type { ReactNode } from "react";
import { cn } from "@/shared/lib/cn";

/** Page column of the public site: 1160px wide, 16px gutter on phones. */
export function Container({ children, className }: { children: ReactNode; className?: string }) {
  return <div className={cn("mx-auto w-full max-w-[1160px] px-4 sm:px-6", className)}>{children}</div>;
}
