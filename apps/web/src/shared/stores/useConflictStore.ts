import { create } from "zustand";

// "Someone else changed this" (ADR-0036). The HTTP client opens the dialog when a save is refused with 409; the page that
// owns the data registers how to load it again. Kept free of the HTTP client so that one can import this.

interface ConflictState {
  open: boolean;
  show: () => void;
  dismiss: () => void;
}

export const useConflictStore = create<ConflictState>((set) => ({
  open: false,
  show: () => set({ open: true }),
  dismiss: () => set({ open: false }),
}));

const revalidators = new Set<() => void>();

/** A page that holds data which can go stale says how to reload it. Returns the function that removes it. */
export function registerRevalidator(reload: () => void): () => void {
  revalidators.add(reload);
  return () => {
    revalidators.delete(reload);
  };
}

/** Reloads what the page registered; with nothing registered the whole page is reloaded, which is always correct. */
export function revalidateAll(): void {
  if (revalidators.size === 0) {
    window.location.reload();
    return;
  }
  revalidators.forEach((reload) => reload());
}
