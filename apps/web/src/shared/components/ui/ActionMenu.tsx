import type { ReactNode } from "react";
import { MoreHorizontal } from "lucide-react";
import { Button } from "@/shared/components/kit/button";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/shared/components/kit/dropdown-menu";

export interface ActionMenuItem {
  label: string;
  icon?: ReactNode;
  onSelect: () => void;
  /** Destructive actions are drawn in the danger color. */
  tone?: "default" | "danger";
  /** Left out of the menu entirely (e.g. no permission). */
  hidden?: boolean;
}

interface ActionMenuProps {
  items: ActionMenuItem[];
  /** Accessible name of the "⋯" button. */
  label?: string;
  disabled?: boolean;
}

/**
 * A "⋯" button that opens a short list of secondary actions. Built on the kit
 * DropdownMenu (Base UI): arrow keys move between items, Escape closes the
 * list (and only the list: a dialog the menu sits in stays open), and focus
 * returns to the button.
 */
export function ActionMenu({ items, label = "Aksi lainnya", disabled }: ActionMenuProps) {
  const visible = items.filter((i) => !i.hidden);
  if (visible.length === 0) return null;

  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        disabled={disabled}
        aria-label={label}
        render={<Button variant="outline" size="icon" className="bg-card text-muted-foreground" />}
      >
        <MoreHorizontal />
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" aria-label={label}>
        {visible.map((item) => (
          <DropdownMenuItem
            key={item.label}
            variant={item.tone === "danger" ? "destructive" : "default"}
            onClick={item.onSelect}
            className="text-[13px] font-medium"
          >
            {item.icon}
            {item.label}
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
