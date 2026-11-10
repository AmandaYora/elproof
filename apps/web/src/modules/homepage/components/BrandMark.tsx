import { Link } from "react-router-dom";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import { APP_NAME } from "@/shared/constants/brand";
import { cn } from "@/shared/lib/cn";

/** The ElProof logo on the public site: the round "EP" mark and the name. Links home. */
export function BrandMark({ className, inverted = false }: { className?: string; inverted?: boolean }) {
  return (
    <Link
      to={ROUTE_PATHS.homepage}
      className={cn(
        "inline-flex items-center gap-2.5 rounded-md text-[18px] font-bold tracking-tight outline-none focus-visible:outline-2 focus-visible:outline-solid focus-visible:outline-offset-2 focus-visible:outline-ring",
        inverted ? "text-primary-foreground" : "text-foreground",
        className,
      )}
    >
      <span
        aria-hidden
        className={cn(
          "grid size-8 place-items-center rounded-full text-[11.5px] font-bold",
          inverted ? "bg-primary-foreground text-primary" : "bg-primary text-primary-foreground",
        )}
      >
        EP
      </span>
      {APP_NAME}
    </Link>
  );
}
