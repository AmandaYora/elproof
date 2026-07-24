import { Suspense, useState } from "react";
import { Outlet } from "react-router-dom";
import { Sidebar } from "@/shared/layouts/Sidebar";
import { Topbar } from "@/shared/layouts/Topbar";

export function AppLayout() {
  const [sidebarOpen, setSidebarOpen] = useState(false);

  return (
    <div className="min-h-screen bg-background">
      <Sidebar open={sidebarOpen} onClose={() => setSidebarOpen(false)} />
      <div className="lg:pl-64">
        <Topbar onMenuClick={() => setSidebarOpen(true)} />
        <main className="mx-auto max-w-[1400px] px-4 py-5 sm:px-6 sm:py-6">
          <Suspense fallback={<div className="py-20 text-center text-sm text-text-secondary">Memuat halaman...</div>}>
            <Outlet />
          </Suspense>
        </main>
      </div>
    </div>
  );
}
