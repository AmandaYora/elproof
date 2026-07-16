import { Search, X } from "lucide-react";
import type { ChangeEvent, InputHTMLAttributes } from "react";
import { cn } from "@/shared/lib/cn";

export function SearchInput({ className, value, onChange, ...props }: InputHTMLAttributes<HTMLInputElement>) {
  const hasValue = typeof value === "string" && value.length > 0;

  function clear() {
    onChange?.({ target: { value: "" } } as ChangeEvent<HTMLInputElement>);
  }

  return (
    <div className={cn("group relative", className)}>
      <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-text-secondary transition-colors group-focus-within:text-navy-900" />
      <input
        type="text"
        value={value}
        onChange={onChange}
        className={cn(
          "h-9 w-full rounded-lg border border-border bg-white pl-9 text-sm text-text-primary placeholder:text-text-secondary/70 shadow-sm transition-shadow",
          "focus:outline-none focus:ring-2 focus:ring-navy-900/20 focus:border-navy-900",
          hasValue ? "pr-8" : "pr-3"
        )}
        {...props}
      />
      {hasValue && (
        <button
          type="button"
          onClick={clear}
          aria-label="Bersihkan pencarian"
          className="absolute right-2 top-1/2 flex h-5 w-5 -translate-y-1/2 items-center justify-center rounded-full text-text-secondary transition-colors hover:bg-surface-muted hover:text-text-primary"
        >
          <X className="h-3.5 w-3.5" />
        </button>
      )}
    </div>
  );
}
