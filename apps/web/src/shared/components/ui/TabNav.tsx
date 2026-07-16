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
}

export function TabNav({ items, className, sticky = true }: TabNavProps) {
  return (
    <nav
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
              "shrink-0 border-b-2 py-3 text-[13.5px] font-medium transition-colors",
              isActive ? "border-navy-900 font-semibold text-navy-900" : "border-transparent text-text-secondary hover:text-text-primary"
            )
          }
        >
          {item.label}
        </NavLink>
      ))}
    </nav>
  );
}
