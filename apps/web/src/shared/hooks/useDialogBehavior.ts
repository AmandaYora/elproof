import { useEffect, useRef, type RefObject } from "react";

const FOCUSABLE =
  'a[href], button:not([disabled]), input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])';

// Dialogs that are open right now, oldest first. Only the last one reacts to
// Escape and Tab, so a confirm dialog opened over a form closes by itself
// instead of taking the form down with it.
const openDialogs: symbol[] = [];
// The page's own overflow value, remembered while any dialog is open.
let savedOverflow = "";

interface DialogBehaviorOptions {
  open: boolean;
  onClose: () => void;
  /** While true (a save is running) Escape does nothing. */
  busy?: boolean;
}

/**
 * Shared keyboard behavior for Modal and Drawer: focus moves into the panel
 * when it opens, Tab stays inside it, Escape closes it (unless busy), the page
 * behind does not scroll, and focus goes back to whatever opened it.
 */
export function useDialogBehavior<T extends HTMLElement>({ open, onClose, busy = false }: DialogBehaviorOptions): RefObject<T | null> {
  const panelRef = useRef<T>(null);
  // Latest values without re-running the effect (which would steal focus
  // back to the first field on every keystroke-driven re-render).
  const onCloseRef = useRef(onClose);
  const busyRef = useRef(busy);
  useEffect(() => {
    onCloseRef.current = onClose;
    busyRef.current = busy;
  });

  useEffect(() => {
    if (!open) return;
    const id = Symbol("dialog");
    if (openDialogs.length === 0) savedOverflow = document.body.style.overflow;
    openDialogs.push(id);
    const opener = document.activeElement as HTMLElement | null;
    const panel = panelRef.current;

    // Prefer the first form control over the close button.
    const firstField = panel?.querySelector<HTMLElement>(
      "input:not([disabled]):not([type=hidden]), textarea:not([disabled]), select:not([disabled])",
    );
    (firstField ?? panel?.querySelector<HTMLElement>(FOCUSABLE))?.focus();

    const onKey = (e: KeyboardEvent) => {
      if (openDialogs[openDialogs.length - 1] !== id) return;
      if (e.key === "Escape") {
        if (e.defaultPrevented) return;
        e.stopPropagation();
        if (!busyRef.current) onCloseRef.current();
        return;
      }
      if (e.key !== "Tab" || !panel) return;
      const items = Array.from(panel.querySelectorAll<HTMLElement>(FOCUSABLE)).filter((el) => el.offsetParent !== null);
      if (items.length === 0) return;
      const first = items[0];
      const last = items[items.length - 1];
      const active = document.activeElement;
      if (!panel.contains(active)) {
        e.preventDefault();
        first.focus();
      } else if (e.shiftKey && active === first) {
        e.preventDefault();
        last.focus();
      } else if (!e.shiftKey && active === last) {
        e.preventDefault();
        first.focus();
      }
    };
    document.addEventListener("keydown", onKey);
    document.body.style.overflow = "hidden";
    return () => {
      document.removeEventListener("keydown", onKey);
      const index = openDialogs.indexOf(id);
      if (index >= 0) openDialogs.splice(index, 1);
      // Unlock the page only when the last dialog closes.
      if (openDialogs.length === 0) document.body.style.overflow = savedOverflow;
      opener?.focus?.();
    };
  }, [open]);

  return panelRef;
}
