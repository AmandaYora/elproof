import type { LucideIcon } from "lucide-react";
import { cn } from "@/shared/lib/cn";
import { useWriteLocked } from "@/shared/stores/usePermissionStore";
import { Button } from "@/shared/components/kit/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/shared/components/kit/tooltip";

export type IconActionTone = "neutral" | "info" | "success" | "danger" | "brand";

const TONE_CLASSES: Record<IconActionTone, string> = {
  neutral: "bg-neutral-soft text-muted-foreground hover:bg-border hover:text-foreground",
  info: "bg-info-soft text-info-text hover:bg-info/20",
  success: "bg-success-soft text-success-text hover:bg-success/20",
  danger: "bg-danger-soft text-danger-text hover:bg-danger/20",
  brand: "bg-accent text-accent-foreground hover:bg-primary/20",
};

interface IconActionButtonProps {
  icon: LucideIcon;
  label: string;
  tone?: IconActionTone;
  onClick?: () => void;
  disabled?: boolean;
  /** Only looks at data. Every other action disappears while the workspace is read-only. */
  readOnly?: boolean;
}

/** A row action shown as an icon: the label is the accessible name and the tooltip. */
export function IconActionButton({ icon: Icon, label, tone = "neutral", onClick, disabled = false, readOnly = false }: IconActionButtonProps) {
  const writeLocked = useWriteLocked();
  if (writeLocked && !readOnly) return null;
  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <Button
            type="button"
            variant="ghost"
            size="icon-sm"
            onClick={onClick}
            disabled={disabled}
            aria-label={label}
            className={cn("shrink-0 disabled:opacity-40", TONE_CLASSES[tone])}
          />
        }
      >
        <Icon className="size-4" />
      </TooltipTrigger>
      <TooltipContent>{label}</TooltipContent>
    </Tooltip>
  );
}
