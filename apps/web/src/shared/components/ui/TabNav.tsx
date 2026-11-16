import { NavLink } from "react-router-dom";
import { cn } from "@/shared/lib/cn";

interface TabItem {
  to: string;
  label: string;
  end?: boolean;
}

interface TabNavProps {
  items: TabItem[];
  className?: string;
  sticky?: boolean;
  /** "underline" (default): a page's tabs. "pills": a quieter switch between views inside a tab. */
  variant?: "underline" | "pills";
  "aria-label"?: string;
}

export function TabNav({ items, className, sticky = true, variant = "underline", "aria-label": ariaLabel }: TabNavProps) {
  if (variant === "pills") {
    return (
      <nav aria-label={ariaLabel} className={cn("flex gap-1 overflow-x-auto", className)}>
        {items.map((item) => (
          <NavLink
            key={item.to}
            to={item.to}
            end={item.end}
            className={({ isActive }) =>
              cn(
                "inline-flex h-8 shrink-0 items-center rounded-md px-3 text-[13px] font-medium transition-colors focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-solid focus-visible:outline-ring",
                isActive ? "bg-accent text-accent-foreground" : "text-muted-foreground hover:bg-muted hover:text-foreground",
              )
            }
          >
            {item.label}
          </NavLink>
        ))}
      </nav>
    );
  }
  return (
    <nav
      aria-label={ariaLabel}
      className={cn(
        "flex gap-6 overflow-x-auto border-b border-border bg-background/95 px-1 backdrop-blur",
        sticky && "sticky top-16 z-10",
        className
      )}
    >
      {items.map((item) => (
        <NavLink
          key={item.to}
          to={item.to}
          end={item.end}
          className={({ isActive }) =>
            cn(
              // The nav scrolls sideways, which clips an outside outline: the focus ring is drawn inside the tab.
              "shrink-0 border-b-2 py-3 text-[13.5px] font-medium transition-colors focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-solid focus-visible:outline-ring",
              isActive ? "border-primary font-semibold text-primary" : "border-transparent text-text-secondary hover:text-text-primary"
            )
          }
        >
          {item.label}
        </NavLink>
      ))}
    </nav>
  );
}
