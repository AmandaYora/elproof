import { useState } from "react";
import { Link } from "react-router-dom";
import { AlertTriangle, AlertCircle, Clock, CheckCircle2 } from "lucide-react";
import { Card, CardHeader } from "@/shared/components/ui/Card";
import type { AttentionGroup, AttentionItem, AttentionTone } from "@/modules/dashboard/lib/attention";
import { cn } from "@/shared/lib/cn";

const TONE_ICON: Record<AttentionTone, typeof AlertTriangle> = {
  danger: AlertTriangle,
  warning: AlertCircle,
  info: Clock,
};

const TONE_ICON_CLASS: Record<AttentionTone, string> = {
  danger: "text-danger-text",
  warning: "text-warning-text",
  info: "text-info-text",
};

const TONE_LABEL: Record<AttentionTone, string> = {
  danger: "Mendesak",
  warning: "Perlu dicek",
  info: "Informasi",
};

type Filter = "semua" | Exclude<AttentionGroup, "lainnya">;

const FILTERS: { key: Filter; label: string; empty: string }[] = [
  { key: "semua", label: "Semua", empty: "Tidak ada yang perlu ditindaklanjuti. Seluruh proyek berjalan sesuai rencana." },
  { key: "tagihan", label: "Tagihan", empty: "Tidak ada tagihan klien yang terlambat." },
  { key: "kendala", label: "Kendala", empty: "Tidak ada kendala yang masih terbuka." },
  { key: "bukti", label: "Bukti", empty: "Semua pembayaran sudah memiliki bukti lengkap." },
];

/** How many rows show before "Tampilkan semua". */
const FIRST_ROWS = 6;

/**
 * "Perlu ditindaklanjuti": everything that needs someone, most urgent first, one line each with a "Buka" link to the
 * place where it is handled. Chips narrow it to unpaid client installments, open kendala, or payments without proof.
 */
export function AttentionQueue({ items, counts, hasProjects }: { items: AttentionItem[]; counts: Record<AttentionGroup, number>; hasProjects: boolean }) {
  const [filter, setFilter] = useState<Filter>("semua");
  const [expanded, setExpanded] = useState(false);
  const shown = filter === "semua" ? items : items.filter((i) => i.group === filter);
  const visible = expanded ? shown : shown.slice(0, FIRST_ROWS);
  const current = FILTERS.find((f) => f.key === filter)!;

  return (
    <Card>
      <CardHeader title="Perlu ditindaklanjuti">
        <div role="group" aria-label="Saring daftar" className="mt-3 flex flex-wrap gap-2">
          {FILTERS.map((f) => {
            const n = f.key === "semua" ? items.length : counts[f.key];
            const active = filter === f.key;
            return (
              <button
                key={f.key}
                type="button"
                aria-pressed={active}
                onClick={() => {
                  setFilter(f.key);
                  setExpanded(false);
                }}
                className={cn(
                  "inline-flex h-8 items-center gap-1.5 rounded-full border px-3 text-[13px] font-medium transition-colors pointer-coarse:h-10",
                  "outline-none focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-solid focus-visible:outline-ring",
                  active ? "border-primary bg-primary text-primary-foreground" : "border-border bg-card text-muted-foreground hover:bg-muted hover:text-foreground",
                )}
              >
                {f.label}
                <span className={cn("font-semibold tabular-nums", active ? "text-primary-foreground" : "text-foreground")}>{n}</span>
              </button>
            );
          })}
        </div>
      </CardHeader>

      {shown.length === 0 ? (
        <div className="flex items-center gap-3 px-5 py-6">
          <CheckCircle2 aria-hidden className="size-5 shrink-0 text-success-text" />
          <p className="text-[14px] text-muted-foreground">
            {hasProjects ? current.empty : "Belum ada proyek. Tagihan, kendala, dan bukti yang perlu dicek akan muncul di sini setelah proyek berjalan."}
          </p>
        </div>
      ) : (
        <>
          <ul className="divide-y divide-border-light">
            {visible.map((item) => {
              const Icon = TONE_ICON[item.tone];
              return (
                <li key={item.id} className="relative flex items-center gap-3 px-5 py-2.5 hover:bg-muted">
                  <Icon aria-hidden className={cn("size-4 shrink-0", TONE_ICON_CLASS[item.tone])} />
                  <span className="min-w-0 flex-1 md:flex md:items-baseline md:gap-2">
                    <span className="block truncate text-[14px] font-medium text-foreground md:max-w-[60%] md:shrink-0">
                      <span className="sr-only">{`${TONE_LABEL[item.tone]}: `}</span>
                      {item.title}
                    </span>
                    <span className="block truncate text-[13px] text-muted-foreground md:min-w-0">{item.description}</span>
                  </span>
                  <Link
                    to={item.to}
                    aria-label={`Buka: ${item.title}`}
                    className={cn(
                      "inline-flex h-8 shrink-0 items-center rounded-md px-2.5 text-[13px] font-semibold text-primary hover:bg-card pointer-coarse:h-10",
                      "outline-none focus-visible:outline-2 focus-visible:outline-solid focus-visible:outline-ring",
                      // The whole row opens the item; the link stays the one control in it.
                      "after:absolute after:inset-0 after:content-['']",
                    )}
                  >
                    Buka
                  </Link>
                </li>
              );
            })}
          </ul>
          {shown.length > FIRST_ROWS && (
            <div className="border-t border-border-light px-5 py-2.5">
              <button
                type="button"
                onClick={() => setExpanded(!expanded)}
                aria-expanded={expanded}
                className="rounded-md text-[13px] font-semibold text-primary underline-offset-2 hover:underline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-solid focus-visible:outline-ring"
              >
                {expanded ? "Tampilkan lebih sedikit" : `Tampilkan ${shown.length - FIRST_ROWS} lainnya`}
              </button>
            </div>
          )}
        </>
      )}
    </Card>
  );
}
