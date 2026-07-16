import type { ReactNode } from "react";
import { useEffect } from "react";
import { X } from "lucide-react";
import { cn } from "@/shared/lib/cn";

interface ModalProps {
  open: boolean;
  onClose: () => void;
  title: string;
  description?: string;
  children: ReactNode;
  footer?: ReactNode;
  size?: "sm" | "md" | "lg";
}

const SIZE_CLASSES: Record<NonNullable<ModalProps["size"]>, string> = {
  sm: "max-w-md",
  md: "max-w-xl",
  lg: "max-w-3xl",
};

export function Modal({ open, onClose, title, description, children, footer, size = "md" }: ModalProps) {
  useEffect(() => {
    if (!open) return;
    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    document.addEventListener("keydown", onKeyDown);
    return () => document.removeEventListener("keydown", onKeyDown);
  }, [open, onClose]);

  if (!open) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-start justify-center overflow-y-auto bg-navy-950/50 px-4 py-10">
      <div
        role="dialog"
        aria-modal="true"
        aria-label={title}
        className={cn("w-full rounded-lg border border-border bg-surface shadow-xl", SIZE_CLASSES[size])}
      >
        <div className="flex items-start justify-between gap-4 border-b border-border-light px-6 py-4">
          <div>
            <h2 className="text-base font-semibold text-text-primary">{title}</h2>
            {description && <p className="mt-1 text-[13px] text-text-secondary">{description}</p>}
          </div>
          <button
            onClick={onClose}
            aria-label="Tutup"
            className="rounded-md p-1 text-text-secondary hover:bg-surface-muted focus-visible:outline focus-visible:outline-2 focus-visible:outline-navy-900"
          >
            <X className="h-4 w-4" />
          </button>
        </div>
        <div className="max-h-[65vh] overflow-y-auto px-6 py-5">{children}</div>
        {footer && <div className="flex justify-end gap-2 border-t border-border-light px-6 py-4">{footer}</div>}
      </div>
    </div>
  );
}
