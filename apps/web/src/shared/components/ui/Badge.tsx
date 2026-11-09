import type { ReactNode } from "react";
import { Circle, CircleAlert, CircleCheck, Info, TriangleAlert, type LucideIcon } from "lucide-react";
import { Badge as KitBadge } from "@/shared/components/kit/badge";

export type BadgeTone = "neutral" | "success" | "warning" | "danger" | "info" | "navy";

const VARIANT = { neutral: "neutral", success: "success", warning: "warning", danger: "danger", info: "info", navy: "brand" } as const;

// Status is never color alone (DESIGN.md T5): each tone carries its own icon
// next to the text, in the same -text color.
const ICON: Record<BadgeTone, LucideIcon> = {
  neutral: Circle,
  success: CircleCheck,
  warning: TriangleAlert,
  danger: CircleAlert,
  info: Info,
  navy: Circle,
};

interface BadgeProps {
  tone?: BadgeTone;
  /** Shows the tone's icon before the text (default). Pass false for a plain label. */
  dot?: boolean;
  children: ReactNode;
  className?: string;
}

export function Badge({ tone = "neutral", dot = true, children, className }: BadgeProps) {
  const Icon = ICON[tone];
  return (
    <KitBadge variant={VARIANT[tone]} className={className}>
      {dot && <Icon aria-hidden />}
      {children}
    </KitBadge>
  );
}
