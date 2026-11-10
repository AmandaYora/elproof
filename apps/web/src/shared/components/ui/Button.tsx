import type { ButtonHTMLAttributes, ReactNode, Ref } from "react";
import { Button as KitButton } from "@/shared/components/kit/button";

type ButtonVariant = "primary" | "secondary" | "ghost" | "danger";
type ButtonSize = "sm" | "md";

interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: ButtonVariant;
  size?: ButtonSize;
  icon?: ReactNode;
  /** React 19 passes ref as a regular prop (e.g. to focus "Batal"). */
  ref?: Ref<HTMLButtonElement>;
}

const KIT_VARIANT = {
  primary: "default",
  secondary: "secondary",
  ghost: "ghost",
  danger: "destructive",
} as const;

/**
 * The app's Button, kept with its original props (variant, size, icon) so
 * the ~80 call sites did not change when it moved onto shadcn/ui + Base UI
 * (PLAN.md Tahap 1). New code may use `kit/button` directly.
 */
export function Button({ variant = "primary", size = "md", icon, children, ...props }: ButtonProps) {
  return (
    <KitButton variant={KIT_VARIANT[variant]} size={size === "sm" ? "sm" : "default"} {...props}>
      {icon}
      {children}
    </KitButton>
  );
}
