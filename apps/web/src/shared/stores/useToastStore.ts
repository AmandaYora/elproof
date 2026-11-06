import { create } from "zustand";

export type ToastTone = "success" | "error" | "info";

export interface ToastItem {
  id: number;
  tone: ToastTone;
  message: string;
}

interface ToastState {
  toasts: ToastItem[];
  show: (tone: ToastTone, message: string) => void;
  dismiss: (id: number) => void;
}

let nextId = 1;
const LIFETIME_MS = { success: 4000, info: 5000, error: 7000 } as const;

export const useToastStore = create<ToastState>((set, get) => ({
  toasts: [],
  show: (tone, message) => {
    const id = nextId++;
    // At most three at once; the oldest goes first.
    set((s) => ({ toasts: [...s.toasts.slice(-2), { id, tone, message }] }));
    window.setTimeout(() => get().dismiss(id), LIFETIME_MS[tone]);
  },
  dismiss: (id) => set((s) => ({ toasts: s.toasts.filter((t) => t.id !== id) })),
}));

/** Short confirmation of a finished action ("Peran disimpan"). */
export const toast = {
  success: (message: string) => useToastStore.getState().show("success", message),
  error: (message: string) => useToastStore.getState().show("error", message),
  info: (message: string) => useToastStore.getState().show("info", message),
};
