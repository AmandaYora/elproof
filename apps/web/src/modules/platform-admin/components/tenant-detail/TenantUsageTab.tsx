import { Card, CardHeader, CardContent } from "@/shared/components/ui/Card";
import type { UsageItem } from "@/modules/platform-admin/data/console.types";

function UsageBar({ label, item }: { label: string; item: UsageItem }) {
  const limited = item.max !== null;
  const percent = limited && item.max! > 0 ? Math.min(100, (item.used / item.max!) * 100) : 0;
  const full = limited && item.used >= item.max!;
  return (
    <div>
      <div className="flex items-baseline justify-between gap-3">
        <span className="text-[13.5px] font-medium text-text-primary">{label}</span>
        <span className={`text-[13px] tabular-nums ${full ? "font-semibold text-danger-text" : "text-text-secondary"}`}>
          {item.used}
          {limited ? ` dari ${item.max}` : ", tanpa batas"}
        </span>
      </div>
      {limited && (
        <div className="mt-2 h-2 w-full overflow-hidden rounded-full bg-surface-muted" role="img" aria-label={`${label}: ${item.used} dari ${item.max}`}>
          <div className={`h-full rounded-full ${full ? "bg-danger" : "bg-navy-900"}`} style={{ width: `${percent}%` }} />
        </div>
      )}
    </div>
  );
}

interface Props {
  isTrial: boolean;
  staff: UsageItem;
  projects: UsageItem;
  clients: UsageItem;
}

export function TenantUsageTab({ isTrial, staff, projects, clients }: Props) {
  return (
    <Card>
      <CardHeader
        title="Pemakaian"
        subtitle={isTrial ? "Selama masa coba, tiap sumber daya dibatasi. Batas diatur di halaman Paket." : "Paket berbayar tidak dibatasi, angka ini hanya gambaran pemakaian."}
      />
      <CardContent className="flex flex-col gap-5">
        <UsageBar label="Staf aktif" item={staff} />
        <UsageBar label="Project aktif" item={projects} />
        <UsageBar label="Klien" item={clients} />
      </CardContent>
    </Card>
  );
}
