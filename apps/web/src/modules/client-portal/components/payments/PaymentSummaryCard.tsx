import { useState } from "react";
import { Circle, CircleCheck, CircleDot, Send, TriangleAlert, type LucideIcon } from "lucide-react";
import { Button } from "@/shared/components/kit/button";
import { Badge } from "@/shared/components/ui/Badge";
import { cn } from "@/shared/lib/cn";
import { TERM_STATUS_LABELS } from "@/shared/lib/labels";
import { formatCurrency, formatCurrencyCompact, formatDate } from "@/shared/lib/formatters";
import { BankAccount } from "@/modules/client-portal/components/home/NextPaymentCard";
import { WhatsAppButton } from "@/modules/client-portal/components/WhatsAppButton";
import { askMessage, type PortalContact } from "@/modules/client-portal/lib/usePortalContact";
import type { PaymentTerm } from "@/modules/projects/types";
import type { PortalHome } from "@/modules/client-portal/types";

type Part = "paid" | "partial" | "unpaid" | "late";

function partOf(t: PaymentTerm): Part {
  if (t.status === "paid") return "paid";
  if (t.status === "overdue") return "late";
  return t.status === "partial" || t.paid > 0 ? "partial" : "unpaid";
}

// Status is never colour alone: each part of the bar has an icon and words in the legend, and the bar has a text name.
const LEGEND: Record<Part, { Icon: LucideIcon; icon: string; words: string }> = {
  paid: { Icon: CircleCheck, icon: "text-primary", words: "lunas" },
  partial: { Icon: CircleDot, icon: "text-primary", words: "dibayar sebagian" },
  late: { Icon: TriangleAlert, icon: "text-warning-text", words: "lewat jatuh tempo" },
  unpaid: { Icon: Circle, icon: "text-muted-foreground", words: "belum dibayar" },
};

/** One segment per term, as wide as its amount: navy for what is paid, amber behind a term that is late. */
function TermBar({ terms, paid, total }: { terms: PaymentTerm[]; paid: number; total: number }) {
  if (terms.length === 0) {
    const pct = total > 0 ? Math.min(100, Math.round((paid / total) * 100)) : 0;
    return (
      <div role="img" aria-label={`${pct}% dari nilai kontrak sudah dibayar`} className="mt-4 h-2.5 overflow-hidden rounded-full bg-muted">
        <div className="h-full rounded-full bg-primary" style={{ width: `${pct}%` }} />
      </div>
    );
  }
  const label = terms.map((t) => `${t.label} ${(TERM_STATUS_LABELS.client[t.status] ?? "").toLowerCase()}`).join(", ");
  return (
    <div role="img" aria-label={label} className="mt-4 flex gap-1">
      {terms.map((t) => {
        const part = partOf(t);
        const pct = t.amount > 0 ? Math.min(100, Math.round((t.paid / t.amount) * 100)) : part === "paid" ? 100 : 0;
        return (
          <div key={t.id} className={cn("h-2.5 overflow-hidden rounded-full", part === "late" ? "bg-warning" : "bg-muted")} style={{ flexGrow: Math.max(t.amount, 1), flexBasis: 0 }}>
            <div className="h-full bg-primary" style={{ width: `${part === "paid" ? 100 : pct}%` }} />
          </div>
        );
      })}
    </div>
  );
}

interface Props {
  finance: NonNullable<PortalHome["finance"]>;
  terms: PaymentTerm[];
  payTo: PortalHome["payTo"];
  contact: PortalContact;
  /** Shown when the client's role may send a transfer proof (portal.finance.submit). */
  onSubmitProof?: () => void;
}

/**
 * The whole money picture in one card (RISET #2): how much is paid of the contract, one bar segment per term, what is
 * next, and the three things a couple does from here (send proof, see the account, ask). Plain amounts stay in the
 * text colour; only a late term gets a status colour.
 */
export function PaymentSummaryCard({ finance, terms, payTo, contact, onSubmitProof }: Props) {
  const [showBank, setShowBank] = useState(false);
  const hasBank = Boolean(payTo?.bankName && payTo.accountNumber);
  const next = finance.nextTerm;
  const counts = terms.reduce<Record<Part, number>>((acc, t) => ({ ...acc, [partOf(t)]: acc[partOf(t)] + 1 }), { paid: 0, partial: 0, late: 0, unpaid: 0 });
  const settled = finance.contractValue > 0 && finance.remaining <= 0;

  return (
    <section aria-labelledby="summary-heading" className="rounded-2xl border border-border bg-card p-5 sm:p-6">
      <h2 id="summary-heading" className="text-[13px] font-medium text-muted-foreground">Sudah dibayar</h2>
      <p className="mt-1 flex flex-wrap items-baseline gap-x-2">
        <span aria-hidden className="text-[30px] leading-tight font-extrabold tracking-[-0.02em] text-foreground tabular-nums">{formatCurrencyCompact(finance.paid)}</span>
        <span aria-hidden className="text-[16px] font-medium text-muted-foreground tabular-nums">dari {formatCurrencyCompact(finance.contractValue)}</span>
        <span className="sr-only">
          {formatCurrency(finance.paid)} dari {formatCurrency(finance.contractValue)}
        </span>
      </p>

      <TermBar terms={terms} paid={finance.paid} total={finance.contractValue} />

      {terms.length > 0 && (
        <ul className="mt-3 flex flex-wrap gap-x-4 gap-y-1.5 text-[13px] text-muted-foreground">
          {(Object.keys(LEGEND) as Part[])
            .filter((p) => counts[p] > 0)
            .map((p) => {
              const { Icon, icon, words } = LEGEND[p];
              return (
                <li key={p} className="inline-flex items-center gap-1.5">
                  <Icon className={cn("size-3.5", icon)} aria-hidden />
                  {counts[p]} {words}
                </li>
              );
            })}
        </ul>
      )}

      <div className="mt-5 border-t border-border-light pt-4 text-[14px]">
        {settled ? (
          <p className="font-medium text-foreground">Seluruh tagihan sudah lunas. Terima kasih.</p>
        ) : next ? (
          <div className="flex flex-wrap items-center gap-x-3 gap-y-1.5">
            <p className="text-foreground">
              Berikutnya: <span className="font-semibold">{next.label}</span>, {formatDate(next.dueDate)}
              <span className="text-muted-foreground tabular-nums"> · {formatCurrency(next.remaining)}</span>
            </p>
            {next.status === "overdue" && <Badge tone="warning">{TERM_STATUS_LABELS.client.overdue}</Badge>}
          </div>
        ) : (
          <p className="text-foreground">
            Sisa tagihan <span className="font-semibold tabular-nums">{formatCurrency(finance.remaining)}</span>
          </p>
        )}
      </div>

      <div className="mt-4 flex flex-col gap-2.5 sm:flex-row sm:flex-wrap">
        {onSubmitProof && (
          <Button onClick={onSubmitProof}>
            <Send aria-hidden /> Kirim bukti transfer
          </Button>
        )}
        {hasBank && (
          <Button variant="outline" aria-expanded={showBank} aria-controls={showBank ? "pay-to" : undefined} onClick={() => setShowBank((v) => !v)}>
            {showBank ? "Sembunyikan rekening" : "Lihat rekening"}
          </Button>
        )}
        {hasBank && (
          <WhatsAppButton
            contact={contact}
            label="Tanyakan lewat WhatsApp"
            quietWhenMissing={false}
            message={askMessage(contact, next ? `pembayaran ${next.label} ${formatCurrency(next.remaining)}` : "pembayaran kami")}
          />
        )}
      </div>

      {(showBank || !hasBank) && !settled && (
        <div id="pay-to" className="mt-4">
          <BankAccount payTo={payTo} contact={contact} forWhat={next?.label ?? "tagihan"} />
        </div>
      )}
    </section>
  );
}
