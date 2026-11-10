import { useRef, type ReactNode } from "react";
import { X } from "lucide-react";
import { cn } from "@/shared/lib/cn";
import { Button } from "@/shared/components/kit/button";
import { Dialog, DialogClose, DialogContent, DialogDescription, DialogTitle } from "@/shared/components/kit/dialog";

interface ModalProps {
  open: boolean;
  onClose: () => void;
  title: string;
  description?: string;
  children: ReactNode;
  footer?: ReactNode;
  /** xl: a wide table of several money columns. */
  size?: "sm" | "md" | "lg" | "xl";
  /** A save is running: Escape, outside clicks and the close button stop working. */
  busy?: boolean;
  /** Why the last save failed. Shown above the buttons, inside the modal. */
  error?: string | null;
}

const SIZE_CLASSES: Record<NonNullable<ModalProps["size"]>, string> = {
  sm: "sm:max-w-md",
  md: "sm:max-w-xl",
  lg: "sm:max-w-3xl",
  xl: "sm:max-w-5xl",
};

const FIRST_FIELD =
  "input:not([disabled]):not([type=hidden]):not([data-no-autofocus]), textarea:not([disabled]):not([data-no-autofocus]), select:not([disabled]):not([data-no-autofocus])";

/**
 * A centered dialog. Built on the kit Dialog (Base UI): focus moves in and
 * stays inside, Escape closes only the topmost dialog, the page behind does
 * not scroll, and focus returns to whatever opened it.
 */
export function Modal({ open, onClose, title, description, children, footer, size = "md", busy = false, error = null }: ModalProps) {
  const popupRef = useRef<HTMLDivElement>(null);

  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        if (!next && !busy) onClose();
      }}
    >
      <DialogContent
        ref={popupRef}
        showCloseButton={false}
        aria-busy={busy || undefined}
        // Prefer the first form control over the close button. An optional field marked
        // data-no-autofocus is skipped (on a phone focusing it opens the keyboard); with no other
        // field the dialog itself takes focus.
        initialFocus={() => {
          const popup = popupRef.current;
          const field = popup?.querySelector<HTMLElement>(FIRST_FIELD);
          if (field) return field;
          return popup?.querySelector("[data-no-autofocus]") ? popup : true;
        }}
        className={cn("flex w-full max-w-[calc(100%-2rem)] flex-col gap-0 p-0", SIZE_CLASSES[size])}
      >
        <header className="flex shrink-0 items-start justify-between gap-4 border-b px-6 py-4">
          <div className="min-w-0">
            <DialogTitle className="text-base leading-snug font-semibold">{title}</DialogTitle>
            {description && <DialogDescription className="mt-1 text-[13px]">{description}</DialogDescription>}
          </div>
          <DialogClose
            disabled={busy}
            render={<Button variant="ghost" size="icon-sm" className="-mt-1 -mr-2 shrink-0 pointer-coarse:size-11" />}
          >
            <X />
            <span className="sr-only">Tutup</span>
          </DialogClose>
        </header>
        <div className="min-h-0 flex-1 overflow-y-auto px-6 py-5">{children}</div>
        {error && (
          <p role="alert" className="mx-6 mb-4 rounded-md border border-danger/30 bg-danger-soft px-3.5 py-2.5 text-[13px] font-medium text-danger-text">
            {error}
          </p>
        )}
        {footer && <div className="flex shrink-0 justify-end gap-2 border-t px-6 py-4">{footer}</div>}
      </DialogContent>
    </Dialog>
  );
}
