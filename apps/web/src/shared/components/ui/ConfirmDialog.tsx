import { useEffect, useRef, useState, type ReactNode } from "react";
import { Modal } from "@/shared/components/ui/Modal";
import { Button } from "@/shared/components/ui/Button";
import { Field, Input } from "@/shared/components/ui/Input";

interface ConfirmDialogProps {
  open: boolean;
  title: string;
  /** What happens, in plain words — including what it affects. */
  children: ReactNode;
  /** Button label naming the action, e.g. "Bekukan tenant". */
  confirmLabel: string;
  tone?: "danger" | "primary";
  /**
   * For actions that are hard to undo: the user must type this exact text
   * (e.g. the tenant's name) before the action button enables.
   */
  confirmText?: string;
  isSubmitting?: boolean;
  /** Failure from the action, shown here so the dialog stays in context. */
  error?: string | null;
  onConfirm: () => void;
  onClose: () => void;
}

export function ConfirmDialog({
  open,
  title,
  children,
  confirmLabel,
  tone = "danger",
  confirmText,
  isSubmitting = false,
  error,
  onConfirm,
  onClose,
}: ConfirmDialogProps) {
  const [typed, setTyped] = useState("");
  const cancelRef = useRef<HTMLButtonElement>(null);

  useEffect(() => {
    if (!open) return;
    setTyped("");
    // Safe default: focus lands on "Batal", never on the destructive action.
    const id = requestAnimationFrame(() => cancelRef.current?.focus());
    return () => cancelAnimationFrame(id);
  }, [open]);

  const matches = !confirmText || typed.trim() === confirmText.trim();

  function handleClose() {
    if (!isSubmitting) onClose();
  }

  return (
    <Modal
      open={open}
      onClose={handleClose}
      title={title}
      size="sm"
      footer={
        <>
          <Button ref={cancelRef} variant="secondary" onClick={handleClose} disabled={isSubmitting}>
            Batal
          </Button>
          <Button variant={tone} onClick={onConfirm} disabled={!matches || isSubmitting}>
            {isSubmitting ? "Memproses..." : confirmLabel}
          </Button>
        </>
      }
    >
      <div className="flex flex-col gap-4 text-[13.5px] leading-relaxed text-text-secondary">
        {error && (
          <p role="alert" className="rounded-md border border-danger/30 bg-danger-soft px-3.5 py-2.5 text-[13px] font-medium text-danger">
            {error}
          </p>
        )}
        <div>{children}</div>
        {confirmText && (
          <Field label={`Ketik "${confirmText}" untuk melanjutkan`}>
            <Input
              value={typed}
              onChange={(e) => setTyped(e.target.value)}
              autoComplete="off"
              spellCheck={false}
              onKeyDown={(e) => e.key === "Enter" && matches && !isSubmitting && onConfirm()}
            />
          </Field>
        )}
      </div>
    </Modal>
  );
}
