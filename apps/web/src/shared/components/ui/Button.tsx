import type { ButtonHTMLAttributes, ReactNode } from "react";
import { cn } from "@/shared/lib/cn";

type ButtonVariant = "primary" | "secondary" | "ghost" | "danger" | "neutral";
type ButtonSize = "sm" | "md";

interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: ButtonVariant;
  size?: ButtonSize;
  icon?: ReactNode;
}

const VARIANT_CLASSES: Record<ButtonVariant, string> = {
  primary: "bg-navy-900 text-white hover:bg-navy-800 disabled:bg-navy-900/50",
  secondary: "bg-white text-text-primary border border-border hover:bg-surface-muted",
  ghost: "bg-transparent text-text-secondary hover:bg-surface-muted",
  danger: "bg-danger text-white hover:bg-danger/90",
  // Deliberately NOT tied to --brand-navy-* — for the rare screen (LoginPage)
  // that must stay tenant-agnostic since no tenant is known pre-auth.
  neutral: "bg-slate-800 text-white hover:bg-slate-700 disabled:bg-slate-800/50",
};

const SIZE_CLASSES: Record<ButtonSize, string> = {
  sm: "h-8 px-3 text-[13px] gap-1.5",
  md: "h-9 px-4 text-sm gap-2",
};

export function Button({ variant = "primary", size = "md", icon, className, children, ...props }: ButtonProps) {
  return (
    <button
      className={cn(
        "inline-flex items-center justify-center rounded-md font-semibold transition-colors disabled:cursor-not-allowed disabled:opacity-60 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-navy-900",
        VARIANT_CLASSES[variant],
        SIZE_CLASSES[size],
        className
      )}
      {...props}
    >
      {icon}
      {children}
    </button>
  );
}
