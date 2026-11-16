import { Copy } from "lucide-react";
import { toast } from "@/shared/stores/useToastStore";
import { Badge } from "@/shared/components/ui/Badge";
import { TERM_STATUS_LABELS } from "@/shared/lib/labels";
import { formatCurrency, formatDate } from "@/shared/lib/formatters";
import { Button } from "@/shared/components/kit/button";
import { daysUntil } from "@/modules/projects/lib/dates";
import { paymentConfirmationMessage } from "@/modules/client-portal/lib/payment-message";
import { WhatsAppButton } from "@/modules/client-portal/components/WhatsAppButton";
import { askMessage, type PortalContact } from "@/modules/client-portal/lib/usePortalContact";
import type { PortalHome } from "@/modules/client-portal/types";

async function copy(text: string) {
  try {
    await navigator.clipboard.writeText(text);
    toast.success("Nomor rekening disalin");
  } catch {
    toast.error("Belum bisa menyalin. Tekan lama pada nomor untuk menyalinnya.");
  }
}

/** Where to transfer, with a copy button; or, when the WO has not shared it, a button that asks for it. */
export function BankAccount({ payTo, contact, forWhat }: { payTo: PortalHome["payTo"]; contact: PortalContact; forWhat: string }) {
  if (payTo?.bankName && payTo.accountNumber) {
    return (
      <div className="rounded-xl bg-muted p-4">
        <p className="text-[13px] font-medium text-muted-foreground">Transfer ke</p>
        <p className="mt-0.5 text-[14px] font-semibold text-foreground">{payTo.bankName}</p>
        <div className="mt-1 flex items-center justify-between gap-3">
          <p className="font-mono text-[18px] font-semibold tracking-wide text-foreground tabular-nums select-all">{payTo.accountNumber}</p>
          <Button variant="outline" onClick={() => void copy(payTo.accountNumber)} className="shrink-0 bg-card">
            <Copy aria-hidden /> Salin
          </Button>
        </div>
        {payTo.accountHolder && <p className="mt-1 text-[13px] text-muted-foreground">a.n. {payTo.accountHolder}</p>}
      </div>
    );
  }
  return (
    <div className="rounded-xl bg-muted p-4">
      <p className="text-[13px] text-muted-foreground">Nomor rekening untuk pembayaran belum dibagikan.</p>
      <WhatsAppButton
        contact={contact}
        className="mt-3 h-auto min-h-11 w-full bg-card py-2 whitespace-normal sm:w-auto"
        quietClassName="mt-2"
        label="Minta nomor rekening lewat WhatsApp"
        message={askMessage(contact, `nomor rekening untuk pembayaran ${forWhat}`)}
      />
    </div>
  );
}

/** What to pay next, where, and a one-tap way to tell the WO it was paid. */
export function NextPaymentCard({ finance, payTo, contact }: { finance: NonNullable<PortalHome["finance"]>; payTo: PortalHome["payTo"]; contact: PortalContact }) {
  const term = finance.nextTerm;
  const hasBank = Boolean(payTo?.bankName && payTo.accountNumber);

  if (finance.remaining <= 0 && !term) {
    return (
      <section id="tagihan" aria-labelledby="pay-heading" className="rounded-2xl border border-border bg-card p-5 sm:p-6">
        <h2 id="pay-heading" className="text-base font-semibold text-foreground">Tagihan berikutnya</h2>
        <p className="mt-2 text-[14px] text-muted-foreground">
          {finance.contractValue > 0 ? "Seluruh tagihan Anda sudah lunas. Terima kasih." : "Belum ada tagihan yang diatur. Wedding organizer Anda akan memberi tahu begitu ada."}
        </p>
      </section>
    );
  }

  const amount = term ? term.remaining : finance.remaining;
  const label = term ? term.label : "Sisa tagihan";
  const days = term ? daysUntil(term.dueDate) : null;
  const late = term?.status === "overdue";

  return (
    <section id="tagihan" aria-labelledby="pay-heading" className="rounded-2xl border border-border bg-card p-5 sm:p-6">
      <h2 id="pay-heading" className="text-base font-semibold text-foreground">Tagihan berikutnya</h2>

      <p className="mt-3 text-[13px] text-muted-foreground">{label}</p>
      <p className="text-[32px] leading-tight font-extrabold tracking-[-0.02em] text-foreground tabular-nums">{formatCurrency(amount)}</p>
      {term && (
        <div className="mt-1 flex flex-wrap items-center gap-x-3 gap-y-1.5 text-[14px] text-muted-foreground">
          <span>
            Jatuh tempo {formatDate(term.dueDate)}
            {!late && term.status !== "paid" && days !== null && days >= 0 && ` · ${days === 0 ? "hari ini" : `${days} hari lagi`}`}
          </span>
          {/* Amber, not red: red is reserved for the "you need to act" list. */}
          {(late || term.status === "partial") && <Badge tone="warning">{TERM_STATUS_LABELS.client[term.status]}</Badge>}
        </div>
      )}

      <div className="mt-5">
        <BankAccount payTo={payTo} contact={contact} forWhat={label} />
      </div>

      {hasBank && (
        <WhatsAppButton
          contact={contact}
          tone="solid"
          size="lg"
          className="mt-4 w-full"
          label="Konfirmasi lewat WhatsApp"
          quietWhenMissing={false}
          message={paymentConfirmationMessage({
            businessName: contact.businessName || contact.name,
            clientName: contact.clientName,
            projectName: contact.projectName,
            termLabel: label,
            amount,
            bankName: payTo?.bankName,
            bankAccountNumber: payTo?.accountNumber,
          })}
        />
      )}
    </section>
  );
}
