import { Link } from "react-router-dom";
import { AlertTriangle, AlertCircle, Clock, CheckCircle2 } from "lucide-react";
import { Card, CardHeader, CardContent } from "@/shared/components/ui/Card";
import { EmptyState } from "@/shared/components/feedback/EmptyState";
import type { AttentionItem, AttentionTone } from "@/modules/dashboard/lib/attention";
import { cn } from "@/shared/lib/cn";

const TONE_ICON: Record<AttentionTone, typeof AlertTriangle> = {
  danger: AlertTriangle,
  warning: AlertCircle,
  info: Clock,
};

const TONE_ICON_CLASS: Record<AttentionTone, string> = {
  danger: "text-danger",
  warning: "text-warning",
  info: "text-info",
};

export function AttentionQueue({ items, counts }: { items: AttentionItem[]; counts: Record<string, number> }) {
  return (
    <Card>
      <CardHeader
        title="Perlu Ditindaklanjuti"
        subtitle="Diurutkan dari yang paling mendesak — klik untuk membuka detail terkait."
      />
      {Object.keys(counts).length > 0 && (
        <div className="flex flex-wrap items-center gap-2 border-b border-border-light px-5 py-3">
          {Object.entries(counts).map(([label, count]) => (
            <span key={label} className="rounded-full bg-surface-muted px-2.5 py-1 text-[11px] font-semibold text-text-secondary">
              {label} · {count}
            </span>
          ))}
        </div>
      )}
      <CardContent className="p-0">
        {items.length === 0 ? (
          <EmptyState
            icon={<CheckCircle2 className="h-8 w-8 text-success" />}
            title="Tidak ada yang perlu ditindaklanjuti"
            description="Seluruh project berjalan sesuai rencana tanpa kendala terbuka."
          />
        ) : (
          <ul className="divide-y divide-border-light">
            {items.map((item) => {
              const Icon = TONE_ICON[item.tone];
              return (
                <li key={item.id}>
                  <Link to={item.to} className="flex items-start gap-3 px-5 py-3.5 transition-colors hover:bg-surface-muted">
                    <Icon className={cn("mt-0.5 h-4 w-4 shrink-0", TONE_ICON_CLASS[item.tone])} />
                    <span className="min-w-0 flex-1">
                      <span className="block text-[13.5px] font-semibold text-text-primary">{item.title}</span>
                      <span className="block truncate text-[12.5px] text-text-secondary">{item.description}</span>
                    </span>
                  </Link>
                </li>
              );
            })}
          </ul>
        )}
      </CardContent>
    </Card>
  );
}
