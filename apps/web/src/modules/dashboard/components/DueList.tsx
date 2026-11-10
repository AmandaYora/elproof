import { Link } from "react-router-dom";
import { Card, CardHeader, CardContent } from "@/shared/components/ui/Card";
import type { ReactNode } from "react";

export interface DueListRow {
  id: string;
  to: string;
  title: string;
  subtitle: string;
  trailing: ReactNode;
}

/** A short list of dated things (events, tasks, payments) with an honest empty line. */
export function DueList({ title, subtitle, rows, empty }: { title: string; subtitle: string; rows: DueListRow[]; empty: string }) {
  return (
    <Card>
      <CardHeader title={title} subtitle={subtitle} />
      <CardContent className="p-0">
        {rows.length === 0 ? (
          <p className="px-5 py-5 text-[13px] text-text-secondary">{empty}</p>
        ) : (
          <ul className="divide-y divide-border-light">
            {rows.map((r) => (
              <li key={r.id}>
                <Link to={r.to} className="flex items-center justify-between gap-3 px-5 py-3 hover:bg-surface-muted focus-visible:outline focus-visible:outline-2 focus-visible:outline-ring">
                  <span className="min-w-0">
                    <span className="block truncate text-[13.5px] font-semibold text-text-primary">{r.title}</span>
                    <span className="block truncate text-[12.5px] text-text-secondary">{r.subtitle}</span>
                  </span>
                  <span className="shrink-0 text-[12.5px] font-semibold tabular-nums text-primary">{r.trailing}</span>
                </Link>
              </li>
            ))}
          </ul>
        )}
      </CardContent>
    </Card>
  );
}
