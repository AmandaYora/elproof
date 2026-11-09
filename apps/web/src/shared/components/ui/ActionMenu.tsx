import { useEffect, useId, useRef, useState, type ReactNode } from "react";
import { createPortal } from "react-dom";
import { MoreHorizontal } from "lucide-react";
import { cn } from "@/shared/lib/cn";

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
 * A "⋯" button that opens a short list of secondary actions. Arrow keys move
 * between items, Escape closes the list (and only the list: a dialog the menu
 * sits in stays open), and focus returns to the button.
 */
export function ActionMenu({ items, label = "Aksi lainnya", disabled }: ActionMenuProps) {
  const visible = items.filter((i) => !i.hidden);
  const [open, setOpen] = useState(false);
  const [rect, setRect] = useState<{ top: number; right: number } | null>(null);
  const triggerRef = useRef<HTMLButtonElement>(null);
  const menuRef = useRef<HTMLDivElement>(null);
  const menuId = useId();

  function toggle() {
    if (open) {
      setOpen(false);
      return;
    }
    const r = triggerRef.current?.getBoundingClientRect();
    if (r) setRect({ top: r.bottom + 4, right: window.innerWidth - r.right });
    setOpen(true);
  }

  function close(returnFocus = true) {
    setOpen(false);
    if (returnFocus) triggerRef.current?.focus();
  }

  useEffect(() => {
    if (!open) return;
    menuRef.current?.querySelector<HTMLElement>('[role="menuitem"]')?.focus();
    const onMouseDown = (e: MouseEvent) => {
      const t = e.target as Node;
      if (triggerRef.current?.contains(t) || menuRef.current?.contains(t)) return;
      setOpen(false);
    };
    // A fixed-position list would drift away from its button on scroll.
    const onViewportChange = (e: Event) => {
      if (menuRef.current?.contains(e.target as Node)) return;
      setOpen(false);
    };
    document.addEventListener("mousedown", onMouseDown);
    window.addEventListener("scroll", onViewportChange, true);
    window.addEventListener("resize", onViewportChange);
    return () => {
      document.removeEventListener("mousedown", onMouseDown);
      window.removeEventListener("scroll", onViewportChange, true);
      window.removeEventListener("resize", onViewportChange);
    };
  }, [open]);

  function onMenuKeyDown(e: React.KeyboardEvent) {
    const entries = Array.from(menuRef.current?.querySelectorAll<HTMLElement>('[role="menuitem"]') ?? []);
    const index = entries.indexOf(document.activeElement as HTMLElement);
    if (e.key === "ArrowDown") {
      e.preventDefault();
      entries[(index + 1) % entries.length]?.focus();
    } else if (e.key === "ArrowUp") {
      e.preventDefault();
      entries[(index - 1 + entries.length) % entries.length]?.focus();
    } else if (e.key === "Home") {
      e.preventDefault();
      entries[0]?.focus();
    } else if (e.key === "End") {
      e.preventDefault();
      entries[entries.length - 1]?.focus();
    } else if (e.key === "Escape") {
      e.preventDefault();
      e.stopPropagation();
      close();
    } else if (e.key === "Tab") {
      close(false);
    }
  }

  if (visible.length === 0) return null;

  return (
    <>
      <button
        ref={triggerRef}
        type="button"
        disabled={disabled}
        aria-label={label}
        aria-haspopup="menu"
        aria-expanded={open}
        aria-controls={open ? menuId : undefined}
        onClick={toggle}
        className={cn(
          "inline-flex h-9 w-9 items-center justify-center rounded-md border border-border bg-white text-text-secondary transition-colors hover:bg-surface-muted",
          "focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-navy-900 disabled:opacity-50",
          open && "bg-surface-muted",
        )}
      >
        <MoreHorizontal className="h-4 w-4" />
      </button>
      {open &&
        rect &&
        createPortal(
          <div
            ref={menuRef}
            id={menuId}
            role="menu"
            aria-label={label}
            onKeyDown={onMenuKeyDown}
            style={{ position: "fixed", top: rect.top, right: rect.right }}
            className="z-50 min-w-48 overflow-hidden rounded-md border border-border bg-white py-1 shadow-lg"
          >
            {visible.map((item) => (
              <button
                key={item.label}
                type="button"
                role="menuitem"
                onClick={() => {
                  close(false);
                  item.onSelect();
                }}
                className={cn(
                  "flex w-full items-center gap-2.5 px-3 py-2.5 text-left text-[13px] font-medium hover:bg-surface-muted focus:bg-surface-muted focus:outline-none",
                  item.tone === "danger" ? "text-danger-text" : "text-text-primary",
                )}
              >
                {item.icon}
                {item.label}
              </button>
            ))}
          </div>,
          document.body,
        )}
    </>
  );
}
