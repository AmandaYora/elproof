import type { ReactElement } from "react";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/shared/components/kit/tooltip";

/** The name of an icon-only menu item, shown to the side. Loaded on demand: see NavMenu. */
export default function RailTooltip({ label, children }: { label: string; children: ReactElement }) {
  return (
    <Tooltip>
      <TooltipTrigger render={children} />
      <TooltipContent side="right">{label}</TooltipContent>
    </Tooltip>
  );
}
