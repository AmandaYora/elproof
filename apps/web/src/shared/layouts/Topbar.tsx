import { GlobalSearch } from "@/shared/layouts/GlobalSearch";
import { formatDate, todayISO } from "@/shared/lib/formatters";

export function Topbar() {
  return (
    <header className="sticky top-0 z-20 flex h-16 items-center justify-between gap-4 border-b border-border bg-surface px-6">
      <GlobalSearch />
      <div className="shrink-0 text-[13px] font-medium text-text-secondary">{formatDate(todayISO())}</div>
    </header>
  );
}
