import { Link } from "react-router-dom";
import { ROUTE_PATHS } from "@/app/routes/route-paths";

export function NotFoundPage() {
  return (
    <div className="flex min-h-screen flex-col items-center justify-center gap-3 bg-background text-center">
      <p className="text-sm font-semibold text-text-secondary">404</p>
      <h1 className="text-xl font-bold text-text-primary">Halaman tidak ditemukan</h1>
      <Link to={ROUTE_PATHS.dashboard} className="text-sm font-medium text-navy-900 underline underline-offset-2">
        Kembali ke Dashboard
      </Link>
    </div>
  );
}
