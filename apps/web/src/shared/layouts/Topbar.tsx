import { Menu } from "lucide-react";
import { GlobalSearch } from "@/shared/layouts/GlobalSearch";
import { formatDate, todayISO } from "@/shared/lib/formatters";

interface TopbarProps {
  onMenuClick: () => void;
}

export function Topbar({ onMenuClick }: TopbarProps) {
  return (
    <header className="sticky top-0 z-20 flex h-16 items-center justify-between gap-3 border-b border-border bg-surface px-4 sm:px-6">
      <button
        onClick={onMenuClick}
        aria-label="Buka menu"
        className="shrink-0 rounded-md p-1.5 text-text-secondary hover:bg-surface-muted lg:hidden"
      >
        <Menu className="h-5 w-5" />
      </button>
      <GlobalSearch />
      <div className="hidden shrink-0 text-[13px] font-medium text-text-secondary sm:block">{formatDate(todayISO())}</div>
    </header>
  );
}
