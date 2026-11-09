import { RouterProvider } from "react-router-dom";
import { router } from "@/app/routes";
import { TooltipProvider } from "@/shared/components/kit/tooltip";

export default function App() {
  return (
    <TooltipProvider>
      <RouterProvider router={router} />
    </TooltipProvider>
  );
}
