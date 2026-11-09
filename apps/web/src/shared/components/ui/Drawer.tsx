import { useRef, type ReactNode } from "react";
import { X } from "lucide-react";
import { Button } from "@/shared/components/kit/button";
import { Sheet, SheetClose, SheetContent, SheetDescription, SheetTitle } from "@/shared/components/kit/sheet";

interface DrawerProps {
  open: boolean;
  onClose: () => void;
  title: string;
  description?: string;
  children: ReactNode;
  footer?: ReactNode;
  size?: "md" | "lg";
  /** A save is running: Escape, outside clicks and the close button do nothing. */
  busy?: boolean;
}

const SIZE: Record<NonNullable<DrawerProps["size"]>, string> = {
  md: "data-[side=right]:sm:max-w-md",
  lg: "data-[side=right]:sm:max-w-2xl",
};

const FIRST_FIELD = "input:not([disabled]):not([type=hidden]), textarea:not([disabled]), select:not([disabled])";

/**
 * A panel sliding in from the right for editing something without leaving
 * the page (PLAN.md Fase 3b). Full width on phones. Built on the kit Sheet
 * (Base UI), so focus handling and Escape match Modal.
 */
export function Drawer({ open, onClose, title, description, children, footer, size = "md", busy = false }: DrawerProps) {
  const popupRef = useRef<HTMLDivElement>(null);

  return (
    <Sheet
      open={open}
      onOpenChange={(next) => {
        if (!next && !busy) onClose();
      }}
    >
      <SheetContent
        ref={popupRef}
        side="right"
        showCloseButton={false}
        aria-busy={busy || undefined}
        initialFocus={() => popupRef.current?.querySelector<HTMLElement>(FIRST_FIELD) ?? true}
        className={`gap-0 ${SIZE[size]}`}
      >
        <header className="flex shrink-0 items-start justify-between gap-4 border-b px-[var(--density-pad-x)] py-4">
          <div className="min-w-0">
            <SheetTitle className="text-base leading-snug font-semibold">{title}</SheetTitle>
            {description && <SheetDescription className="mt-1 text-[13px] leading-relaxed">{description}</SheetDescription>}
          </div>
          <SheetClose
            disabled={busy}
            render={<Button variant="ghost" size="icon-sm" className="-mt-1 -mr-2 shrink-0 pointer-coarse:size-11" />}
          >
            <X />
            <span className="sr-only">Tutup</span>
          </SheetClose>
        </header>
        <div className="min-h-0 flex-1 overflow-y-auto px-[var(--density-pad-x)] py-5">{children}</div>
        {footer && <footer className="flex shrink-0 justify-end gap-2 border-t px-[var(--density-pad-x)] py-4">{footer}</footer>}
      </SheetContent>
    </Sheet>
  );
}
