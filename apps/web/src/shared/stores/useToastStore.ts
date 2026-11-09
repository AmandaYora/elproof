import { toast as sonner } from "sonner";

// Errors stay a little longer than confirmations; a toast that offers "Urungkan"
// stays longest so the undo can actually be reached.
const LIFETIME_MS = { success: 4000, info: 5000, error: 7000, undo: 8000 } as const;

export interface ToastOptions {
  /**
   * Offers "Urungkan" on the toast. Call it only for actions that can really be
   * taken back (soft delete, status change), never for ones that cannot.
   */
  undo?: () => void;
}

function show(tone: "success" | "error" | "info", message: string, options?: ToastOptions) {
  const base = options?.undo
    ? {
        duration: LIFETIME_MS.undo,
        action: { label: "Urungkan", onClick: options.undo },
      }
    : { duration: LIFETIME_MS[tone] };
  sonner[tone](message, base);
}

/** Short confirmation of a finished action ("Peran disimpan"). */
export const toast = {
  success: (message: string, options?: ToastOptions) => show("success", message, options),
  error: (message: string) => show("error", message),
  info: (message: string) => show("info", message),
};
