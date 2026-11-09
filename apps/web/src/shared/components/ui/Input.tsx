import { cloneElement, isValidElement, useId, type LabelHTMLAttributes, type ReactElement, type ReactNode } from "react";
import { CircleAlert } from "lucide-react";
import { cn } from "@/shared/lib/cn";
import { Input } from "@/shared/components/kit/input";
import { Textarea } from "@/shared/components/kit/textarea";

export { Combobox as Select } from "@/shared/components/ui/Combobox";

export { Input } from "@/shared/components/kit/input";
export { Textarea } from "@/shared/components/kit/textarea";

/** Controls that take id and aria-* straight onto the element Field wraps. */
function acceptsAria(child: ReactElement): boolean {
  const type = child.type;
  return type === "input" || type === "textarea" || type === "select" || type === Input || type === Textarea;
}

/**
 * Label + control + help or error. The label wraps the control (so clicking it
 * focuses the field even where no htmlFor is given); the message is tied to the
 * control with aria-describedby and a failed field gets aria-invalid, so a screen
 * reader hears what is wrong. The error also carries an icon: never color alone.
 */
export function Field({
  label,
  htmlFor,
  name,
  hint,
  error,
  required,
  children,
}: {
  label: string;
  htmlFor?: string;
  /** The field's key in the form: lets an error summary jump to it (data-field). */
  name?: string;
  hint?: string;
  /** Validation message: shown in the danger color with an icon, replacing the hint. */
  error?: string;
  required?: boolean;
} & { children: ReactNode }) {
  const messageId = `${useId()}-message`;
  const message = error ?? hint;
  const control =
    isValidElement(children) && acceptsAria(children as ReactElement)
      ? cloneElement(children as ReactElement<Record<string, unknown>>, {
          "aria-describedby": message ? messageId : undefined,
          "aria-invalid": error ? true : (children.props as Record<string, unknown>)["aria-invalid"],
        })
      : children;
  return (
    <label htmlFor={htmlFor} data-field={name} className="flex flex-col gap-1.5">
      <span className="text-[13px] font-medium text-foreground">
        {label}
        {required && (
          <>
            <span aria-hidden className="text-danger-text"> *</span>
            <span className="sr-only"> (wajib diisi)</span>
          </>
        )}
      </span>
      {control}
      {error ? (
        <span id={messageId} role="alert" className="flex items-start gap-1 text-xs font-medium text-danger-text">
          <CircleAlert aria-hidden className="mt-px size-3.5 shrink-0" />
          {error}
        </span>
      ) : (
        hint && (
          <span id={messageId} className="text-xs text-muted-foreground">
            {hint}
          </span>
        )
      )}
    </label>
  );
}

export function FieldLabel({ className, ...props }: LabelHTMLAttributes<HTMLLabelElement>) {
  return <label className={cn("text-[13px] font-medium text-foreground", className)} {...props} />;
}
