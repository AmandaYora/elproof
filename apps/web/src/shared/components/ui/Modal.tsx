import type { ReactNode } from "react";
import { useId } from "react";
import { X } from "lucide-react";
import { cn } from "@/shared/lib/cn";
import { useDialogBehavior } from "@/shared/hooks/useDialogBehavior";

interface ModalProps {
  open: boolean;
  onClose: () => void;
  title: string;
  description?: string;
  children: ReactNode;
  footer?: ReactNode;
  size?: "sm" | "md" | "lg";
  /** A save is running: Escape and the close button stop working. */
  busy?: boolean;
  /** Why the last save failed. Shown above the buttons, inside the modal. */
  error?: string | null;
}

const SIZE_CLASSES: Record<NonNullable<ModalProps["size"]>, string> = {
  sm: "max-w-md",
  md: "max-w-xl",
  lg: "max-w-3xl",
};

export function Modal({ open, onClose, title, description, children, footer, size = "md", busy = false, error = null }: ModalProps) {
  const panelRef = useDialogBehavior<HTMLDivElement>({ open, onClose, busy });
  const titleId = useId();
  const descriptionId = useId();

  if (!open) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-start justify-center overflow-y-auto bg-navy-950/50 px-4 py-10">
      <div
        ref={panelRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        aria-describedby={description ? descriptionId : undefined}
        aria-busy={busy || undefined}
        className={cn("w-full rounded-lg border border-border bg-surface shadow-xl", SIZE_CLASSES[size])}
      >
        <div className="flex items-start justify-between gap-4 border-b border-border-light px-6 py-4">
          <div>
            <h2 id={titleId} className="text-base font-semibold text-text-primary">
              {title}
            </h2>
            {description && (
              <p id={descriptionId} className="mt-1 text-[13px] text-text-secondary">
                {description}
              </p>
            )}
          </div>
          <button
            type="button"
            onClick={onClose}
            disabled={busy}
            aria-label="Tutup"
            className="-m-1.5 flex h-11 w-11 shrink-0 items-center justify-center rounded-md text-text-secondary hover:bg-surface-muted focus-visible:outline focus-visible:outline-2 focus-visible:outline-navy-900 disabled:opacity-40 sm:m-0 sm:h-8 sm:w-8"
          >
            <X className="h-4 w-4" />
          </button>
        </div>
        <div className="max-h-[65vh] overflow-y-auto px-6 py-5">{children}</div>
        {error && (
          <p role="alert" className="mx-6 mb-4 rounded-md border border-danger/30 bg-danger-soft px-3.5 py-2.5 text-[13px] font-medium text-danger">
            {error}
          </p>
        )}
        {footer && <div className="flex justify-end gap-2 border-t border-border-light px-6 py-4">{footer}</div>}
      </div>
    </div>
  );
}
