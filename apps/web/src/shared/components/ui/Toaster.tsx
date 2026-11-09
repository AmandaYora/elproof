import type { ComponentProps } from "react";
import { Toaster as KitToaster } from "@/shared/components/kit/sonner";

/**
 * Mounted once per layout. Toasts are announced politely and never take focus.
 * A layout with a fixed bottom bar (the client portal) passes `mobileOffset`
 * so toasts clear it.
 */
export function Toaster(props: ComponentProps<typeof KitToaster>) {
  return <KitToaster {...props} />;
}
