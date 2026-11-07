import type { ReactNode } from "react";
import { X } from "lucide-react";
import { cn } from "@/shared/lib/cn";
import { useDialogBehavior } from "@/shared/hooks/useDialogBehavior";

interface DrawerProps {
  open: boolean;
  onClose: () => void;
  title: string;
  description?: string;
  children: ReactNode;
  footer?: ReactNode;
  size?: "md" | "lg";
  /** A save is running: Escape does nothing. */
  busy?: boolean;
}

const SIZE: Record<NonNullable<DrawerProps["size"]>, string> = {
  md: "sm:max-w-md",
  lg: "sm:max-w-2xl",
};

/**
 * A panel sliding in from the right for editing something without leaving
 * the page (PLAN.md Fase 3b). Full width on phones. Focus moves into the
 * panel, Tab stays inside it, Escape closes it, and focus returns to the
 * element that opened it (shared with Modal via useDialogBehavior).
 */
export function Drawer({ open, onClose, title, description, children, footer, size = "md", busy = false }: DrawerProps) {
  const panelRef = useDialogBehavior<HTMLDivElement>({ open, onClose, busy });

  if (!open) return null;

  return (
    <div className="fixed inset-0 z-50 flex justify-end">
      <div className="absolute inset-0 bg-navy-950/40 motion-safe:animate-[fade-in_150ms_ease-out]" onClick={busy ? undefined : onClose} aria-hidden />
      <div
        ref={panelRef}
        role="dialog"
        aria-modal="true"
        aria-label={title}
        className={cn(
          "relative flex h-full w-full flex-col border-l border-border bg-surface shadow-2xl motion-safe:animate-[drawer-in_200ms_ease-out]",
          SIZE[size],
        )}
      >
        <header className="flex items-start justify-between gap-4 border-b border-border-light px-[var(--density-pad-x)] py-4">
          <div className="min-w-0">
            <h2 className="text-base font-semibold text-text-primary">{title}</h2>
            {description && <p className="mt-1 text-[13px] leading-relaxed text-text-secondary">{description}</p>}
          </div>
          <button
            type="button"
            onClick={onClose}
            disabled={busy}
            aria-label="Tutup"
            className="flex h-11 w-11 items-center justify-center rounded-md text-text-secondary hover:bg-surface-muted focus-visible:outline focus-visible:outline-2 focus-visible:outline-navy-900 disabled:opacity-40 sm:h-8 sm:w-8"
          >
            <X className="h-4 w-4" />
          </button>
        </header>
        <div className="flex-1 overflow-y-auto px-[var(--density-pad-x)] py-5">{children}</div>
        {footer && <footer className="flex justify-end gap-2 border-t border-border-light px-[var(--density-pad-x)] py-4">{footer}</footer>}
      </div>
    </div>
  );
}
