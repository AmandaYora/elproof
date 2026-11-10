import { cn } from "@/shared/lib/cn";

type CtaVariant = "primary" | "outline" | "inverted" | "outline-inverted";

const VARIANT: Record<CtaVariant, string> = {
  primary: "bg-primary text-primary-foreground hover:bg-primary-hover",
  outline: "border-input bg-transparent text-foreground hover:bg-muted",
  inverted: "bg-primary-foreground text-primary hover:bg-accent",
  "outline-inverted": "border-on-primary-muted/60 bg-transparent text-primary-foreground hover:bg-primary-foreground/10",
};

/**
 * Class for a link that looks like a button on the public site (pill, 48px on
 * touch). Real <button>s keep using shared/components/ui/Button.
 */
export function ctaClass(variant: CtaVariant = "primary", size: "md" | "sm" = "md", className?: string) {
  return cn(
    "inline-flex items-center justify-center gap-2 rounded-full border border-transparent font-semibold whitespace-nowrap transition-colors outline-none focus-visible:outline-2 focus-visible:outline-solid focus-visible:outline-offset-2 focus-visible:outline-ring",
    size === "md" ? "h-12 px-6 text-[15px]" : "h-10 px-4 text-[14px] pointer-coarse:h-11",
    VARIANT[variant],
    className,
  );
}
