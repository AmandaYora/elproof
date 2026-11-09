import { Copy, MessageCircle } from "lucide-react";
import { toast } from "@/shared/stores/useToastStore";
import { TERM_STATUS_LABELS } from "@/shared/lib/labels";
import { formatCurrency, formatDate } from "@/shared/lib/formatters";
import { whatsappUrl } from "@/shared/lib/access-link";
import { Button } from "@/shared/components/kit/button";
import { daysUntil } from "@/modules/projects/lib/dates";
import { paymentConfirmationMessage } from "@/modules/client-portal/lib/payment-message";
import type { PortalHome } from "@/modules/client-portal/types";

interface Props {
  finance: NonNullable<PortalHome["finance"]>;
  payTo: PortalHome["payTo"];
  projectName: string;
  clientName: string;
  businessName: string;
  /** The PIC's number if usable, else the business's; undefined when neither is. */
  contactPhone: string | undefined;
}

async function copy(text: string) {
  try {
    await navigator.clipboard.writeText(text);
    toast.success("Nomor rekening disalin");
  } catch {
    toast.error("Belum bisa menyalin. Tekan lama pada nomor untuk menyalinnya.");
  }
}

/** What to pay next, where, and a one-tap way to tell the WO it was paid. */
export function NextPaymentCard({ finance, payTo, projectName, clientName, businessName, contactPhone }: Props) {
  const term = finance.nextTerm;
  const hasBank = Boolean(payTo?.bankName && payTo.accountNumber);

  if (finance.remaining <= 0 && !term) {
    return (
      <section id="tagihan" aria-labelledby="pay-heading" className="rounded-2xl border border-border bg-card p-5 sm:p-6">
        <h2 id="pay-heading" className="text-[15px] font-semibold text-foreground">Tagihan berikutnya</h2>
        <p className="mt-2 text-[14px] text-muted-foreground">
          {finance.contractValue > 0 ? "Seluruh tagihan Anda sudah lunas. Terima kasih." : "Belum ada tagihan yang diatur. Wedding organizer Anda akan memberi tahu begitu ada."}
        </p>
      </section>
    );
  }

  const amount = term ? term.remaining : finance.remaining;
  const label = term ? term.label : "Sisa tagihan";
  const days = term ? daysUntil(term.dueDate) : null;

  return (
    <section id="tagihan" aria-labelledby="pay-heading" className="rounded-2xl border border-border bg-card p-5 sm:p-6">
      <h2 id="pay-heading" className="text-[15px] font-semibold text-foreground">Tagihan berikutnya</h2>

      <p className="mt-3 text-[13px] text-muted-foreground">{label}</p>
      <p className="font-display text-[32px] leading-tight font-semibold text-navy-950 tabular-nums">{formatCurrency(amount)}</p>
      {term && (
        <p className="mt-1 text-[13.5px] text-muted-foreground">
          Jatuh tempo {formatDate(term.dueDate)}
          {term.status !== "paid" && days !== null && days >= 0 && ` · ${days === 0 ? "hari ini" : `${days} hari lagi`}`}
        </p>
      )}

      {term && term.status !== "upcoming" && term.status !== "paid" && (
        // Amber, not red: red is reserved for the single "you need to act" card.
        <p className="mt-2 text-[13.5px] font-medium text-warning-text">{TERM_STATUS_LABELS.client[term.status]}</p>
      )}

      <div className="mt-5 rounded-xl bg-muted p-4">
        {hasBank ? (
          <>
            <p className="text-[12.5px] font-medium text-muted-foreground">Transfer ke</p>
            <p className="mt-0.5 text-[14px] font-semibold text-foreground">{payTo!.bankName}</p>
            <div className="mt-1 flex items-center justify-between gap-3">
              <p className="font-mono text-[18px] font-semibold tracking-wide text-foreground tabular-nums select-all">{payTo!.accountNumber}</p>
              <Button variant="outline" onClick={() => void copy(payTo!.accountNumber)} className="shrink-0 bg-card">
                <Copy aria-hidden /> Salin
              </Button>
            </div>
            {payTo!.accountHolder && <p className="mt-1 text-[13px] text-muted-foreground">a.n. {payTo!.accountHolder}</p>}
          </>
        ) : (
          <p className="text-[13.5px] text-muted-foreground">Rekening pembayaran belum dibagikan. Tanyakan lewat WhatsApp ke wedding organizer Anda.</p>
        )}
      </div>

      {contactPhone && (
        <Button
          size="lg"
          className="mt-4 w-full"
          nativeButton={false}
          render={
            <a
              href={whatsappUrl(
                contactPhone,
                hasBank
                  ? paymentConfirmationMessage({
                      businessName,
                      clientName,
                      projectName,
                      termLabel: label,
                      amount,
                      bankName: payTo?.bankName,
                      bankAccountNumber: payTo?.accountNumber,
                    })
                  : `Halo ${businessName}, saya ${clientName} (${projectName}). Mohon info rekening untuk pembayaran ${label}.`,
              )}
              target="_blank"
              rel="noopener noreferrer"
            />
          }
        >
          <MessageCircle aria-hidden />
          {hasBank ? "Konfirmasi pembayaran via WhatsApp" : "Tanya rekening via WhatsApp"}
        </Button>
      )}
    </section>
  );
}
