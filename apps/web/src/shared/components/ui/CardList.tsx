import type { ReactNode } from "react";
import { cn } from "@/shared/lib/cn";

interface CardListProps<T> {
  items: T[];
  keyFor: (item: T) => string;
  renderItem: (item: T) => ReactNode;
  className?: string;
}

export function CardList<T>({ items, keyFor, renderItem, className }: CardListProps<T>) {
  return (
    <ul className={cn("divide-y divide-border-light", className)}>
      {items.map((item) => (
        <li key={keyFor(item)} className="flex flex-col gap-2.5 px-4 py-3.5">
          {renderItem(item)}
        </li>
      ))}
    </ul>
  );
}

export function CardListField({ label, value }: { label: string; value: ReactNode }) {
  return (
    <div className="flex items-center justify-between gap-3 text-[12.5px]">
      <span className="shrink-0 text-text-secondary">{label}</span>
      <span className="min-w-0 truncate text-right font-medium text-text-primary">{value}</span>
    </div>
  );
}
