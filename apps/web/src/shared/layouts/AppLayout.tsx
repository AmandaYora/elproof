import { Suspense } from "react";
import { Outlet } from "react-router-dom";
import { Sidebar } from "@/shared/layouts/Sidebar";
import { Topbar } from "@/shared/layouts/Topbar";

export function AppLayout() {
  return (
    <div className="min-h-screen bg-background">
      <Sidebar />
      <div className="pl-64">
        <Topbar />
        <main className="mx-auto max-w-[1400px] px-6 py-6">
          <Suspense fallback={<div className="py-20 text-center text-sm text-text-secondary">Memuat halaman...</div>}>
            <Outlet />
          </Suspense>
        </main>
      </div>
    </div>
  );
}
