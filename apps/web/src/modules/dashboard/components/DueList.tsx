import { Link } from "react-router-dom";
import { Card, CardHeader } from "@/shared/components/ui/Card";
import type { ReactNode } from "react";

export interface DueListRow {
  id: string;
  to: string;
  title: string;
  subtitle: string;
  trailing: ReactNode;
}

/** A short list of dated things (tasks, payments) with an honest empty line. */
export function DueList({ title, subtitle, rows, empty }: { title: string; subtitle?: string; rows: DueListRow[]; empty: string }) {
  return (
    <Card>
      <CardHeader title={title} subtitle={subtitle} />
      {rows.length === 0 ? (
        <p className="px-5 py-5 text-[13px] text-muted-foreground">{empty}</p>
      ) : (
        <ul className="divide-y divide-border-light">
          {rows.map((r) => (
            <li key={r.id}>
              <Link
                to={r.to}
                className="flex items-center justify-between gap-3 px-5 py-2.5 hover:bg-muted focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-solid focus-visible:outline-ring"
              >
                <span className="min-w-0">
                  <span className="block truncate text-[14px] font-medium text-foreground">{r.title}</span>
                  <span className="block truncate text-[12px] text-muted-foreground">{r.subtitle}</span>
                </span>
                <span className="shrink-0 text-[13px] font-semibold tabular-nums text-foreground">{r.trailing}</span>
              </Link>
            </li>
          ))}
        </ul>
      )}
    </Card>
  );
}
