import { cn } from "@/shared/lib/cn";
import { initialsFromName } from "@/shared/lib/formatters";
import { Avatar as KitAvatar, AvatarFallback } from "@/shared/components/kit/avatar";

interface AvatarProps {
  name: string;
  size?: "sm" | "md";
  /** Colors the circle (e.g. on a dark surface). */
  className?: string;
}

export function Avatar({ name, size = "sm", className }: AvatarProps) {
  return (
    <KitAvatar size={size === "sm" ? "sm" : "default"} aria-hidden>
      <AvatarFallback className={cn(size === "md" && "text-xs", className)}>{initialsFromName(name)}</AvatarFallback>
    </KitAvatar>
  );
}
