import type { ReactNode } from "react";
import { Link } from "react-router-dom";
import { LoginPhotoSlider } from "@/modules/auth/components/LoginPhotoSlider";
import { ProofSeal } from "@/modules/homepage/components/ProofSeal";
import { ROUTE_PATHS } from "@/app/routes/route-paths";

/**
 * Two-panel frame for the signup steps: the same photo panel as the login
 * page on the left (large screens only), the form on the right.
 */
export function AuthSplitLayout({ aside, children }: { aside: ReactNode; children: ReactNode }) {
  return (
    <div className="flex min-h-screen">
      <div className="relative hidden overflow-hidden bg-gradient-to-br from-slate-900 to-slate-800 lg:flex lg:w-1/2 lg:flex-col lg:justify-end lg:px-16 lg:pb-16">
        <LoginPhotoSlider />
        <div className="pointer-events-none absolute inset-0 bg-gradient-to-t from-slate-950/95 via-slate-950/40 to-transparent" />
        <div className="relative z-10 max-w-md">{aside}</div>
      </div>
      <div className="flex w-full flex-col items-center justify-center bg-background px-4 py-10 sm:px-6 lg:w-1/2">
        <Link to={ROUTE_PATHS.homepage} className="mb-6 flex items-center gap-2 text-text-primary lg:hidden">
          <ProofSeal size={28} className="text-navy-900" />
          <span className="text-[17px] font-bold tracking-tight">ElProof</span>
        </Link>
        <div className="w-full max-w-md">{children}</div>
      </div>
    </div>
  );
}
