import type { ReactNode } from "react";
import { Link } from "react-router-dom";
import { ArrowRight } from "lucide-react";
import { cn } from "@/shared/lib/cn";

type StatTone = "neutral" | "success" | "warning" | "danger" | "info";

const ACCENT_CLASSES: Record<StatTone, string> = {
  neutral: "text-text-primary",
  success: "text-success-text",
  warning: "text-warning-text",
  danger: "text-danger-text",
  info: "text-info-text",
};

interface StatCardProps {
  label: string;
  value: ReactNode;
  sublabel?: string;
  tone?: StatTone;
  to?: string;
  icon?: ReactNode;
}

export function StatCard({ label, value, sublabel, tone = "neutral", to, icon }: StatCardProps) {
  const content = (
    <>
      <div className="flex items-center justify-between">
        <span className="text-[13px] font-medium text-text-secondary">{label}</span>
        {icon}
      </div>
      <div className={cn("mt-2 font-bold tabular-nums", ACCENT_CLASSES[tone], "text-[28px] leading-none")}>{value}</div>
      {sublabel && (
        <div className="mt-2 flex items-center gap-1 text-[12px] text-text-secondary">
          <span>{sublabel}</span>
          {to && <ArrowRight className="h-3 w-3" />}
        </div>
      )}
    </>
  );

  const className = "block rounded-lg border border-border bg-surface px-5 py-4 transition-colors hover:border-primary/30";

  if (to) {
    return (
      <Link to={to} className={className}>
        {content}
      </Link>
    );
  }
  return <div className={className}>{content}</div>;
}
