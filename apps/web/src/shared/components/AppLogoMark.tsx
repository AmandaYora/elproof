import logoMark from "@/assets/brand/elproof-mark.webp";
import { cn } from "@/shared/lib/cn";

/**
 * ElProof's own logo mark: the navy rounded square with the heart, ring and
 * checklist. The one place the app's logo is drawn (public site, sign-in,
 * platform console, plan cards); a WO's own logo is tenant branding, not this.
 * Decorative by default because it always sits next to the name "ElProof";
 * pass `alt` when it stands alone.
 */
export function AppLogoMark({ className, alt }: { className?: string; alt?: string }) {
  return (
    <img
      src={logoMark}
      alt={alt ?? ""}
      aria-hidden={alt ? undefined : true}
      width={32}
      height={32}
      decoding="async"
      draggable={false}
      className={cn("size-8 shrink-0 select-none", className)}
    />
  );
}
