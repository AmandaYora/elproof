import { forwardRef, type InputHTMLAttributes, type LabelHTMLAttributes, type ReactNode } from "react";
import { cn } from "@/shared/lib/cn";

export { Combobox as Select } from "@/shared/components/ui/Combobox";

export const Input = forwardRef<HTMLInputElement, InputHTMLAttributes<HTMLInputElement>>(function Input(
  { className, ...props },
  ref
) {
  return (
    <input
      ref={ref}
      className={cn(
        "h-9 w-full rounded-md border border-border bg-white px-3 text-sm text-text-primary placeholder:text-text-secondary/70",
        "focus:outline-none focus:ring-2 focus:ring-navy-900/20 focus:border-navy-900",
        "disabled:bg-surface-muted disabled:text-text-secondary",
        className
      )}
      {...props}
    />
  );
});

export const Textarea = forwardRef<HTMLTextAreaElement, React.TextareaHTMLAttributes<HTMLTextAreaElement>>(
  function Textarea({ className, ...props }, ref) {
    return (
      <textarea
        ref={ref}
        className={cn(
          "w-full rounded-md border border-border bg-white px-3 py-2 text-sm text-text-primary placeholder:text-text-secondary/70",
          "focus:outline-none focus:ring-2 focus:ring-navy-900/20 focus:border-navy-900",
          className
        )}
        {...props}
      />
    );
  }
);

export function Field({
  label,
  htmlFor,
  hint,
  required,
  children,
}: {
  label: string;
  htmlFor?: string;
  hint?: string;
  required?: boolean;
} & { children: ReactNode }) {
  return (
    <label htmlFor={htmlFor} className="flex flex-col gap-1.5">
      <span className="text-[13px] font-medium text-text-primary">
        {label}
        {required && <span className="text-danger"> *</span>}
      </span>
      {children}
      {hint && <span className="text-xs text-text-secondary">{hint}</span>}
    </label>
  );
}

export function FieldLabel({ className, ...props }: LabelHTMLAttributes<HTMLLabelElement>) {
  return <label className={cn("text-[13px] font-medium text-text-primary", className)} {...props} />;
}
