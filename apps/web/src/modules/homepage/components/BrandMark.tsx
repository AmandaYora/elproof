import { Link } from "react-router-dom";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import { AppLogoMark } from "@/shared/components/AppLogoMark";
import { APP_NAME } from "@/shared/constants/brand";
import { cn } from "@/shared/lib/cn";

/** The ElProof logo on the public site and sign-in: the logo mark and the name. Links home. */
export function BrandMark({ className, inverted = false }: { className?: string; inverted?: boolean }) {
  return (
    <Link
      to={ROUTE_PATHS.homepage}
      className={cn(
        "inline-flex items-center gap-2.5 rounded-md align-top text-[18px] font-bold tracking-tight outline-none focus-visible:outline-2 focus-visible:outline-solid focus-visible:outline-offset-2",
        inverted ? "text-primary-foreground focus-visible:outline-primary-foreground" : "text-foreground focus-visible:outline-ring",
        className,
      )}
    >
      <AppLogoMark className={cn(inverted && "rounded-[9px] ring-1 ring-primary-foreground/40")} />
      {APP_NAME}
    </Link>
  );
}
