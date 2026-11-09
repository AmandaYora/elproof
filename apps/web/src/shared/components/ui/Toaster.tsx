import { lazy, Suspense, useEffect, type ComponentProps } from "react";
import { markToasterReady } from "@/shared/stores/useToastStore";

const KitToaster = lazy(() => import("@/shared/components/kit/sonner").then((m) => ({ default: m.Toaster })));

/** Tells the toast helper a Toaster is mounted; rendered inside the lazy one so it only fires once Sonner is there. */
function Ready() {
  useEffect(() => markToasterReady(), []);
  return null;
}

/**
 * Mounted once per layout, loaded on demand. Toasts are announced politely and
 * never take focus. A layout with a fixed bottom bar (the client portal) passes
 * `mobileOffset` so toasts clear it.
 */
export function Toaster(props: ComponentProps<typeof import("@/shared/components/kit/sonner").Toaster>) {
  return (
    <Suspense fallback={null}>
      <KitToaster {...props} />
      <Ready />
    </Suspense>
  );
}
