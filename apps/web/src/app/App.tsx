import { useEffect } from "react";
import { RouterProvider } from "react-router-dom";
import { router } from "@/app/routes";
import { hasInterruptedSupportSession } from "@/shared/stores/useAuthStore";

export default function App() {
  useEffect(() => {
    // Rare (a reload during a support session), so its code loads on demand.
    if (hasInterruptedSupportSession()) void import("@/shared/lib/impersonation").then((m) => m.finishInterruptedSupportSession());
  }, []);
  return <RouterProvider router={router} />;
}
