import type { HTMLAttributes, ReactNode } from "react";
import { cn } from "@/shared/lib/cn";

export function Card({ className, ...props }: HTMLAttributes<HTMLDivElement>) {
  return <div className={cn("rounded-lg border border-border bg-surface", className)} {...props} />;
}

interface CardHeaderProps extends Omit<HTMLAttributes<HTMLDivElement>, "title"> {
  title?: ReactNode;
  subtitle?: ReactNode;
  action?: ReactNode;
}

export function CardHeader({ title, subtitle, action, className, children, ...props }: CardHeaderProps) {
  return (
    <div className={cn("flex flex-wrap items-start justify-between gap-x-3 gap-y-2 border-b border-border-light px-5 py-4", className)} {...props}>
      <div className="min-w-0">
        {title && <h3 className="text-[15px] font-semibold text-text-primary">{title}</h3>}
        {subtitle && <p className="mt-0.5 text-[13px] text-text-secondary">{subtitle}</p>}
        {children}
      </div>
      {action && <div className="shrink-0">{action}</div>}
    </div>
  );
}

export function CardContent({ className, ...props }: HTMLAttributes<HTMLDivElement>) {
  return <div className={cn("px-5 py-4", className)} {...props} />;
}
