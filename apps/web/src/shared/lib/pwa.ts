import { useSyncExternalStore } from "react";

/**
 * Registers the portal's service worker (production only, where it can be
 * served) and starts listening for the browser's install prompt. Failure to
 * register is silent: the portal works the same without it.
 */
export function registerPortalServiceWorker() {
  captureInstallPrompt();
  if (!import.meta.env.PROD || !("serviceWorker" in navigator)) return;
  void navigator.serviceWorker.register("/sw.js").catch(() => undefined);
}

/** The browser's "install" prompt event. */
export interface InstallPromptEvent extends Event {
  prompt: () => Promise<void>;
  userChoice: Promise<{ outcome: "accepted" | "dismissed" }>;
}

// Chrome fires `beforeinstallprompt` once, soon after load, so it is kept here
// (not in the page that shows the button, which may not be open yet).
let deferred: InstallPromptEvent | null = null;
let listening = false;
const subscribers = new Set<() => void>();

function emit() {
  subscribers.forEach((notify) => notify());
}

function captureInstallPrompt() {
  if (listening) return;
  listening = true;
  window.addEventListener("beforeinstallprompt", (e) => {
    e.preventDefault();
    deferred = e as InstallPromptEvent;
    emit();
  });
  window.addEventListener("appinstalled", () => {
    deferred = null;
    emit();
  });
}

/** The saved install prompt, or null when none is available (or already used). */
export function useInstallPrompt(): { event: InstallPromptEvent | null; consume: () => void } {
  const event = useSyncExternalStore(
    (notify) => {
      subscribers.add(notify);
      return () => subscribers.delete(notify);
    },
    () => deferred,
    () => null,
  );
  return {
    event,
    consume: () => {
      deferred = null;
      emit();
    },
  };
}

/** True when the app is already running from the home screen. */
export function isInstalledApp(): boolean {
  return window.matchMedia("(display-mode: standalone)").matches || (navigator as Navigator & { standalone?: boolean }).standalone === true;
}

/** iPhone/iPad Safari has no install prompt: it needs the Share-menu steps. */
export function isIOS(): boolean {
  return /iphone|ipad|ipod/i.test(navigator.userAgent);
}
