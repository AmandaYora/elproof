import { cn } from "@/shared/lib/cn";
import type { DashboardPeriod } from "@/modules/platform-admin/lib/trend";

const OPTIONS: { value: DashboardPeriod; label: string }[] = [
  { value: "month", label: "Bulan Ini" },
  { value: "year", label: "Tahun Ini" },
];

interface PeriodToggleProps {
  value: DashboardPeriod;
  onChange: (value: DashboardPeriod) => void;
}

export function PeriodToggle({ value, onChange }: PeriodToggleProps) {
  return (
    <div className="inline-flex rounded-md border border-border bg-surface p-1">
      {OPTIONS.map((option) => (
        <button
          key={option.value}
          type="button"
          onClick={() => onChange(option.value)}
          className={cn(
            "rounded px-3 py-1.5 text-[13px] font-medium transition-colors",
            value === option.value ? "bg-navy-900 text-white" : "text-text-secondary hover:text-text-primary"
          )}
        >
          {option.label}
        </button>
      ))}
    </div>
  );
}
