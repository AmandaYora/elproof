import { Copy, MessageCircle } from "lucide-react";
import { toast } from "@/shared/stores/useToastStore";
import { TERM_STATUS_LABELS } from "@/shared/lib/labels";
import { formatCurrency, formatDate } from "@/shared/lib/formatters";
import { whatsappUrl } from "@/shared/lib/access-link";
import { daysUntil } from "@/modules/projects/lib/dates";
import { paymentConfirmationMessage } from "@/modules/client-portal/lib/payment-message";
import type { TenantContact } from "@/shared/stores/useContactStore";
import type { PaymentTerm } from "@/modules/projects/types";

interface Props {
  projectName: string;
  clientName: string;
  term: PaymentTerm | undefined;
  /** What is still owed against the contract (may be 0). */
  remaining: number;
  contact: TenantContact | null;
  /** The PIC's number, if shared; the business number is the fallback. */
  contactPhone: string | undefined;
  /** The contact details could not be loaded (not the same as "none shared"). */
  contactFailed?: boolean;
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
export function NextPaymentCard({ projectName, clientName, term, remaining, contact, contactPhone, contactFailed }: Props) {
  if (remaining <= 0 && !term) {
    return (
      <section id="tagihan" aria-labelledby="pay-heading" className="rounded-2xl border border-border bg-white p-5 sm:p-6">
        <h2 id="pay-heading" className="text-[15px] font-semibold text-text-primary">Tagihan berikutnya</h2>
        <p className="mt-2 text-[14px] text-text-secondary">Seluruh tagihan Anda sudah lunas. Terima kasih.</p>
      </section>
    );
  }

  const amount = term ? term.remaining : remaining;
  const label = term ? term.label : "Sisa tagihan";
  const waPhone = contactPhone;
  const hasBank = Boolean(contact?.bankName && contact.bankAccountNumber);

  return (
    <section id="tagihan" aria-labelledby="pay-heading" className="rounded-2xl border border-border bg-white p-5 sm:p-6">
      <h2 id="pay-heading" className="text-[15px] font-semibold text-text-primary">Tagihan berikutnya</h2>

      <p className="mt-3 text-[13px] text-text-secondary">{label}</p>
      <p className="font-display text-[32px] font-semibold leading-tight tabular-nums text-navy-950">{formatCurrency(amount)}</p>
      {term && (
        <p className="mt-1 text-[13.5px] text-text-secondary">
          Jatuh tempo {formatDate(term.dueDate)}
          {term.status !== "paid" && daysUntil(term.dueDate) >= 0 && ` · ${daysUntil(term.dueDate) === 0 ? "hari ini" : `${daysUntil(term.dueDate)} hari lagi`}`}
        </p>
      )}

      {term && term.status !== "upcoming" && term.status !== "paid" && (
        // Amber, not red: red is reserved for the single "you need to act" card.
        <p className="mt-2 text-[13.5px] font-medium text-warning-strong">{TERM_STATUS_LABELS.client[term.status]}</p>
      )}

      <div className="mt-5 rounded-xl bg-surface-muted p-4">
        {hasBank ? (
          <>
            <p className="text-[12.5px] font-medium text-text-secondary">Transfer ke</p>
            <p className="mt-0.5 text-[14px] font-semibold text-text-primary">{contact!.bankName}</p>
            <div className="mt-1 flex items-center justify-between gap-3">
              <p className="select-all font-mono text-[18px] font-semibold tracking-wide text-text-primary tabular-nums">{contact!.bankAccountNumber}</p>
              <button
                type="button"
                onClick={() => void copy(contact!.bankAccountNumber!)}
                className="flex min-h-11 shrink-0 items-center gap-1.5 rounded-lg border border-border bg-white px-3 text-[13px] font-semibold text-navy-900 hover:bg-surface-muted focus-visible:outline focus-visible:outline-2 focus-visible:outline-navy-900"
              >
                <Copy className="h-4 w-4" aria-hidden /> Salin
              </button>
            </div>
            {contact!.bankAccountHolder && <p className="mt-1 text-[13px] text-text-secondary">a.n. {contact!.bankAccountHolder}</p>}
          </>
        ) : (
          <p className="text-[13.5px] text-text-secondary">
            {contactFailed
              ? "Rekening pembayaran belum bisa dimuat. Muat ulang halaman ini, atau tanyakan lewat WhatsApp ke wedding organizer Anda."
              : "Rekening pembayaran belum dibagikan. Tanyakan lewat WhatsApp ke wedding organizer Anda."}
          </p>
        )}
      </div>

      {waPhone && (
        <a
          href={whatsappUrl(
            waPhone,
            hasBank
              ? paymentConfirmationMessage({
                  businessName: contact?.businessName ?? "",
                  clientName,
                  projectName,
                  termLabel: label,
                  amount,
                  bankName: contact?.bankName,
                  bankAccountNumber: contact?.bankAccountNumber,
                })
              : `Halo ${contact?.businessName ?? ""}, saya ${clientName} (${projectName}). Mohon info rekening untuk pembayaran ${label}.`,
          )}
          target="_blank"
          rel="noopener noreferrer"
          className="mt-4 flex min-h-12 w-full items-center justify-center gap-2 rounded-xl bg-navy-900 px-4 text-[14px] font-semibold text-white hover:bg-navy-800 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-navy-900"
        >
          <MessageCircle className="h-4 w-4" aria-hidden />
          {hasBank ? "Konfirmasi pembayaran via WhatsApp" : "Tanya rekening via WhatsApp"}
        </a>
      )}
    </section>
  );
}
