import { cn } from "@/shared/lib/cn";
import { initialsFromName } from "@/shared/lib/formatters";

interface AvatarProps {
  name: string;
  size?: "sm" | "md";
  className?: string;
}

export function Avatar({ name, size = "sm", className }: AvatarProps) {
  return (
    <span
      className={cn(
        "inline-flex shrink-0 items-center justify-center rounded-full bg-navy-900/10 font-semibold text-navy-900",
        size === "sm" ? "h-6 w-6 text-[10px]" : "h-8 w-8 text-xs",
        className
      )}
    >
      {initialsFromName(name)}
    </span>
  );
}
